package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"srl-tui/pkg/ndk"
	"srl-tui/pkg/tui/theme"
)

type RouteView struct {
	SelectedIdx  int
	ScrollOffset int
}

func NewRouteView() *RouteView {
	return &RouteView{
		SelectedIdx:  0,
		ScrollOffset: 0,
	}
}

func (v *RouteView) ScrollUp() {
	if v.SelectedIdx > 0 {
		v.SelectedIdx--
	}
}

func (v *RouteView) ScrollDown(total int) {
	if total > 0 && v.SelectedIdx < total-1 {
		v.SelectedIdx++
	}
}

func GetFilteredRoutes(snap *ndk.TelemetryState, searchQuery string) []ndk.RouteEntry {
	var filteredRoutes []ndk.RouteEntry
	qLower := strings.ToLower(searchQuery)

	for _, r := range snap.RouteTable {
		if qLower != "" {
			match := strings.Contains(strings.ToLower(r.Prefix), qLower) ||
				strings.Contains(strings.ToLower(r.Protocol), qLower) ||
				strings.Contains(strings.ToLower(r.NextHop), qLower) ||
				strings.Contains(strings.ToLower(r.NetInst), qLower) ||
				strings.Contains(fmt.Sprintf("%d", r.Preference), qLower) ||
				strings.Contains(fmt.Sprintf("%d", r.Metric), qLower)
			if !match {
				for _, nh := range r.NextHops {
					if strings.Contains(strings.ToLower(nh), qLower) {
						match = true
						break
					}
				}
			}
			if !match {
				continue
			}
		}
		filteredRoutes = append(filteredRoutes, r)
	}
	return filteredRoutes
}

