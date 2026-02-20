package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sergi/go-diff/diffmatchpatch"
)

var (
	// Palette
	gold   = lipgloss.Color("#FFD700")
	silver = lipgloss.Color("#CCCCCC")
	grey   = lipgloss.Color("#555555")
	red    = lipgloss.Color("#FF5F5F")
	green  = lipgloss.Color("#5FFF5F")

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(gold).
			Padding(0, 1).
			Bold(true)

	headerStyle = lipgloss.NewStyle().Foreground(gold)
	footerStyle = lipgloss.NewStyle().Foreground(grey)
	sepStyle    = lipgloss.NewStyle().Foreground(gold).Bold(true)
	numStyle    = lipgloss.NewStyle().Foreground(grey)

	addedStyle   = lipgloss.NewStyle().Foreground(green)
	removedStyle = lipgloss.NewStyle().Foreground(red)
	baseStyle    = lipgloss.NewStyle().Foreground(silver)
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#333333"))
)

type diffLine struct {
	lNum string
	left string
	rNum string
	right string
}

type model struct {
	file1    string
	file2    string
	content1 string
	content2 string
	viewport viewport.Model
	ready    bool
	width    int
	height   int
	added    int
	deleted  int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerHeight := 3
		footerHeight := 1
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-headerHeight-footerHeight)
			m.viewport.YPosition = headerHeight
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height-headerHeight-footerHeight
		}
		m.viewport.SetContent(m.renderDiff())
	}
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *model) renderDiff() string {
	dmp := diffmatchpatch.New()
	
	prep := func(s string) string {
		s = strings.ReplaceAll(s, "\t", "    ")
		s = strings.ReplaceAll(s, "\r", "")
		return s
	}

	a, b, c := dmp.DiffLinesToChars(prep(m.content1), prep(m.content2))
	diffs := dmp.DiffMain(a, b, false)
	diffs = dmp.DiffCharsToLines(diffs, c)
	diffs = dmp.DiffCleanupSemantic(diffs)

	var lines []diffLine
	lIdx, rIdx := 1, 1
	m.added, m.deleted = 0, 0

	for _, d := range diffs {
		text := strings.TrimSuffix(d.Text, "\n")
		parts := strings.Split(text, "\n")
		for _, p := range parts {
			switch d.Type {
			case diffmatchpatch.DiffEqual:
				lines = append(lines, diffLine{fmt.Sprintf("%d", lIdx), baseStyle.Render(p), fmt.Sprintf("%d", rIdx), baseStyle.Render(p)})
				lIdx++; rIdx++
			case diffmatchpatch.DiffDelete:
				lines = append(lines, diffLine{fmt.Sprintf("%d", lIdx), removedStyle.Render(p), "", dimStyle.Render("~")})
				lIdx++; m.deleted++
			case diffmatchpatch.DiffInsert:
				lines = append(lines, diffLine{"", dimStyle.Render("~"), fmt.Sprintf("%d", rIdx), addedStyle.Render(p)})
				rIdx++; m.added++
			}
		}
	}

	if m.added == 0 && m.deleted == 0 {
		return "\n  " + lipgloss.NewStyle().Foreground(green).Render("✔ No changes detected between files.")
	}

	safeWidth := m.width - 2
	if safeWidth < 30 { safeWidth = 30 }
	paneWidth := (safeWidth - 15) / 2

	var bld strings.Builder
	for _, dl := range lines {
		padNum := func(n string) string {
			if len(n) > 5 { return n[:5] }
			return strings.Repeat(" ", 5-len(n)) + n
		}

		truncate := func(s string, w int) string {
			w_actual := lipgloss.Width(s)
			if w_actual <= w {
				return s + strings.Repeat(" ", w-w_actual)
			}
			return s[:w-1] + "…"
		}

		lPart := numStyle.Render(padNum(dl.lNum)) + " " + truncate(dl.left, paneWidth)
		rPart := numStyle.Render(padNum(dl.rNum)) + " " + truncate(dl.right, paneWidth)
		
		bld.WriteString(fmt.Sprintf("%s %s %s\n", lPart, sepStyle.Render("┃"), rPart))
	}
	return bld.String()
}

func (m model) View() string {
	if !m.ready { return "\n  Initializing Atlas Diff..." }
	header := titleStyle.Render(" ATLAS.DIFF ") + " " + headerStyle.Render(fmt.Sprintf("%s <-> %s", m.file1, m.file2))
	
	stats := fmt.Sprintf(" %s %s ", 
		addedStyle.Render(fmt.Sprintf("+%d", m.added)), 
		removedStyle.Render(fmt.Sprintf("-%d", m.deleted)))
	
	footer := footerStyle.Render(fmt.Sprintf(" %3.f%% |", m.viewport.ScrollPercent()*100)) + 
		stats + 
		footerStyle.Render("| [q] Quit")
		
	return fmt.Sprintf("%s\n\n%s\n%s", header, m.viewport.View(), footer)
}

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		fmt.Println("atlas.diff v0.1.0")
		return
	}

	if len(os.Args) < 3 {
		fmt.Println("Usage: atlas.diff <file1> <file2>")
		return
	}
	f1, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Printf("Error reading %s: %v\n", os.Args[1], err)
		return
	}
	f2, err := os.ReadFile(os.Args[2])
	if err != nil {
		fmt.Printf("Error reading %s: %v\n", os.Args[2], err)
		return
	}
	
	m := model{
		file1:    os.Args[1],
		file2:    os.Args[2],
		content1: string(f1),
		content2: string(f2),
	}
	
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
