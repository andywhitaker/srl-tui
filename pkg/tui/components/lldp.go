package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"srl-tui/pkg/ndk"
	"srl-tui/pkg/tui/theme"
)

type LLDPView struct {
	SelectedIdx  int
	ScrollOffset int
}

func NewLLDPView() *LLDPView {
	return &LLDPView{
		SelectedIdx:  0,
		ScrollOffset: 0,
	}
}

func (v *LLDPView) ScrollUp() {
	if v.SelectedIdx > 0 {
		v.SelectedIdx--
	}
}

func (v *LLDPView) ScrollDown(total int) {
	if total > 0 && v.SelectedIdx < total-1 {
		v.SelectedIdx++
	}
}

func GetFilteredLLDP(snap *ndk.TelemetryState, searchQuery string) []ndk.LLDPNeighbor {
	var filtered []ndk.LLDPNeighbor
	qLower := strings.ToLower(searchQuery)

	for _, n := range snap.LLDPNeighbors {
		if qLower != "" {
			match := strings.Contains(strings.ToLower(n.LocalPort), qLower) ||
				strings.Contains(strings.ToLower(n.SysName), qLower) ||
				strings.Contains(strings.ToLower(n.RemotePort), qLower) ||
				strings.Contains(strings.ToLower(n.MgmtIP), qLower) ||
				strings.Contains(strings.ToLower(n.Capabilities), qLower) ||
				strings.Contains(strings.ToLower(n.ChassisID), qLower)
			if !match {
				continue
			}
		}
		filtered = append(filtered, n)
	}
	return filtered
}

