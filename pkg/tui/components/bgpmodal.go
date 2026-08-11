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
			Background(pal.Surface).
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

	infoRows := []string{
		fmt.Sprintf("  %-20s %s", lipgloss.NewStyle().Foreground(pal.Subtext).Render("Neighbor IP:"), lipgloss.NewStyle().Bold(true).Foreground(pal.Text).Render(entry.NeighborIP)),
		fmt.Sprintf("  %-20s %s", lipgloss.NewStyle().Foreground(pal.Subtext).Render("Peer ASN:"), lipgloss.NewStyle().Foreground(pal.Secondary).Render(fmt.Sprintf("AS%d", entry.PeerASN))),
		fmt.Sprintf("  %-20s %s", lipgloss.NewStyle().Foreground(pal.Subtext).Render("Local ASN:"), lipgloss.NewStyle().Foreground(pal.Secondary).Render(fmt.Sprintf("AS%d", entry.LocalASN))),
		fmt.Sprintf("  %-20s %s", lipgloss.NewStyle().Foreground(pal.Subtext).Render("Peer Type:"), lipgloss.NewStyle().Foreground(pal.Primary).Render(entry.PeerType)),
		fmt.Sprintf("  %-20s %s", lipgloss.NewStyle().Foreground(pal.Subtext).Render("Session State:"), stateStyle.Render(stateUpper)),
		fmt.Sprintf("  %-20s %s", lipgloss.NewStyle().Foreground(pal.Subtext).Render("Local Interface:"), lipgloss.NewStyle().Foreground(pal.Warning).Render(entry.Interface)),
		fmt.Sprintf("  %-20s %s", lipgloss.NewStyle().Foreground(pal.Subtext).Render("Uptime:"), lipgloss.NewStyle().Foreground(pal.Text).Render(entry.Uptime)),
		fmt.Sprintf("  %-20s %s", lipgloss.NewStyle().Foreground(pal.Subtext).Render("Active AFs:"), lipgloss.NewStyle().Foreground(pal.Primary).Render(afListStr)),
		fmt.Sprintf("  %-20s %s", lipgloss.NewStyle().Foreground(pal.Subtext).Render("Prefixes (Rx/Tx):"), pfxSummaryStr),
		fmt.Sprintf("  %-20s %s", lipgloss.NewStyle().Foreground(pal.Subtext).Render("Maintenance Group:"), lipgloss.NewStyle().Foreground(pal.Text).Render(grpName)),
		fmt.Sprintf("  %-20s %s", lipgloss.NewStyle().Foreground(pal.Subtext).Render("Maintenance Status:"), maintStyle.Render(maintText)),
	}

	infoBox := lipgloss.JoinVertical(lipgloss.Left, infoRows...)

	// 3. Active Address Families Detail Section
	afHdr := lipgloss.NewStyle().Bold(true).Foreground(pal.Primary).Render("  ACTIVE ADDRESS FAMILIES & ROUTE STATS:")
	afTableHdrs := fmt.Sprintf("    %-20s %-16s %-16s", "ADDRESS FAMILY", "RECEIVED ROUTES", "SENT ROUTES")
	afDivider := "    ──────────────────── ──────────────── ────────────────"

	var afRows []string
	afRows = append(afRows, afHdr, afTableHdrs, afDivider)
	if len(entry.AddrFamilies) == 0 {
		rx := entry.RxPrefixes
		tx := entry.TxPrefixes
		if st, ok := entry.AFStats["ipv4-unicast"]; ok {
			rx = st.RxPrefixes
			tx = st.TxPrefixes
		}
		afRows = append(afRows, fmt.Sprintf("    %-20s %-16d %-16d", "ipv4-unicast", rx, tx))
	} else {
		for _, af := range entry.AddrFamilies {
			rx := entry.RxPrefixes
			tx := entry.TxPrefixes
			if st, ok := entry.AFStats[af]; ok {
				rx = st.RxPrefixes
				tx = st.TxPrefixes
			}
			afRows = append(afRows, fmt.Sprintf("    %-20s %-16d %-16d", af, rx, tx))
		}
	}
	afBox := strings.Join(afRows, "\n")

	// 4. Instructions Footer
	maintActionText := "Press 'm' to put neighbor into Maintenance Mode"
	if entry.InMaintenance {
		maintActionText = "Press 'm' to take neighbor out of Maintenance Mode"
	}
	footerStr := lipgloss.NewStyle().Foreground(pal.Muted).Render(fmt.Sprintf("%s • Esc/Enter/q to close", maintActionText))

	modalContent := lipgloss.JoinVertical(
		lipgloss.Left,
		headerStr,
		"",
		infoBox,
		"",
		afBox,
		"",
		footerStr,
	)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(pal.Primary).
		Background(pal.Surface).
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
			Background(pal.Background).
			Padding(1, 2).
			Align(lipgloss.Center).
			Width(modalWidth - 10).
			Render(fmt.Sprintf("%s\n\n%s\n\n%s", promptTitle, promptBody, promptKeys))

		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, promptBox)
	}

	return mainModal
}
