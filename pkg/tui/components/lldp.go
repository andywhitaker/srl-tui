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

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(pal.Primary).Background(pal.Background)
	subStyle := lipgloss.NewStyle().Foreground(pal.Subtext).Background(pal.Background)

	filtered := GetFilteredLLDP(snap, searchQuery)
	totalCount := len(filtered)

	// Clamp SelectedIdx
	if v.SelectedIdx >= totalCount {
		v.SelectedIdx = totalCount - 1
	}
	if v.SelectedIdx < 0 {
		v.SelectedIdx = 0
	}

	summaryStr := lipgloss.NewStyle().Foreground(pal.Text).Background(pal.Background).Render(
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
	// CURSOR (2) + LOCAL PORT (17) + REMOTE SYSTEM NAME (25) + REMOTE PORT (19) + MANAGEMENT IP (17) + CAPABILITIES (16)
	hdrTxt := fmt.Sprintf("  %s%s%s%s%s",
		padRight("LOCAL PORT", 17),
		padRight("REMOTE SYSTEM NAME", 25),
		padRight("REMOTE PORT", 19),
		padRight("MANAGEMENT IP", 17),
		padRight("CAPABILITIES", 16),
	)
	sepTxt := fmt.Sprintf("  %s%s%s%s%s",
		padRight("────────────────", 17),
		padRight("────────────────────────", 25),
		padRight("──────────────────", 19),
		padRight("────────────────", 17),
		padRight("────────────────", 16),
	)

	var rows []string
	rows = append(rows, lipgloss.NewStyle().Bold(true).Foreground(pal.Primary).Background(pal.Background).Render(hdrTxt))
	rows = append(rows, lipgloss.NewStyle().Foreground(pal.Muted).Background(pal.Background).Render(sepTxt))

	if totalCount == 0 {
		rows = append(rows, lipgloss.NewStyle().Foreground(pal.Subtext).Render("  No LLDP neighbors discovered matching search filter"))
	} else {
		for i := v.ScrollOffset; i < endIdx; i++ {
			n := filtered[i]
			mgmtIP := n.MgmtIP
			if mgmtIP == "" {
				mgmtIP = "(unknown)"
			}
			caps := n.Capabilities
			if caps == "" {
				caps = "ROUTER / SWITCH"
			}

			sLocal := padRight(n.LocalPort, 17)
			sSys := padRight(n.SysName, 25)
			sRemote := padRight(n.RemotePort, 19)
			sMgmt := padRight(mgmtIP, 17)
			sCaps := padRight(caps, 16)

			var row string
			if i == v.SelectedIdx {
				rowStr := fmt.Sprintf("► %s%s%s%s%s", sLocal, sSys, sRemote, sMgmt, sCaps)
				row = lipgloss.NewStyle().Foreground(pal.Background).Background(pal.Highlight).Bold(true).Render(rowStr)
			} else {
				fLocal := lipgloss.NewStyle().Foreground(pal.Text).Background(pal.Background).Render(sLocal)
				fSys := lipgloss.NewStyle().Foreground(pal.Secondary).Background(pal.Background).Bold(true).Render(sSys)
				fRemote := lipgloss.NewStyle().Foreground(pal.Success).Background(pal.Background).Render(sRemote)
				fMgmt := lipgloss.NewStyle().Foreground(pal.Warning).Background(pal.Background).Render(sMgmt)
				fCaps := lipgloss.NewStyle().Foreground(pal.Subtext).Background(pal.Background).Render(sCaps)

				row = lipgloss.NewStyle().Background(pal.Background).Render(fmt.Sprintf("  %s%s%s%s%s", fLocal, fSys, fRemote, fMgmt, fCaps))
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
		BorderBackground(pal.Background).
		Background(pal.Background).
		Width(width - 2).
		Height(height - 2).
		Padding(0, 1)

	searchBarStr := ""
	if searchActive {
		searchBarStr = lipgloss.NewStyle().Foreground(pal.Warning).Background(pal.Background).Bold(true).Render(fmt.Sprintf("\n🔍 SEARCH: %s [Esc to exit]", searchInputView))
	} else if searchQuery != "" {
		searchBarStr = lipgloss.NewStyle().Foreground(pal.Secondary).Background(pal.Background).Render(fmt.Sprintf("  [Filtered by: '%s' - press / to edit, Esc to clear]", searchQuery))
	}

	header := lipgloss.NewStyle().Background(pal.Background).Render(fmt.Sprintf("%s %s%s",
		titleStyle.Render("🤝 LLDP NEIGHBORS (Link Layer Discovery Protocol / IEEE 802.1AB)"),
		subStyle.Render("[Discovered Physical Peer Topology]"),
		searchBarStr,
	))

	return boxStyle.Render(header + "\n" + summaryStr + "\n\n" + strings.Join(rows, "\n") + "\n" + scrollStatus)
}

func RenderLLDPDetailModal(n ndk.LLDPNeighbor, snap *ndk.TelemetryState, pal theme.Palette, width, height int) string {
	modalWidth := 74
	if width < modalWidth {
		modalWidth = width - 4
	}

	statusBadge := lipgloss.NewStyle().Bold(true).Foreground(pal.Background).Background(pal.Success).Padding(0, 1).Render("● ADJACENCY UP / LINK DISCOVERY ACTIVE")

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(pal.Secondary).Background(pal.Background)

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

	fmtRow := func(lbl string, val string) string {
		l := lipgloss.NewStyle().Foreground(pal.Subtext).Background(pal.Background).Render(padRight(lbl, 24))
		gap := lipgloss.NewStyle().Foreground(pal.Muted).Background(pal.Background).Render(" : ")
		v := lipgloss.NewStyle().Foreground(pal.Text).Background(pal.Background).Render(val)
		return lipgloss.NewStyle().Background(pal.Background).Render("  " + l + gap + v)
	}

	lines := []string{
		titleStyle.Render("🤝 LLDP DISCOVERY NEIGHBOR DETAILS - " + n.SysName),
		statusBadge,
		lipgloss.NewStyle().Foreground(pal.Muted).Background(pal.Background).Render(strings.Repeat("─", modalWidth-4)),
		fmtRow("Local Physical Port", n.LocalPort),
		fmtRow("Remote System Name", n.SysName),
		fmtRow("Remote Port ID", n.RemotePort),
		fmtRow("Remote Management IP", mgmtIP),
		fmtRow("Chassis ID", chassis),
		fmtRow("Enabled Capabilities", caps),
		fmtRow("Remote System Description", sysDesc),
		fmtRow("Discovery Specification", "IEEE 802.1AB (LLDP)"),
		"",
		lipgloss.NewStyle().Foreground(pal.Muted).Background(pal.Background).Render("Press ESC or ENTER to close detail window"),
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(pal.Secondary).
		BorderBackground(pal.Background).
		Background(pal.Background).
		Padding(1, 2).
		Width(modalWidth)

	rawContent := strings.Join(lines, "\n")
	contentWidth := modalWidth - 4
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
