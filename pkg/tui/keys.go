package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"srl-tui/pkg/tui/theme"
)

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func RenderHelpBar(pal theme.Palette, width int) string {
	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(pal.Background).
		Background(pal.Primary).
		Padding(0, 1)

	descStyle := lipgloss.NewStyle().
		Foreground(pal.Subtext).
		Background(pal.Background)

	gap := lipgloss.NewStyle().Background(pal.Background).Render("  ")
	items := []string{
		keyStyle.Render("TAB / 1-6") + descStyle.Render(" Switch Tabs"),
		keyStyle.Render("←↑↓→ / hjkl") + descStyle.Render(" Navigate Grid / Scroll"),
		keyStyle.Render("/") + descStyle.Render(" Search Filter"),
		keyStyle.Render("u") + descStyle.Render(" Unimported"),
		keyStyle.Render("c") + descStyle.Render(" Theme"),
		keyStyle.Render("q") + descStyle.Render(" Quit"),
	}

	helpLine := lipgloss.JoinHorizontal(lipgloss.Top,
		items[0], gap, items[1], gap, items[2], gap, items[3], gap, items[4], gap, items[5],
	)

	boxStyle := lipgloss.NewStyle().
		Background(pal.Background).
		Width(width - 2).
		Padding(0, 1)

	return boxStyle.Render(helpLine)
}

func RenderHelpOverlay(pal theme.Palette, width, height int) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(pal.Background).
		Background(pal.Primary).
		Padding(0, 1)

	fmtRow := func(k, v string) string {
		kStr := lipgloss.NewStyle().Bold(true).Foreground(pal.Secondary).Background(pal.Background).Render(padRight(k, 18))
		gap := lipgloss.NewStyle().Foreground(pal.Muted).Background(pal.Background).Render(" : ")
		vStr := lipgloss.NewStyle().Foreground(pal.Text).Background(pal.Background).Render(v)
		return lipgloss.NewStyle().Background(pal.Background).Render("  " + kStr + gap + vStr)
	}

	lines := []string{
		titleStyle.Render("⚡ SRL-NDK CYBER-DASHBOARD KEYBINDINGS"),
		"",
		fmtRow("Tab / Shift+Tab", "Cycle page tabs (Ports -> Topo -> ARP/MAC -> LLDP -> Routes -> EVPN)"),
		fmtRow("1 .. 6", "Direct jump: 1=Ports, 2=Topo, 3=ARP/MAC, 4=LLDP, 5=Routes, 6=EVPN"),
		fmtRow("h, j, k, l / ←↑↓→", "Navigate 2D port matrix, scroll tables, or cycle EVPN filters"),
		fmtRow("PgUp / PgDn / ^U ^D", "Scroll 1 page (10 lines) in YANG Inspector and all table views"),
		fmtRow("Space / a / m", "Toggle active pane focus in ARP & MAC view"),
		fmtRow("/", "Activate live search filter (Pages 2-5) or YANG Inspector (Ports)"),
		fmtRow("Enter", "Open interactive YANG Inspector (Ports) or EVPN Route Details (EVPN)"),
		fmtRow("Esc", "Exit search mode or clear active search filter"),
		fmtRow("c / t", "Cycle theme (Cyberpunk -> Synthwave -> Matrix -> Monokai -> Cobalt2 -> Solarized)"),
		fmtRow("?", "Toggle this keybindings help overlay"),
		fmtRow("q / Ctrl+C", "Quit application"),
		"",
		lipgloss.NewStyle().Foreground(pal.Muted).Background(pal.Background).Render("Press ESC or ? to close this help window"),
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(pal.Primary).
		BorderBackground(pal.Background).
		Background(pal.Background).
		Padding(1, 2)

	rawContent := strings.Join(lines, "\n")
	contentWidth := 90
	r, g, b, _ := pal.Background.RGBA()
	bgSeq := fmt.Sprintf("\x1b[48;2;%d;%d;%dm", uint8(r>>8), uint8(g>>8), uint8(b>>8))
	resetSeq := fmt.Sprintf("\x1b[0m%s", bgSeq)

	var styledLines []string
	for _, l := range strings.Split(rawContent, "\n") {
		w := lipgloss.Width(l)
		if w < contentWidth {
			l += strings.Repeat(" ", contentWidth-w)
		}
		cleaned := strings.ReplaceAll(l, "\x1b[0m", resetSeq)
		styledLines = append(styledLines, bgSeq+cleaned+"\x1b[0m")
	}
	return boxStyle.Render(strings.Join(styledLines, "\n"))
}
