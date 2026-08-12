package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"srl-tui/pkg/ndk"
	"srl-tui/pkg/tui/theme"
)

func GetFilteredBGP(snap *ndk.TelemetryState, searchQuery string) []ndk.BGPPeerState {
	var activeBgp []ndk.BGPPeerState
	qLower := strings.ToLower(searchQuery)
	for _, bgp := range snap.BGPPeers {
		if !strings.HasPrefix(bgp.Interface, "mgmt") && !strings.HasPrefix(bgp.Interface, "mgmt0") {
			if qLower != "" {
				afStr := strings.Join(bgp.AddrFamilies, " ")
				match := strings.Contains(strings.ToLower(bgp.NeighborIP), qLower) ||
					strings.Contains(fmt.Sprintf("%d", bgp.PeerASN), qLower) ||
					strings.Contains(strings.ToLower(bgp.Interface), qLower) ||
					strings.Contains(strings.ToLower(bgp.SessionState), qLower) ||
					strings.Contains(strings.ToLower(bgp.PeerType), qLower) ||
					strings.Contains(strings.ToLower(afStr), qLower)
				if !match {
					continue
				}
			}
			activeBgp = append(activeBgp, bgp)
		}
	}
	return activeBgp
}

func RenderTopoMesh(snap *ndk.TelemetryState, focused bool, pal theme.Palette, width, height int, searchQuery string, searchActive bool, searchInputView string, selectedIdx int) string {
	if width < 50 {
		width = 50
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(pal.Primary).Background(pal.Background)
	subStyle := lipgloss.NewStyle().Foreground(pal.Subtext).Background(pal.Background)

	// Filter BGP peers
	activeBgp := GetFilteredBGP(snap, searchQuery)

	// Build top neighbor node boxes dynamically for ALL active BGP routing adjacencies
	var topBoxStrings []string
	seenNodes := make(map[string]bool)

	for _, bgp := range activeBgp {
		nodeName := bgp.NeighborIP
		if !seenNodes[nodeName] {
			seenNodes[nodeName] = true

			stateColor := pal.Success
			stateUpper := strings.ToUpper(bgp.SessionState)
			if bgp.InMaintenance {
				stateColor = pal.Warning
			} else if stateUpper == "CONNECT" || stateUpper == "ACTIVE" || stateUpper == "CONNECTING" {
				stateColor = pal.Warning
			} else if stateUpper != "ESTABLISHED" {
				stateColor = pal.Error
			}

			boxLabel := fmt.Sprintf("%s\nAS%d [%s]", bgp.NeighborIP, bgp.PeerASN, stateUpper)
			if bgp.InMaintenance {
				boxLabel = fmt.Sprintf("🚧 %s 🚧\nAS%d [MAINT]", bgp.NeighborIP, bgp.PeerASN)
			}

			nBox := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(stateColor).
				BorderBackground(pal.Background).
				Background(pal.Background).
				Padding(0, 1).
				Align(lipgloss.Center).
				Width(24).
				Render(boxLabel)
			topBoxStrings = append(topBoxStrings, nBox)
		}
	}

	// Calculate total width of top nodes row for connector scaling
	nodeCount := len(topBoxStrings)
	totalWidth := nodeCount*26 + (nodeCount-1)*3
	if totalWidth < 60 {
		totalWidth = 60
	}

	// Local Switch Box
	localBox := lipgloss.NewStyle().
		Bold(true).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(pal.Primary).
		BorderBackground(pal.Background).
		Background(pal.Background).
		Padding(0, 2).
		Align(lipgloss.Center).
		Width(36).
		Render(fmt.Sprintf("LOCAL SWITCH: %s\n(%s)", snap.Hostname, snap.Platform))

	localCentered := lipgloss.NewStyle().Width(totalWidth).Align(lipgloss.Center).Background(pal.Background).Render(localBox)

	var diagramParts []string
	if len(topBoxStrings) > 0 {
		gapLine := lipgloss.NewStyle().Background(pal.Background).Render("   ")
		gapStr := fmt.Sprintf("%s\n%s\n%s\n%s", gapLine, gapLine, gapLine, gapLine)
		topNodesRow := lipgloss.JoinHorizontal(lipgloss.Top, interleaveStrings(topBoxStrings, gapStr)...)
		topNodesPadded := lipgloss.NewStyle().Width(totalWidth).Background(pal.Background).Render(topNodesRow)

		diagramParts = append(diagramParts, topNodesPadded, "", "", localCentered)
	} else {
		noMatchMsg := lipgloss.NewStyle().
			Foreground(pal.Muted).
			Background(pal.Background).
			Render("  (no BGP peers match filter)")
		diagramParts = append(diagramParts, noMatchMsg, "", localCentered)
	}

	rawDiagram := lipgloss.JoinVertical(lipgloss.Left, diagramParts...)
	var styledDiagramLines []string
	bgStyle := lipgloss.NewStyle().Background(pal.Background)

	for _, line := range strings.Split(rawDiagram, "\n") {
		styledDiagramLines = append(styledDiagramLines, bgStyle.Render(line))
	}
	diagramStr := strings.Join(styledDiagramLines, "\n")

	// BGP Session Table
	colHdrs := fmt.Sprintf("   %-19s%-11s%-11s%-17s%-17s%-19s%-11s%s",
		"NEIGHBOR IP", "PEER AS", "TYPE", "STATE", "INTERFACE", "ACTIVE AFs", "UPTIME", "PREFIXES (Rx/Tx)",
	)

	var bgpRows []string
	bgpRows = append(bgpRows, lipgloss.NewStyle().Bold(true).Foreground(pal.Primary).Background(pal.Background).Render(colHdrs))
	bgpRows = append(bgpRows, lipgloss.NewStyle().Foreground(pal.Muted).Background(pal.Background).Render("  ────────────────── ────────── ────────── ──────────────── ──────────────── ────────────────── ────────── ────────────────"))

	if len(activeBgp) == 0 {
		bgpRows = append(bgpRows, lipgloss.NewStyle().Foreground(pal.Subtext).Background(pal.Background).Render("  No matching BGP routing protocol adjacencies discovered"))
	} else {
		for i, peer := range activeBgp {
			stateUpper := strings.ToUpper(peer.SessionState)
			stateText := "● " + stateUpper
			stateStyle := lipgloss.NewStyle().Foreground(pal.Success).Background(pal.Background)

			if peer.InMaintenance {
				stateText = "🚧 MAINT MODE"
				stateStyle = lipgloss.NewStyle().Foreground(pal.Warning).Background(pal.Background).Bold(true)
			} else if stateUpper == "ACTIVE" || stateUpper == "CONNECT" || stateUpper == "CONNECTING" {
				stateText = "◯ " + stateUpper
				stateStyle = lipgloss.NewStyle().Foreground(pal.Warning).Background(pal.Background)
			} else if stateUpper != "ESTABLISHED" {
				stateText = "✕ " + stateUpper
				stateStyle = lipgloss.NewStyle().Foreground(pal.Error).Background(pal.Background)
			}

			ipDisplay := peer.NeighborIP
			if peer.InMaintenance {
				ipDisplay = fmt.Sprintf("🚧 %s", peer.NeighborIP)
			}

			afDisplay := strings.Join(peer.AddrFamilies, ", ")
			if afDisplay == "" {
				afDisplay = "ipv4-unicast"
			}

			ipStr := lipgloss.NewStyle().Foreground(pal.Text).Background(pal.Background).Render(padRight(ipDisplay, 19))
			asStr := lipgloss.NewStyle().Foreground(pal.Secondary).Background(pal.Background).Render(padRight(fmt.Sprintf("%d", peer.PeerASN), 11))
			typeStr := lipgloss.NewStyle().Foreground(pal.Primary).Background(pal.Background).Render(padRight(peer.PeerType, 11))
			stStr := stateStyle.Render(padRight(stateText, 17))
			intfStr := lipgloss.NewStyle().Foreground(pal.Warning).Background(pal.Background).Render(padRight(peer.Interface, 17))
			afStr := lipgloss.NewStyle().Foreground(pal.Primary).Background(pal.Background).Render(padRight(afDisplay, 19))
			upStr := lipgloss.NewStyle().Foreground(pal.Subtext).Background(pal.Background).Render(padRight(peer.Uptime, 11))
			var pfxParts []string
			if len(peer.AddrFamilies) > 0 {
				for _, af := range peer.AddrFamilies {
					if st, ok := peer.AFStats[af]; ok {
						pfxParts = append(pfxParts, fmt.Sprintf("%d/%d", st.RxPrefixes, st.TxPrefixes))
					} else {
						pfxParts = append(pfxParts, fmt.Sprintf("%d/%d", peer.RxPrefixes, peer.TxPrefixes))
					}
				}
			}
			pfxTxt := strings.Join(pfxParts, ", ")
			if pfxTxt == "" {
				pfxTxt = fmt.Sprintf("%d/%d", peer.RxPrefixes, peer.TxPrefixes)
			}
			pfxStr := lipgloss.NewStyle().Foreground(pal.Text).Background(pal.Background).Render(pfxTxt)

			prefixPointer := "  "
			if i == selectedIdx {
				prefixPointer = "> "
			}

			var row string
			if i == selectedIdx {
				rowStr := fmt.Sprintf("%s%-18s %-10d %-10s %-16s %-16s %-18s %-10s %s", prefixPointer, ipDisplay, peer.PeerASN, peer.PeerType, stateText, peer.Interface, afDisplay, peer.Uptime, pfxTxt)
				row = lipgloss.NewStyle().Foreground(pal.Background).Background(pal.Highlight).Bold(true).Render(rowStr)
			} else {
				row = lipgloss.NewStyle().Background(pal.Background).Render(fmt.Sprintf("%s%s%s%s%s%s%s%s%s", prefixPointer, ipStr, asStr, typeStr, stStr, intfStr, afStr, upStr, pfxStr))
			}
			bgpRows = append(bgpRows, row)
		}
	}

	tableStr := strings.Join(bgpRows, "\n")

	borderColor := pal.Muted
	if focused {
		borderColor = pal.Secondary
	}

	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
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
		titleStyle.Render("🌐 BGP ROUTING PROTOCOL TOPOLOGY & MESH"),
		subStyle.Render("[Real-time BGP routing adjacencies & session states • Arrow Up/Down to select, Enter for details]"),
		searchBarStr,
	))

	return panelStyle.Render(header + "\n\n" + diagramStr + "\n\n" + tableStr)
}

func interleaveStrings(items []string, gap string) []string {
	if len(items) == 0 {
		return nil
	}
	var res []string
	for i, item := range items {
		res = append(res, item)
		if i < len(items)-1 {
			res = append(res, gap)
		}
	}
	return res
}
