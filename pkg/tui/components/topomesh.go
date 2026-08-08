package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"srl-tui/pkg/ndk"
	"srl-tui/pkg/tui/theme"
)

func RenderTopoMesh(snap *ndk.TelemetryState, focused bool, pal theme.Palette, width, height int, searchQuery string, searchActive bool, searchInputView string) string {
	if width < 50 {
		width = 50
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(pal.Primary)
	subStyle := lipgloss.NewStyle().Foreground(pal.Subtext)

	// Filter BGP peers
	var activeBgp []ndk.BGPPeerState
	qLower := strings.ToLower(searchQuery)
	for _, bgp := range snap.BGPPeers {
		if !strings.HasPrefix(bgp.Interface, "mgmt") && !strings.HasPrefix(bgp.Interface, "mgmt0") {
			if qLower != "" {
				match := strings.Contains(strings.ToLower(bgp.NeighborIP), qLower) ||
					strings.Contains(fmt.Sprintf("%d", bgp.PeerASN), qLower) ||
					strings.Contains(strings.ToLower(bgp.Interface), qLower) ||
					strings.Contains(strings.ToLower(bgp.SessionState), qLower) ||
					strings.Contains(strings.ToLower(bgp.PeerType), qLower)
				if !match {
					continue
				}
			}
			activeBgp = append(activeBgp, bgp)
		}
	}

	// Build top neighbor node boxes dynamically for ALL active BGP routing adjacencies
	var topBoxStrings []string
	seenNodes := make(map[string]bool)

	for _, bgp := range activeBgp {
		nodeName := bgp.NeighborIP
		if !seenNodes[nodeName] {
			seenNodes[nodeName] = true

			stateColor := pal.Success
			stateUpper := strings.ToUpper(bgp.SessionState)
			if stateUpper == "CONNECT" || stateUpper == "ACTIVE" || stateUpper == "CONNECTING" {
				stateColor = pal.Warning
			} else if stateUpper != "ESTABLISHED" {
				stateColor = pal.Error
			}

			nBox := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(stateColor).
				Background(pal.Surface).
				Padding(0, 1).
				Align(lipgloss.Center).
				Width(22).
				Render(fmt.Sprintf("%s\nAS%d [%s]", bgp.NeighborIP, bgp.PeerASN, stateUpper))
			topBoxStrings = append(topBoxStrings, nBox)
		}
	}

	// Calculate total width of top nodes row for connector scaling
	nodeCount := len(topBoxStrings)
	totalWidth := nodeCount*24 + (nodeCount-1)*3
	if totalWidth < 60 {
		totalWidth = 60
	}

	// Local Switch Box
	localBox := lipgloss.NewStyle().
		Bold(true).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(pal.Primary).
		Background(pal.Surface).
		Padding(0, 2).
		Align(lipgloss.Center).
		Width(36).
		Render(fmt.Sprintf("LOCAL SWITCH: %s\n(%s)", snap.Hostname, snap.Platform))

	localCentered := lipgloss.NewStyle().Width(totalWidth).Align(lipgloss.Center).Render(localBox)

	var diagramParts []string
	if len(topBoxStrings) > 0 {
		gapStr := "   "
		topNodesRow := lipgloss.JoinHorizontal(lipgloss.Top, interleaveStrings(topBoxStrings, gapStr)...)
		diagramParts = append(diagramParts, topNodesRow, "", "", localCentered)
	} else {
		noMatchMsg := lipgloss.NewStyle().
			Foreground(pal.Muted).
			Render("  (no BGP peers match filter)")
		diagramParts = append(diagramParts, noMatchMsg, "", localCentered)
	}

	diagramStr := lipgloss.JoinVertical(lipgloss.Left, diagramParts...)

	// BGP Session Table
	colHdrs := fmt.Sprintf("  %-16s %-10s %-10s %-16s %-16s %-10s %s",
		"NEIGHBOR IP", "PEER AS", "TYPE", "STATE", "INTERFACE", "UPTIME", "PREFIXES (Rx/Tx)",
	)

	var bgpRows []string
	bgpRows = append(bgpRows, lipgloss.NewStyle().Bold(true).Foreground(pal.Primary).Render(colHdrs))
	bgpRows = append(bgpRows, lipgloss.NewStyle().Foreground(pal.Muted).Render("  ──────────────── ────────── ────────── ──────────────── ──────────────── ────────── ────────────────"))

	if len(activeBgp) == 0 {
		bgpRows = append(bgpRows, lipgloss.NewStyle().Foreground(pal.Subtext).Render("  No matching BGP routing protocol adjacencies discovered"))
	} else {
		for _, peer := range activeBgp {
			stateUpper := strings.ToUpper(peer.SessionState)
			stateText := "● " + stateUpper
			stateStyle := lipgloss.NewStyle().Foreground(pal.Success)

			if stateUpper == "ACTIVE" || stateUpper == "CONNECT" || stateUpper == "CONNECTING" {
				stateText = "◯ " + stateUpper
				stateStyle = lipgloss.NewStyle().Foreground(pal.Warning)
			} else if stateUpper != "ESTABLISHED" {
				stateText = "✕ " + stateUpper
				stateStyle = lipgloss.NewStyle().Foreground(pal.Error)
			}

			ipStr := lipgloss.NewStyle().Foreground(pal.Text).Render(fmt.Sprintf("%-16s", peer.NeighborIP))
			asStr := lipgloss.NewStyle().Foreground(pal.Secondary).Render(fmt.Sprintf("%-10d", peer.PeerASN))
			typeStr := lipgloss.NewStyle().Foreground(pal.Primary).Render(fmt.Sprintf("%-10s", peer.PeerType))
			stStr := stateStyle.Render(fmt.Sprintf("%-16s", stateText))
			intfStr := lipgloss.NewStyle().Foreground(pal.Warning).Render(fmt.Sprintf("%-16s", peer.Interface))
			upStr := lipgloss.NewStyle().Foreground(pal.Subtext).Render(fmt.Sprintf("%-10s", peer.Uptime))
			pfxStr := fmt.Sprintf("%d / %d", peer.RxPrefixes, peer.TxPrefixes)

			row := fmt.Sprintf("  %s %s %s %s %s %s %s", ipStr, asStr, typeStr, stStr, intfStr, upStr, pfxStr)
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
		titleStyle.Render("🌐 BGP ROUTING PROTOCOL TOPOLOGY & MESH"),
		subStyle.Render("[Real-time BGP routing adjacencies & session states]"),
		searchBarStr,
	)

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
