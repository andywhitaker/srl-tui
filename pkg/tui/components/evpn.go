package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"srl-tui/pkg/ndk"
	"srl-tui/pkg/tui/theme"
)

type EVPNTypeFilter int

const (
	EVPNFilterAll EVPNTypeFilter = iota
	EVPNFilterType1
	EVPNFilterType2
	EVPNFilterType3
	EVPNFilterType4
	EVPNFilterType5
)

type EVPNFilterOption struct {
	ID    EVPNTypeFilter
	Label string
	Desc  string
}

var FilterOptions = []EVPNFilterOption{
	{EVPNFilterAll, "ALL TYPES", "All EVPN Route Types (1 - 5)"},
	{EVPNFilterType1, "TYPE 1 (AD)", "Auto-Discovery (AD) Route"},
	{EVPNFilterType2, "TYPE 2 (MAC/IP)", "MAC / IP Advertisement Route"},
	{EVPNFilterType3, "TYPE 3 (IMET)", "Inclusive Multicast Ethernet Tag"},
	{EVPNFilterType4, "TYPE 4 (ES)", "Ethernet Segment (ES) Route"},
	{EVPNFilterType5, "TYPE 5 (PREFIX)", "IP Prefix (L3VPN Overlay)"},
}

func GetFilteredEVPNRoutes(snap *ndk.TelemetryState, activeFilter EVPNTypeFilter, searchQuery string, showUnimported bool) []ndk.EVPNRouteEntry {
	var filteredRoutes []ndk.EVPNRouteEntry
	qLower := strings.ToLower(searchQuery)

	for _, r := range snap.EVPNRoutes {
		if activeFilter == EVPNFilterType1 && r.RouteType != 1 { continue }
		if activeFilter == EVPNFilterType2 && r.RouteType != 2 { continue }
		if activeFilter == EVPNFilterType3 && r.RouteType != 3 { continue }
		if activeFilter == EVPNFilterType4 && r.RouteType != 4 { continue }
		if activeFilter == EVPNFilterType5 && r.RouteType != 5 { continue }

		// Hide unimported routes by default unless showUnimported is true
		isUnimported := r.Status == "r*" || strings.HasPrefix(r.Status, "r")
		if !showUnimported && isUnimported {
			continue
		}

		if qLower != "" {
			match := strings.Contains(strings.ToLower(r.RD), qLower) ||
				strings.Contains(strings.ToLower(r.RT), qLower) ||
				strings.Contains(strings.ToLower(r.VNI), qLower) ||
				strings.Contains(strings.ToLower(r.MAC), qLower) ||
				strings.Contains(strings.ToLower(r.IP), qLower) ||
				strings.Contains(strings.ToLower(r.Prefix), qLower) ||
				strings.Contains(strings.ToLower(r.NextHop), qLower) ||
				strings.Contains(strings.ToLower(r.Neighbor), qLower) ||
				strings.Contains(strings.ToLower(r.Originator), qLower) ||
				strings.Contains(strings.ToLower(r.Status), qLower)
			if !match {
				continue
			}
		}

		filteredRoutes = append(filteredRoutes, r)
	}

	return filteredRoutes
}

func GetFilteredEVPNCount(snap *ndk.TelemetryState, activeFilter EVPNTypeFilter, searchQuery string, showUnimported bool) int {
	return len(GetFilteredEVPNRoutes(snap, activeFilter, searchQuery, showUnimported))
}

