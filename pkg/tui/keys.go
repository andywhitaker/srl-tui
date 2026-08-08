package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"srl-tui/pkg/tui/theme"
)

func RenderHelpBar(pal theme.Palette, width int) string {
	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(pal.Background).
		Background(pal.Primary).
		Padding(0, 1)

	descStyle := lipgloss.NewStyle().
		Foreground(pal.Subtext)

	items := []string{
		keyStyle.Render("TAB / 1-6") + " " + descStyle.Render("Switch Tabs"),
		keyStyle.Render("←↑↓→ / hjkl") + " " + descStyle.Render("Navigate Grid / Scroll"),
		keyStyle.Render("/") + " " + descStyle.Render("Search Filter"),
		keyStyle.Render("u") + " " + descStyle.Render("Unimported"),
		keyStyle.Render("c") + " " + descStyle.Render("Theme"),
		keyStyle.Render("q") + " " + descStyle.Render("Quit"),
	}

	helpLine := lipgloss.JoinHorizontal(lipgloss.Top,
		items[0], "  ", items[1], "  ", items[2], "  ", items[3], "  ", items[4], "  ", items[5],
	)

	boxStyle := lipgloss.NewStyle().
		Background(pal.Surface).
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

	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(pal.Secondary)
	valStyle := lipgloss.NewStyle().Foreground(pal.Text)

	lines := []string{
		titleStyle.Render("⚡ SRL-NDK CYBER-DASHBOARD KEYBINDINGS"),
		"",
		fmt.Sprintf("  %-18s : %s", keyStyle.Render("Tab / Shift+Tab"), valStyle.Render("Cycle page tabs (Ports -> Topo -> ARP/MAC -> LLDP -> Routes -> EVPN)")),
		fmt.Sprintf("  %-18s : %s", keyStyle.Render("1 .. 6"), valStyle.Render("Direct jump: 1=Ports, 2=Topo, 3=ARP/MAC, 4=LLDP, 5=Routes, 6=EVPN")),
		fmt.Sprintf("  %-18s : %s", keyStyle.Render("h, j, k, l / ←↑↓→"), valStyle.Render("Navigate 2D port matrix, scroll tables, or cycle EVPN filters")),
		fmt.Sprintf("  %-18s : %s", keyStyle.Render("PgUp / PgDn / ^U ^D"), valStyle.Render("Scroll 1 page (10 lines) in YANG Inspector and all table views")),
		fmt.Sprintf("  %-18s : %s", keyStyle.Render("Space / a / m"), valStyle.Render("Toggle active pane focus in ARP & MAC view")),
		fmt.Sprintf("  %-18s : %s", keyStyle.Render("/"), valStyle.Render("Activate live search filter (Pages 2-5) or YANG Inspector (Ports)")),
		fmt.Sprintf("  %-18s : %s", keyStyle.Render("Enter"), valStyle.Render("Open interactive YANG Inspector (Ports) or EVPN Route Details (EVPN)")),
		fmt.Sprintf("  %-18s : %s", keyStyle.Render("Esc"), valStyle.Render("Exit search mode or clear active search filter")),
		fmt.Sprintf("  %-18s : %s", keyStyle.Render("c / t"), valStyle.Render("Cycle theme (Cyberpunk -> Synthwave -> Matrix -> Monokai -> Cobalt2 -> Solarized)")),
		fmt.Sprintf("  %-18s : %s", keyStyle.Render("?"), valStyle.Render("Toggle this keybindings help overlay")),
		fmt.Sprintf("  %-18s : %s", keyStyle.Render("q / Ctrl+C"), valStyle.Render("Quit application")),
		"",
		lipgloss.NewStyle().Foreground(pal.Muted).Render("Press ESC or ? to close this help window"),
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(pal.Primary).
		Background(pal.Surface).
		Padding(1, 2)

	return boxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}
