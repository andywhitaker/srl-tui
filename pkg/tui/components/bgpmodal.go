package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"srl-tui/pkg/ndk"
	"srl-tui/pkg/tui/theme"
)

func RenderBGPDetailModal(entry ndk.BGPPeerState, pal theme.Palette, width, height int, snap *ndk.TelemetryState, confirmPrompt bool, confirmAction, confirmGroup string) string {
	modalWidth := width - 14
	if modalWidth < 70 {
		modalWidth = 70
	}
	if modalWidth > 110 {
		modalWidth = 110
	}

	// 1. Header Banner
	var headerStr string
	if entry.InMaintenance {
		// Checkered header with orange road cones
		checkeredBar := "🏁 🚧 🏁 🚧 🏁  NEIGHBOR IN MAINTENANCE MODE  🏁 🚧 🏁 🚧 🏁"
		headerStr = lipgloss.NewStyle().
			Bold(true).
			Foreground(pal.Warning).
			Background(pal.Background).
			Align(lipgloss.Center).
			Width(modalWidth - 4).
			Render(checkeredBar)
	} else {
		headerStr = lipgloss.NewStyle().
			Bold(true).
			Foreground(pal.Primary).
			Render(fmt.Sprintf("🌐 BGP NEIGHBOR DETAILS: %s", entry.NeighborIP))
	}

	// 2. Info Grid
	maintText := "INACTIVE"
	maintStyle := lipgloss.NewStyle().Foreground(pal.Subtext)
	if entry.InMaintenance {
		maintText = "ACTIVE 🚧"
		maintStyle = lipgloss.NewStyle().Foreground(pal.Warning).Bold(true)
	}

	grpName := entry.MaintenanceGroup
	if grpName == "" {
		grpName = fmt.Sprintf("maint-bgp-%s", strings.ReplaceAll(entry.NeighborIP, ".", "-"))
	}

	stateUpper := strings.ToUpper(entry.SessionState)
	stateStyle := lipgloss.NewStyle().Foreground(pal.Success)
	if stateUpper != "ESTABLISHED" {
		stateStyle = lipgloss.NewStyle().Foreground(pal.Error)
	}

	afListStr := strings.Join(entry.AddrFamilies, ", ")
	if afListStr == "" {
		afListStr = "ipv4-unicast"
	}

	var pfxParts []string
	if len(entry.AddrFamilies) > 0 {
		for _, af := range entry.AddrFamilies {
			if st, ok := entry.AFStats[af]; ok {
				pfxParts = append(pfxParts, fmt.Sprintf("%d/%d", st.RxPrefixes, st.TxPrefixes))
			} else {
				pfxParts = append(pfxParts, fmt.Sprintf("%d/%d", entry.RxPrefixes, entry.TxPrefixes))
			}
		}
	}
	pfxSummaryStr := strings.Join(pfxParts, ", ")
	if pfxSummaryStr == "" {
		pfxSummaryStr = fmt.Sprintf("%d/%d", entry.RxPrefixes, entry.TxPrefixes)
	}
	pfxSummaryStr += fmt.Sprintf(" (Total: %d rx / %d tx)", entry.RxPrefixes, entry.TxPrefixes)

	fmtRow := func(lbl string, val string) string {
		l := lipgloss.NewStyle().Foreground(pal.Subtext).Background(pal.Background).Render(padRight(lbl, 20))
		gap := lipgloss.NewStyle().Foreground(pal.Muted).Background(pal.Background).Render(" : ")
		v := lipgloss.NewStyle().Background(pal.Background).Render(val)
		return lipgloss.NewStyle().Background(pal.Background).Render("  " + l + gap + v)
	}

	infoRows := []string{
		fmtRow("Neighbor IP", lipgloss.NewStyle().Bold(true).Foreground(pal.Text).Background(pal.Background).Render(entry.NeighborIP)),
		fmtRow("Peer ASN", lipgloss.NewStyle().Foreground(pal.Secondary).Background(pal.Background).Render(fmt.Sprintf("AS%d", entry.PeerASN))),
		fmtRow("Local ASN", lipgloss.NewStyle().Foreground(pal.Secondary).Background(pal.Background).Render(fmt.Sprintf("AS%d", entry.LocalASN))),
		fmtRow("Peer Type", lipgloss.NewStyle().Foreground(pal.Primary).Background(pal.Background).Render(entry.PeerType)),
		fmtRow("Session State", stateStyle.Background(pal.Background).Render(stateUpper)),
		fmtRow("Local Interface", lipgloss.NewStyle().Foreground(pal.Warning).Background(pal.Background).Render(entry.Interface)),
		fmtRow("Uptime", lipgloss.NewStyle().Foreground(pal.Text).Background(pal.Background).Render(entry.Uptime)),
		fmtRow("Active AFs", lipgloss.NewStyle().Foreground(pal.Primary).Background(pal.Background).Render(afListStr)),
		fmtRow("Prefixes (Rx/Tx)", lipgloss.NewStyle().Foreground(pal.Text).Background(pal.Background).Render(pfxSummaryStr)),
		fmtRow("Maintenance Group", lipgloss.NewStyle().Foreground(pal.Text).Background(pal.Background).Render(grpName)),
		fmtRow("Maintenance Status", maintStyle.Background(pal.Background).Render(maintText)),
	}

	infoBox := lipgloss.JoinVertical(lipgloss.Left, infoRows...)

	// 3. Active Address Families Detail Section
	afHdr := lipgloss.NewStyle().Bold(true).Foreground(pal.Primary).Background(pal.Background).Render("  ACTIVE ADDRESS FAMILIES & ROUTE STATS:")
	afTableHdrs := lipgloss.NewStyle().Bold(true).Foreground(pal.Subtext).Background(pal.Background).Render(fmt.Sprintf("    %-20s %-16s %-16s", "ADDRESS FAMILY", "RECEIVED ROUTES", "SENT ROUTES"))
	afDivider := lipgloss.NewStyle().Foreground(pal.Muted).Background(pal.Background).Render("    ──────────────────── ──────────────── ────────────────")

	fmtAfRow := func(af string, rx, tx uint32) string {
		c1 := lipgloss.NewStyle().Foreground(pal.Text).Background(pal.Background).Render(padRight(af, 20))
		c2 := lipgloss.NewStyle().Foreground(pal.Success).Background(pal.Background).Render(padRight(fmt.Sprintf("%d", rx), 16))
		c3 := lipgloss.NewStyle().Foreground(pal.Secondary).Background(pal.Background).Render(fmt.Sprintf("%d", tx))
		gap := lipgloss.NewStyle().Background(pal.Background).Render(" ")
		return lipgloss.NewStyle().Background(pal.Background).Render("    " + c1 + gap + c2 + gap + c3)
	}

	var afRows []string
	afRows = append(afRows, afHdr, afTableHdrs, afDivider)
	if len(entry.AddrFamilies) == 0 {
		rx := entry.RxPrefixes
		tx := entry.TxPrefixes
		if st, ok := entry.AFStats["ipv4-unicast"]; ok {
			rx = st.RxPrefixes
			tx = st.TxPrefixes
		}
		afRows = append(afRows, fmtAfRow("ipv4-unicast", rx, tx))
	} else {
		for _, af := range entry.AddrFamilies {
			rx := entry.RxPrefixes
			tx := entry.TxPrefixes
			if st, ok := entry.AFStats[af]; ok {
				rx = st.RxPrefixes
				tx = st.TxPrefixes
			}
			afRows = append(afRows, fmtAfRow(af, rx, tx))
		}
	}
	afBox := strings.Join(afRows, "\n")

	// 4. Instructions Footer
	maintActionText := "Press 'm' to put neighbor into Maintenance Mode"
	if entry.InMaintenance {
		maintActionText = "Press 'm' to take neighbor out of Maintenance Mode"
	}
	footerStr := lipgloss.NewStyle().Foreground(pal.Muted).Background(pal.Background).Render(fmt.Sprintf("%s • Esc/Enter/q to close", maintActionText))

	rawContent := lipgloss.JoinVertical(
		lipgloss.Left,
		headerStr,
		"",
		infoBox,
		"",
		afBox,
		"",
		footerStr,
	)

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
	modalContent := strings.Join(styledLines, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(pal.Primary).
		BorderBackground(pal.Background).
		Background(pal.Background).
		Padding(1, 2).
		Width(modalWidth)

	if entry.InMaintenance {
		boxStyle = boxStyle.BorderForeground(pal.Warning)
	}

	mainModal := boxStyle.Render(modalContent)

	// Render confirmation sub-overlay if prompt is active
	if confirmPrompt {
		actionVerb := "put into"
		if confirmAction == "disable" {
			actionVerb = "take out of"
		}

		promptTitle := lipgloss.NewStyle().Bold(true).Foreground(pal.Warning).Render("⚠️  BGP MAINTENANCE MODE CONFIRMATION")
		promptBody := fmt.Sprintf("Are you sure you want to %s Maintenance Mode for neighbor %s?\nMaintenance Group: '%s'", actionVerb, entry.NeighborIP, confirmGroup)
		promptKeys := lipgloss.NewStyle().Bold(true).Foreground(pal.Success).Render(" [Y] Yes, Confirm    [N] Cancel / Esc")

		promptBox := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(pal.Warning).
			BorderBackground(pal.Background).
			Background(pal.Background).
			Padding(1, 2).
			Align(lipgloss.Center).
			Width(modalWidth - 10).
			Render(fmt.Sprintf("%s\n\n%s\n\n%s", promptTitle, promptBody, promptKeys))

		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, promptBox)
	}

	return mainModal
}