func (v *LLDPView) Render(snap *ndk.TelemetryState, pal theme.Palette, width, height int, searchQuery string, searchActive bool, searchInputView string) string {
	if width < 50 {
		width = 50
	}
	if height < 10 {
		height = 10
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(pal.Primary)
	subStyle := lipgloss.NewStyle().Foreground(pal.Subtext)

	filtered := GetFilteredLLDP(snap, searchQuery)
	totalCount := len(filtered)

	// Clamp SelectedIdx
	if v.SelectedIdx >= totalCount {
		v.SelectedIdx = totalCount - 1
	}
	if v.SelectedIdx < 0 {
		v.SelectedIdx = 0
	}

	summaryStr := lipgloss.NewStyle().Foreground(pal.Text).Render(
		fmt.Sprintf("Total Discovered LLDP Neighbors: %d  |  Protocol: 802.1AB  |  Holdtimer: Active", totalCount),
	)

	// Dynamic scroll window calculation
	visibleRowCount := height - 9
	if visibleRowCount < 3 {
		visibleRowCount = 3
	}

	if v.SelectedIdx >= v.ScrollOffset+visibleRowCount {
		v.ScrollOffset = v.SelectedIdx - visibleRowCount + 1
	}
	if v.SelectedIdx < v.ScrollOffset {
		v.ScrollOffset = v.SelectedIdx
	}

	endIdx := v.ScrollOffset + visibleRowCount
	if endIdx > totalCount {
		endIdx = totalCount
	}

	// Format plain text headers with exact column widths:
	// CURSOR (2) + LOCAL PORT (16) + REMOTE SYSTEM NAME (24) + REMOTE PORT (18) + MANAGEMENT IP (16) + CAPABILITIES (16)
	hdrTxt := fmt.Sprintf("  %s %s %s %s %s",
		padRight("LOCAL PORT", 16),
		padRight("REMOTE SYSTEM NAME", 24),
		padRight("REMOTE PORT", 18),
		padRight("MANAGEMENT IP", 16),
		padRight("CAPABILITIES", 16),
	)
	sepTxt := fmt.Sprintf("  %s %s %s %s %s",
		padRight("────────────────", 16),
		padRight("────────────────────────", 24),
		padRight("──────────────────", 18),
		padRight("────────────────", 16),
		padRight("────────────────", 16),
	)

	var rows []string
	rows = append(rows, lipgloss.NewStyle().Bold(true).Foreground(pal.Primary).Render(hdrTxt))
	rows = append(rows, lipgloss.NewStyle().Foreground(pal.Muted).Render(sepTxt))

	if totalCount == 0 {
		rows = append(rows, lipgloss.NewStyle().Foreground(pal.Subtext).Render("  No LLDP neighbors discovered matching search filter"))
	} else {
		for i := v.ScrollOffset; i < endIdx; i++ {
			n := filtered[i]
			cursor := "  "
			if i == v.SelectedIdx {
				cursor = "► "
			}

			mgmtIP := n.MgmtIP
			if mgmtIP == "" {
				mgmtIP = "(unknown)"
			}
			caps := n.Capabilities
			if caps == "" {
				caps = "ROUTER / SWITCH"
			}

			sLocal := padRight(n.LocalPort, 16)
			sSys := padRight(n.SysName, 24)
			sRemote := padRight(n.RemotePort, 18)
			sMgmt := padRight(mgmtIP, 16)
			sCaps := padRight(caps, 16)

			fLocal := lipgloss.NewStyle().Foreground(pal.Text).Render(sLocal)
			fSys := lipgloss.NewStyle().Foreground(pal.Secondary).Bold(true).Render(sSys)
			fRemote := lipgloss.NewStyle().Foreground(pal.Success).Render(sRemote)
			fMgmt := lipgloss.NewStyle().Foreground(pal.Warning).Render(sMgmt)
			fCaps := lipgloss.NewStyle().Foreground(pal.Subtext).Render(sCaps)

			row := fmt.Sprintf("%s%s %s %s %s %s", cursor, fLocal, fSys, fRemote, fMgmt, fCaps)
			if i == v.SelectedIdx {
				row = lipgloss.NewStyle().Background(pal.Highlight).Render(row)
			}
			rows = append(rows, row)
		}
	}

	// Footer scroll status line
	scrollStatus := lipgloss.NewStyle().Foreground(pal.Secondary).Bold(true).Render(
		fmt.Sprintf(" [LLDP Neighbor %d/%d]  [↑/↓ to scroll, ENTER for details] ", v.SelectedIdx+1, totalCount),
	)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(pal.Primary).
		Background(pal.Surface).
		Width(width - 2).
		Height(height - 2).
		Padding(0, 1)

	searchBarStr := ""
	if searchActive {
		searchBarStr = lipgloss.NewStyle().Foreground(pal.Warning).Bold(true).Render(fmt.Sprintf("\n🔍 SEARCH: %s [Esc to exit]", searchInputView))
	} else if searchQuery != "" {
		searchBarStr = lipgloss.NewStyle().Foreground(pal.Secondary).Render(fmt.Sprintf("  [Filtered by: '%s' - press / to edit, Esc to clear]", searchQuery))
	}

	header := fmt.Sprintf("%s %s%s",
		titleStyle.Render("🤝 LLDP NEIGHBORS (Link Layer Discovery Protocol / IEEE 802.1AB)"),
		subStyle.Render("[Discovered Physical Peer Topology]"),
		searchBarStr,
	)

	return boxStyle.Render(header + "\n" + summaryStr + "\n\n" + strings.Join(rows, "\n") + "\n" + scrollStatus)
}

func RenderLLDPDetailModal(n ndk.LLDPNeighbor, snap *ndk.TelemetryState, pal theme.Palette, width, height int) string {
	modalWidth := 74
	if width < modalWidth {
		modalWidth = width - 4
	}

	statusBadge := lipgloss.NewStyle().Bold(true).Foreground(pal.Background).Background(pal.Success).Padding(0, 1).Render("● ADJACENCY UP / LINK DISCOVERY ACTIVE")

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(pal.Secondary)
	lblStyle := lipgloss.NewStyle().Bold(true).Foreground(pal.Secondary)
	valStyle := lipgloss.NewStyle().Foreground(pal.Text)

	mgmtIP := n.MgmtIP
	if mgmtIP == "" {
		mgmtIP = "(unknown)"
	}
	sysDesc := n.SysDesc
	if sysDesc == "" {
		sysDesc = "Not Provided by Neighbor"
	}
	chassis := n.ChassisID
	if chassis == "" {
		chassis = "N/A"
	}
	caps := n.Capabilities
	if caps == "" {
		caps = "N/A"
	}

	lines := []string{
		titleStyle.Render("🤝 LLDP DISCOVERY NEIGHBOR DETAILS - " + n.SysName),
		statusBadge,
		lipgloss.NewStyle().Foreground(pal.Muted).Render(strings.Repeat("─", modalWidth-4)),
		fmt.Sprintf("  %-24s : %s", lblStyle.Render("Local Physical Port"), valStyle.Render(n.LocalPort)),
		fmt.Sprintf("  %-24s : %s", lblStyle.Render("Remote System Name"), valStyle.Render(n.SysName)),
		fmt.Sprintf("  %-24s : %s", lblStyle.Render("Remote Port ID"), valStyle.Render(n.RemotePort)),
		fmt.Sprintf("  %-24s : %s", lblStyle.Render("Remote Management IP"), valStyle.Render(mgmtIP)),
		fmt.Sprintf("  %-24s : %s", lblStyle.Render("Chassis ID"), valStyle.Render(chassis)),
		fmt.Sprintf("  %-24s : %s", lblStyle.Render("Enabled Capabilities"), valStyle.Render(caps)),
		fmt.Sprintf("  %-24s : %s", lblStyle.Render("Remote System Description"), valStyle.Render(sysDesc)),
		fmt.Sprintf("  %-24s : %s", lblStyle.Render("Discovery Specification"), valStyle.Render("IEEE 802.1AB (LLDP)")),
		"",
		lipgloss.NewStyle().Foreground(pal.Muted).Render("Press ESC or ENTER to close detail window"),
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(pal.Secondary).
		Background(pal.Surface).
		Padding(1, 2).
		Width(modalWidth)

	return boxStyle.Render(strings.Join(lines, "\n"))
}