func (v *RouteView) Render(snap *ndk.TelemetryState, pal theme.Palette, width, height int, searchQuery string, searchActive bool, searchInputView string) string {
	if width < 50 {
		width = 50
	}
	if height < 10 {
		height = 10
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(pal.Primary).Background(pal.Background)
	subStyle := lipgloss.NewStyle().Foreground(pal.Subtext).Background(pal.Background)

	filteredRoutes := GetFilteredRoutes(snap, searchQuery)
	totalCount := len(filteredRoutes)

	bgpCount := 0
	directCount := 0
	staticCount := 0

	for _, r := range filteredRoutes {
		switch strings.ToLower(r.Protocol) {
		case "bgp", "bgp_evpn", "evpn":
			bgpCount++
		case "direct", "local", "connected":
			directCount++
		case "static":
			staticCount++
		}
	}

	// Clamp SelectedIdx
	if v.SelectedIdx >= totalCount {
		v.SelectedIdx = totalCount - 1
	}
	if v.SelectedIdx < 0 {
		v.SelectedIdx = 0
	}

	summaryStr := lipgloss.NewStyle().Foreground(pal.Text).Background(pal.Background).Render(
		fmt.Sprintf("Total Matching Routes: %d  |  BGP: %d  |  Direct: %d  |  Static: %d",
			totalCount, bgpCount, directCount, staticCount),
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
	// CURSOR (2) + DESTINATION (20) + PROTOCOL (10) + PREFERENCE (12) + METRIC (8) + NEXT-HOP (20) + NET-INST (12)
	hdrTxt := fmt.Sprintf("  %s %s %s %s %s %s",
		padRight("DESTINATION PREFIX", 20),
		padRight("PROTOCOL", 10),
		padRight("PREFERENCE", 12),
		padRight("METRIC", 8),
		padRight("NEXT-HOP", 36),
		padRight("NET-INSTANCE", 12),
	)
	sepTxt := fmt.Sprintf("  %s %s %s %s %s %s",
		padRight("────────────────────", 20),
		padRight("──────────", 10),
		padRight("────────────", 12),
		padRight("────────", 8),
		padRight("────────────────────────────────────", 36),
		padRight("────────────", 12),
	)

	var routeRows []string
	routeRows = append(routeRows, lipgloss.NewStyle().Bold(true).Foreground(pal.Primary).Render(hdrTxt))
	routeRows = append(routeRows, lipgloss.NewStyle().Foreground(pal.Muted).Render(sepTxt))

	if totalCount == 0 {
		routeRows = append(routeRows, lipgloss.NewStyle().Foreground(pal.Subtext).Render("  No IP routes matching search filter"))
	} else {
		for i := v.ScrollOffset; i < endIdx; i++ {
			entry := filteredRoutes[i]
			pName := strings.ToUpper(entry.Protocol)
			pColor := pal.Success
			switch strings.ToLower(entry.Protocol) {
			case "bgp", "bgp_evpn", "evpn":
				pColor = pal.Secondary
			case "direct", "local", "connected":
				pColor = pal.Success
			case "static":
				pColor = pal.Warning
			case "ospf":
				pColor = pal.Primary
			}

			nhDisplay := entry.NextHop
			if len(entry.NextHops) > 1 {
				nhDisplay = fmt.Sprintf("%s [ECMP x%d]", strings.Join(entry.NextHops, ", "), len(entry.NextHops))
			} else if nhDisplay == "" || nhDisplay == "0" {
				nhDisplay = "direct"
			}

			sPrefix := padRight(entry.Prefix, 21)
			sProto := padRight(pName, 11)
			sPref := padRight(fmt.Sprintf("%d", entry.Preference), 13)
			sMetric := padRight(fmt.Sprintf("%d", entry.Metric), 9)
			sNextHop := padRight(nhDisplay, 37)
			sNetInst := padRight(entry.NetInst, 12)

			rowWidth := width - 6
			if rowWidth < 110 {
				rowWidth = 110
			}

			var row string
			if i == v.SelectedIdx {
				rawRow := fmt.Sprintf("► %s%s%s%s%s%s", sPrefix, sProto, sPref, sMetric, sNextHop, sNetInst)
				rawRow = padRight(rawRow, rowWidth)
				row = lipgloss.NewStyle().
					Bold(true).
					Foreground(pal.Background).
					Background(pal.Highlight).
					Render(rawRow)
			} else {
				fPrefix := lipgloss.NewStyle().Foreground(pal.Text).Background(pal.Background).Render(sPrefix)
				fProto := lipgloss.NewStyle().Foreground(pColor).Background(pal.Background).Bold(true).Render(sProto)
				fPref := lipgloss.NewStyle().Foreground(pal.Subtext).Background(pal.Background).Render(sPref)
				fMetric := lipgloss.NewStyle().Foreground(pal.Subtext).Background(pal.Background).Render(sMetric)
				nhColor := pal.Warning
				if len(entry.NextHops) > 1 {
					nhColor = lipgloss.Color("#00FF66")
				}
				fNextHop := lipgloss.NewStyle().Foreground(nhColor).Background(pal.Background).Render(sNextHop)
				fNetInst := lipgloss.NewStyle().Foreground(pal.Primary).Background(pal.Background).Render(sNetInst)

				row = lipgloss.NewStyle().Background(pal.Background).Render(fmt.Sprintf("  %s%s%s%s%s%s", fPrefix, fProto, fPref, fMetric, fNextHop, fNetInst))
			}
			routeRows = append(routeRows, row)
		}
	}

	// Footer scroll status line
	scrollStatus := lipgloss.NewStyle().Foreground(pal.Secondary).Bold(true).Render(
		fmt.Sprintf(" [Route %d/%d]  [↑/↓ to scroll, ENTER for details] ", v.SelectedIdx+1, totalCount),
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
		titleStyle.Render("🔀 IP ROUTING TABLE (RIB / FIB)"),
		subStyle.Render("[Multi-Instance IP Route Table]"),
		searchBarStr,
	))

	return boxStyle.Render(header + "\n" + summaryStr + "\n\n" + strings.Join(routeRows, "\n") + "\n" + scrollStatus)
}

func RenderRouteDetailModal(r ndk.RouteEntry, snap *ndk.TelemetryState, pal theme.Palette, width, height int) string {
	modalWidth := 74
	if width < modalWidth {
		modalWidth = width - 4
	}

	stillExists := false
	for _, live := range snap.RouteTable {
		if live.NetInst == r.NetInst && live.Prefix == r.Prefix {
			stillExists = true
			break
		}
	}

	var statusBadge string
	if stillExists {
		statusBadge = lipgloss.NewStyle().Bold(true).Foreground(pal.Background).Background(pal.Success).Padding(0, 1).Render("● ACTIVE IN FIB / IP ROUTING TABLE")
	} else {
		statusBadge = lipgloss.NewStyle().Bold(true).Foreground(pal.Background).Background(pal.Error).Padding(0, 1).Render("⚠️ ROUTE WITHDRAWN FROM ROUTING TABLE")
	}

	pColor := pal.Success
	switch strings.ToLower(r.Protocol) {
	case "bgp", "bgp_evpn", "evpn":
		pColor = pal.Secondary
	case "direct", "local", "connected":
		pColor = pal.Success
	case "static":
		pColor = pal.Warning
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(pColor).Background(pal.Background)
	valStyle := lipgloss.NewStyle().Foreground(pal.Text).Background(pal.Background)

	nhDisplay := r.NextHop
	if len(r.NextHops) > 1 {
		nhDisplay = fmt.Sprintf("%s (ECMP x%d)", strings.Join(r.NextHops, ", "), len(r.NextHops))
	} else if nhDisplay == "" || nhDisplay == "0" {
		nhDisplay = "direct (locally connected interface)"
	}

	fmtRow := func(lbl string, val string) string {
		l := lipgloss.NewStyle().Foreground(pal.Subtext).Background(pal.Background).Render(padRight(lbl, 24))
		gap := lipgloss.NewStyle().Foreground(pal.Muted).Background(pal.Background).Render(" : ")
		v := lipgloss.NewStyle().Foreground(pal.Text).Background(pal.Background).Render(val)
		return lipgloss.NewStyle().Background(pal.Background).Render("  " + l + gap + v)
	}

	lines := []string{
		titleStyle.Render("🔀 IP ROUTE ENTRY DETAILS - " + r.Prefix),
		statusBadge,
		lipgloss.NewStyle().Foreground(pal.Muted).Background(pal.Background).Render(strings.Repeat("─", modalWidth-4)),
		fmtRow("Destination Prefix", r.Prefix),
		fmtRow("Route Protocol", strings.ToUpper(r.Protocol)),
		fmtRow("Resolved Next-Hop", nhDisplay),
		fmtRow("Network Instance VRF", r.NetInst),
		fmtRow("Route Preference", fmt.Sprintf("%d", r.Preference)),
		fmtRow("Route Metric / Cost", fmt.Sprintf("%d", r.Metric)),
	}

	if len(r.NextHops) > 1 {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(pal.Primary).Background(pal.Background).Render(fmt.Sprintf("Equal-Cost Multi-Path (ECMP): %d Active Next-Hop Paths:", len(r.NextHops))))
		for idx, nh := range r.NextHops {
			lines = append(lines, fmt.Sprintf("  [%d] Next-Hop Path: %s", idx+1, valStyle.Render(nh)))
		}
	}

	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(pal.Muted).Background(pal.Background).Render("Press ESC or ENTER to close detail window"))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(pColor).
		BorderBackground(pal.Background).
		Background(pal.Background).
		Padding(1, 2).
		Width(modalWidth)

	rawContent := strings.Join(lines, "\n")
	contentWidth := modalWidth - 4
	cr, cg, cb, _ := pal.Background.RGBA()
	bgSeq := fmt.Sprintf("\x1b[48;2;%d;%d;%dm", uint8(cr>>8), uint8(cg>>8), uint8(cb>>8))
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