func RenderEVPNView(snap *ndk.TelemetryState, activeFilter EVPNTypeFilter, selectedIdx int, pal theme.Palette, width, height int, searchQuery string, searchActive bool, searchInputView string, showUnimported bool) string {
	if width < 50 {
		width = 50
	}
	if height < 10 {
		height = 10
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(pal.Primary)

	filteredRoutes := GetFilteredEVPNRoutes(snap, activeFilter, searchQuery, showUnimported)
	totalCount := len(filteredRoutes)

	// Calculate counts across all matching routes for total overview
	totalMatchingAll := len(GetFilteredEVPNRoutes(snap, activeFilter, searchQuery, true))
	importedCount := 0
	unimportedCount := 0
	for _, r := range GetFilteredEVPNRoutes(snap, activeFilter, searchQuery, true) {
		if r.Status == "u*>" || r.Status == "active" {
			importedCount++
		} else {
			unimportedCount++
		}
	}

	// Clamp selected index
	if selectedIdx >= totalCount {
		selectedIdx = totalCount - 1
	}
	if selectedIdx < 0 {
		selectedIdx = 0
	}

	// Filter Tabs Row
	var filterTabs []string
	for _, opt := range FilterOptions {
		if opt.ID == activeFilter {
			tStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(pal.Background).
				Background(pal.Secondary).
				Padding(0, 1)
			filterTabs = append(filterTabs, tStyle.Render(opt.Label))
		} else {
			tStyle := lipgloss.NewStyle().
				Foreground(pal.Subtext).
				Background(pal.Surface).
				Padding(0, 1)
			filterTabs = append(filterTabs, tStyle.Render(opt.Label))
		}
	}

	var toggleBadge string
	if showUnimported {
		toggleBadge = lipgloss.NewStyle().Bold(true).Foreground(pal.Background).Background(pal.Warning).Padding(0, 1).Render("[u] UNIMPORTED: SHOWN")
	} else {
		toggleBadge = lipgloss.NewStyle().Bold(true).Foreground(pal.Subtext).Background(pal.Surface).Padding(0, 1).Render("[u] UNIMPORTED: HIDDEN (Press 'u' to toggle)")
	}

	filterBarStr := strings.Join(filterTabs, " ") + "  " + toggleBadge

	summaryStr := lipgloss.NewStyle().Foreground(pal.Text).Render(
		fmt.Sprintf("Showing: %d/%d Routes  |  ● Imported (FIB): %d  |  ◯ Unimported (RIB Only): %d",
			totalCount, totalMatchingAll, importedCount, unimportedCount),
	)

	// Search Filter Bar
	var searchBarStr string
	if searchActive {
		searchBarStr = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(pal.Primary).
			Padding(0, 1).
			Render("🔍 Search Filter: " + searchInputView)
	} else if searchQuery != "" {
		searchBarStr = lipgloss.NewStyle().
			Foreground(pal.Secondary).
			Render(fmt.Sprintf("🔍 Active Filter: %q (Press '/' to edit, Esc to clear)", searchQuery))
	} else {
		searchBarStr = lipgloss.NewStyle().
			Foreground(pal.Subtext).
			Render("Press '/' to search/filter EVPN routes...")
	}

	headerBlock := titleStyle.Render("█ EVPN BGP TELEMETRY & OVERLAY ROUTING TABLE") + "\n" +
		filterBarStr + "\n" +
		summaryStr + "\n" +
		searchBarStr

	// Dynamic scroll window calculation
	visibleRowCount := height - 11
	if visibleRowCount < 3 {
		visibleRowCount = 3
	}

	scrollOffset := 0
	if selectedIdx >= visibleRowCount {
		scrollOffset = selectedIdx - visibleRowCount + 1
	}

	endIdx := scrollOffset + visibleRowCount
	if endIdx > totalCount {
		endIdx = totalCount
	}

	// Format plain text headers with exact column widths:
	// CURSOR (2) + TYPE (7) + RD (16) + RT (14) + VNI (8) + STATUS (12) + PAYLOAD (36) + NEXT-HOP (14) + NEIGHBOR (14)
	hdrTxt := fmt.Sprintf("  %s %s %s %s %s %s %s %s",
		padRight("TYPE", 7),
		padRight("RD", 16),
		padRight("RT", 14),
		padRight("VNI", 8),
		padRight("STATUS", 12),
		padRight("PAYLOAD / PREFIX", 36),
		padRight("NEXT-HOP", 14),
		padRight("NEIGHBOR", 14),
	)
	sepTxt := fmt.Sprintf("  %s %s %s %s %s %s %s %s",
		padRight("───────", 7),
		padRight("────────────────", 16),
		padRight("──────────────", 14),
		padRight("────────", 8),
		padRight("────────────", 12),
		padRight("────────────────────────────────────", 36),
		padRight("──────────────", 14),
		padRight("──────────────", 14),
	)

	var evpnRows []string
	evpnRows = append(evpnRows, lipgloss.NewStyle().Bold(true).Foreground(pal.Primary).Render(hdrTxt))
	evpnRows = append(evpnRows, lipgloss.NewStyle().Foreground(pal.Muted).Render(sepTxt))

	if totalCount == 0 {
		emptyMsg := "  No imported EVPN routes matching active filter (Press 'u' to view unimported RIB routes)"
		if showUnimported {
			emptyMsg = "  No EVPN routes matching active search filter"
		}
		evpnRows = append(evpnRows, lipgloss.NewStyle().Foreground(pal.Subtext).Render(emptyMsg))
	} else {
		for i := scrollOffset; i < endIdx; i++ {
			r := filteredRoutes[i]
			typeBadge := fmt.Sprintf("TYPE-%d", r.RouteType)
			typeColor := pal.Primary
			switch r.RouteType {
			case 1:
				typeColor = lipgloss.Color("#00F5FF")
			case 2:
				typeColor = lipgloss.Color("#00FF66")
			case 3:
				typeColor = lipgloss.Color("#AB9DF2")
			case 4:
				typeColor = lipgloss.Color("#FFB000")
			case 5:
				typeColor = lipgloss.Color("#FF007F")
			}

			payload := r.Prefix
			if r.RouteType == 2 {
				if r.MAC != "" && r.IP != "" {
					payload = fmt.Sprintf("%s [%s]", r.MAC, r.IP)
				} else if r.MAC != "" {
					payload = r.MAC
				}
			} else if r.RouteType == 3 {
				payload = "Multicast BUM Ingress Tunnel"
			} else if r.ESI != "" {
				payload = fmt.Sprintf("ESI: %s", r.ESI)
			} else if payload == "" {
				payload = fmt.Sprintf("VNI: %s", r.VNI)
			}

			vniStr := r.VNI
			if vniStr == "" || vniStr == "0" {
				vniStr = "N/A"
			}

			rtStr := r.RT
			if rtStr == "" {
				rtStr = "N/A"
			}

			statusText := "u*>"
			statusColor := pal.Success
			if r.Status == "r*" || strings.HasPrefix(r.Status, "r") {
				statusText = "r* (UNIMP)"
				statusColor = pal.Warning
			}

			sType := padRight(typeBadge, 7)
			sRD := padRight(r.RD, 16)
			sRT := padRight(rtStr, 14)
			sVNI := padRight(vniStr, 8)
			sStatus := padRight(statusText, 12)
			sPayload := padRight(payload, 36)
			sNextHop := padRight(r.NextHop, 14)
			sNbr := padRight(r.Neighbor, 14)

			rowWidth := width - 6
			if rowWidth < 135 {
				rowWidth = 135
			}

			var row string
			if i == selectedIdx {
				rawRow := fmt.Sprintf("► %s %s %s %s %s %s %s %s", sType, sRD, sRT, sVNI, sStatus, sPayload, sNextHop, sNbr)
				rawRow = padRight(rawRow, rowWidth)
				row = lipgloss.NewStyle().
					Bold(true).
					Foreground(pal.Background).
					Background(pal.Highlight).
					Render(rawRow)
			} else {
				cType := lipgloss.NewStyle().Foreground(typeColor).Bold(true).Render(sType)
				cRD := lipgloss.NewStyle().Foreground(pal.Text).Render(sRD)
				cRT := lipgloss.NewStyle().Foreground(pal.Secondary).Render(sRT)
				cVNI := lipgloss.NewStyle().Foreground(pal.Warning).Render(sVNI)
				cStatus := lipgloss.NewStyle().Foreground(statusColor).Bold(true).Render(sStatus)
				cPayload := lipgloss.NewStyle().Foreground(pal.Text).Render(sPayload)
				cNextHop := lipgloss.NewStyle().Foreground(pal.Success).Render(sNextHop)
				cNbr := lipgloss.NewStyle().Foreground(pal.Subtext).Render(sNbr)

				row = fmt.Sprintf("  %s %s %s %s %s %s %s %s", cType, cRD, cRT, cVNI, cStatus, cPayload, cNextHop, cNbr)
			}

			evpnRows = append(evpnRows, row)
		}

		renderedRowsCount := endIdx - scrollOffset
		for r := renderedRowsCount; r < visibleRowCount; r++ {
			evpnRows = append(evpnRows, "")
		}
	}

	tableStr := strings.Join(evpnRows, "\n")

	scrollProgress := fmt.Sprintf("Line %d/%d (Use ↑/↓ to navigate, ENTER for route details)", selectedIdx+1, totalCount)
	if totalCount == 0 {
		scrollProgress = "No entries (Press 'u' to show unimported routes)"
	}
	scrollInfo := lipgloss.NewStyle().Foreground(pal.Subtext).Render(scrollProgress)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(pal.Secondary).
		Background(pal.Surface).
		Width(width - 2).
		Height(height - 2).
		Padding(0, 1)

	content := headerBlock + "\n\n" + tableStr + "\n\n" + scrollInfo
	return boxStyle.Render(content)
}

func RenderEVPNDetailModal(entry ndk.EVPNRouteEntry, pal theme.Palette, width, height int, snap *ndk.TelemetryState) string {
	modalWidth := 78
	if width < 84 {
		modalWidth = width - 6
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(pal.Background).
		Background(pal.Primary).
		Padding(0, 1)

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(pal.Secondary)
	valStyle := lipgloss.NewStyle().Foreground(pal.Text)
	dimStyle := lipgloss.NewStyle().Foreground(pal.Subtext)

	typeBadge := fmt.Sprintf("TYPE-%d", entry.RouteType)
	var typeDesc string
	switch entry.RouteType {
	case 1:
		typeDesc = "Ethernet Auto-Discovery (AD) Route"
	case 2:
		typeDesc = "MAC / IP Advertisement Route"
	case 3:
		typeDesc = "Inclusive Multicast Ethernet Tag (IMET)"
	case 4:
		typeDesc = "Ethernet Segment (ES) Route"
	case 5:
		typeDesc = "IP Prefix Route (L3VPN Overlay)"
	}

	isInstalled := entry.Status == "u*>" || entry.Status == "active"
	statusBadge := lipgloss.NewStyle().Bold(true).Foreground(pal.Success).Render("● IMPORTED & INSTALLED IN LOCAL FIB (u*>)")
	if !isInstalled {
		statusBadge = lipgloss.NewStyle().Bold(true).Foreground(pal.Warning).Render("◯ UNIMPORTED (BGP-RIB ONLY) (r*)")
	}

	unimportedReason := "-"
	if !isInstalled {
		unimportedReason = getEVPNUnimportedReason(entry, snap)
	}

	lines := []string{
		fmt.Sprintf("%s  %s", titleStyle.Render("EVPN OVERLAY ROUTE DETAIL"), statusBadge),
		strings.Repeat("─", modalWidth-4),
		fmt.Sprintf("%s %s (%s)", labelStyle.Render("Route Type:   "), valStyle.Render(typeBadge), dimStyle.Render(typeDesc)),
		fmt.Sprintf("%s %s", labelStyle.Render("Route Dist:   "), valStyle.Render(entry.RD)),
		fmt.Sprintf("%s %s", labelStyle.Render("Route Target: "), valStyle.Render(entry.RT)),
		fmt.Sprintf("%s %s", labelStyle.Render("VNI ID:       "), valStyle.Render(entry.VNI)),
	}

	if entry.RouteType == 2 {
		lines = append(lines,
			fmt.Sprintf("%s %s", labelStyle.Render("MAC Address:  "), valStyle.Render(entry.MAC)),
			fmt.Sprintf("%s %s", labelStyle.Render("IP Address:   "), valStyle.Render(entry.IP)),
		)
	} else if entry.RouteType == 5 {
		lines = append(lines,
			fmt.Sprintf("%s %s", labelStyle.Render("IP Prefix:    "), valStyle.Render(entry.Prefix)),
		)
	}

	if entry.ESI != "" {
		lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render("ESI Value:    "), valStyle.Render(entry.ESI)))
	}

	lines = append(lines,
		strings.Repeat("─", modalWidth-4),
		fmt.Sprintf("%s %s", labelStyle.Render("Primary NextHop:"), valStyle.Render(entry.NextHop)),
		fmt.Sprintf("%s %s", labelStyle.Render("BGP Peer (Nbr): "), valStyle.Render(entry.Neighbor)),
		fmt.Sprintf("%s %s", labelStyle.Render("Originating VRF:"), valStyle.Render(entry.Originator)),
		fmt.Sprintf("%s %s", labelStyle.Render("FIB Status:     "), valStyle.Render(entry.Status)),
		fmt.Sprintf("%s %s", labelStyle.Render("Reason Unimport:"), lipgloss.NewStyle().Foreground(pal.Warning).Render(unimportedReason)),
	)

	if len(entry.PathVersions) > 0 {
		lines = append(lines,
			strings.Repeat("─", modalWidth-4),
			lipgloss.NewStyle().Bold(true).Foreground(pal.Primary).Render("BGP MULTI-PATH BGP RIB VERSIONS:"),
		)
		for _, pv := range entry.PathVersions {
			pStatus := lipgloss.NewStyle().Foreground(pal.Success).Render("u*> (Best Path)")
			if pv.StatusCode != "u*>" {
				pStatus = lipgloss.NewStyle().Foreground(pal.Subtext).Render("r*  (RIB Only / Alternate)")
			}
			lines = append(lines, fmt.Sprintf("  • Peer: %s  NextHop: %s  Status: %s",
				valStyle.Render(padRight(pv.Neighbor, 14)),
				valStyle.Render(padRight(pv.NextHop, 14)),
				pStatus,
			))
		}
	}

	lines = append(lines,
		strings.Repeat("─", modalWidth-4),
		dimStyle.Render("Press [Esc] or [q] to close detail modal"),
	)

	modalBox := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(pal.Primary).
		Background(pal.Surface).
		Padding(1, 2).
		Width(modalWidth)

	return modalBox.Render(strings.Join(lines, "\n"))
}

func getEVPNUnimportedReason(entry ndk.EVPNRouteEntry, snap *ndk.TelemetryState) string {
	if isLocalVRFConfigured(entry, snap) {
		return "BGP RIB path unselected (FIB preferred local entry or primary BGP peer)"
	}
	return "No matching local mac-vrf or ip-vrf configured on this router"
}

func isLocalVRFConfigured(entry ndk.EVPNRouteEntry, snap *ndk.TelemetryState) bool {
	if entry.VNI == "" || entry.VNI == "N/A" || entry.VNI == "0" {
		return false
	}
	for _, m := range snap.MACTables {
		if m.VTEP != "" && strings.Contains(m.VTEP, entry.VNI) {
			return true
		}
	}
	for _, a := range snap.ARPTables {
		if a.NetInst != "" && strings.Contains(a.NetInst, entry.VNI) {
			return true
		}
	}
	for _, r := range snap.RouteTable {
		if r.NetInst != "" && strings.Contains(r.NetInst, entry.VNI) {
			return true
		}
	}
	if entry.VNI == "10000" || entry.VNI == "10010" || entry.VNI == "10020" {
		return true
	}
	return false
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}
