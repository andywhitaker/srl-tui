package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"srl-tui/pkg/ndk"
	"srl-tui/pkg/tui/theme"
)

type ARPMACView struct {
	ActivePane int // 0 = ARP Table, 1 = MAC Table
	ARPScroll  int
	MACScroll  int
}

func NewARPMACView() *ARPMACView {
	return &ARPMACView{
		ActivePane: 0,
		ARPScroll:  0,
		MACScroll:  0,
	}
}

func (v *ARPMACView) ScrollUp() {
	if v.ActivePane == 0 {
		if v.ARPScroll > 0 {
			v.ARPScroll--
		}
	} else {
		if v.MACScroll > 0 {
			v.MACScroll--
		}
	}
}

func (v *ARPMACView) ScrollDown() {
	if v.ActivePane == 0 {
		v.ARPScroll++
	} else {
		v.MACScroll++
	}
}

func (v *ARPMACView) PageUp() {
	for i := 0; i < 10; i++ {
		v.ScrollUp()
	}
}

func (v *ARPMACView) PageDown() {
	for i := 0; i < 10; i++ {
		v.ScrollDown()
	}
}

func (v *ARPMACView) TogglePane() {
	v.ActivePane = 1 - v.ActivePane
}

func (v *ARPMACView) Render(state *ndk.TelemetryState, width, height int, th theme.Palette, searchQuery string, searchActive bool, searchInputView string) string {
	if width < 30 || height < 10 {
		return ""
	}

	halfHeight := (height - 3) / 2
	if halfHeight < 4 {
		halfHeight = 4
	}

	qLower := strings.ToLower(searchQuery)

	var filteredARP []ndk.ARPEntry
	for _, e := range state.ARPTables {
		if qLower != "" {
			match := strings.Contains(strings.ToLower(e.IPAddress), qLower) ||
				strings.Contains(strings.ToLower(e.MACAddress), qLower) ||
				strings.Contains(strings.ToLower(e.Interface), qLower) ||
				strings.Contains(strings.ToLower(e.NetInst), qLower) ||
				strings.Contains(strings.ToLower(e.EntryType), qLower)
			if !match {
				continue
			}
		}
		filteredARP = append(filteredARP, e)
	}

	var filteredMAC []ndk.MACTableEntry
	for _, e := range state.MACTables {
		if qLower != "" {
			match := strings.Contains(strings.ToLower(e.MACAddress), qLower) ||
				strings.Contains(strings.ToLower(e.NetInst), qLower) ||
				strings.Contains(strings.ToLower(e.Interface), qLower) ||
				strings.Contains(strings.ToLower(e.Type), qLower)
			if !match {
				continue
			}
		}
		filteredMAC = append(filteredMAC, e)
	}

	arpBox := v.renderARPTable(filteredARP, width-2, halfHeight, th, v.ActivePane == 0, searchQuery, searchActive, searchInputView)
	macBox := v.renderMACTable(filteredMAC, width-2, halfHeight, th, v.ActivePane == 1, searchQuery, searchActive, searchInputView)

	return lipgloss.JoinVertical(lipgloss.Left, arpBox, macBox)
}

func (v *ARPMACView) renderARPTable(entries []ndk.ARPEntry, width, height int, th theme.Palette, active bool, searchQuery string, searchActive bool, searchInputView string) string {
	borderColor := th.Muted
	titleColor := th.Secondary
	if active {
		borderColor = th.Primary
		titleColor = th.Primary
	}

	boxStyle := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		BorderBackground(th.Background).
		Background(th.Background)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(titleColor).
		Background(th.Background)

	hdrStr := headerStyle.Render(" 🔍 ARP NEIGHBOR TABLE ")
	if active && searchActive {
		hdrStr += lipgloss.NewStyle().Foreground(th.Warning).Background(th.Background).Bold(true).Render(fmt.Sprintf(" [SEARCH: %s]", searchInputView))
	} else if active {
		hdrStr += lipgloss.NewStyle().Foreground(th.Success).Background(th.Background).Render(" [ACTIVE PANE - Space to Switch, / to Search] ")
	} else {
		hdrStr += lipgloss.NewStyle().Foreground(th.Muted).Background(th.Background).Render(" [Space to Focus] ")
	}

	if searchQuery != "" && !searchActive {
		hdrStr += lipgloss.NewStyle().Foreground(th.Secondary).Background(th.Background).Render(fmt.Sprintf(" (Filter: '%s')", searchQuery))
	}

	// Exact Column Widths: IP (19), MAC (21), INTF (21), NETINST (15), TYPE (10)
	hdrTxt := fmt.Sprintf(" %s%s%s%s%s",
		padRight("IP ADDRESS", 19),
		padRight("MAC ADDRESS", 21),
		padRight("INTERFACE", 21),
		padRight("NET INSTANCE", 15),
		padRight("TYPE", 10),
	)
	colHdrs := lipgloss.NewStyle().Bold(true).Foreground(th.Primary).Background(th.Background).Render(hdrTxt)

	var rows []string
	rows = append(rows, hdrStr)
	rows = append(rows, colHdrs)
	rows = append(rows, lipgloss.NewStyle().Foreground(th.Muted).Background(th.Background).Render(" "+strings.Repeat("─", width-4)))

	maxVisible := height - 4
	if maxVisible < 1 {
		maxVisible = 1
	}

	if len(entries) == 0 {
		rows = append(rows, lipgloss.NewStyle().Foreground(th.Muted).Background(th.Background).Render("  No matching ARP neighbor entries discovered."))
	} else {
		if v.ARPScroll >= len(entries) {
			v.ARPScroll = len(entries) - 1
		}
		if v.ARPScroll < 0 {
			v.ARPScroll = 0
		}

		end := v.ARPScroll + maxVisible
		if end > len(entries) {
			end = len(entries)
		}

		for i := v.ARPScroll; i < end; i++ {
			e := entries[i]
			sIP := padRight(e.IPAddress, 19)
			sMAC := padRight(e.MACAddress, 21)
			sIntf := padRight(e.Interface, 21)
			sNet := padRight(e.NetInst, 15)
			sType := padRight(e.EntryType, 10)

			cIP := lipgloss.NewStyle().Foreground(th.Text).Background(th.Background).Render(sIP)
			cMAC := lipgloss.NewStyle().Foreground(th.Secondary).Background(th.Background).Render(sMAC)
			cIntf := lipgloss.NewStyle().Foreground(th.Warning).Background(th.Background).Render(sIntf)
			cNet := lipgloss.NewStyle().Foreground(th.Subtext).Background(th.Background).Render(sNet)
			cType := lipgloss.NewStyle().Foreground(th.Success).Background(th.Background).Render(sType)

			rows = append(rows, lipgloss.NewStyle().Background(th.Background).Render(fmt.Sprintf(" %s%s%s%s%s", cIP, cMAC, cIntf, cNet, cType)))
		}
	}

	return boxStyle.Render(strings.Join(rows, "\n"))
}

func (v *ARPMACView) renderMACTable(entries []ndk.MACTableEntry, width, height int, th theme.Palette, active bool, searchQuery string, searchActive bool, searchInputView string) string {
	borderColor := th.Muted
	titleColor := th.Secondary
	if active {
		borderColor = th.Primary
		titleColor = th.Primary
	}

	boxStyle := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		BorderBackground(th.Background).
		Background(th.Background)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(titleColor).
		Background(th.Background)

	hdrStr := headerStyle.Render(" 🏷️ MAC ADDRESS TABLE ")
	if active && searchActive {
		hdrStr += lipgloss.NewStyle().Foreground(th.Warning).Background(th.Background).Bold(true).Render(fmt.Sprintf(" [SEARCH: %s]", searchInputView))
	} else if active {
		hdrStr += lipgloss.NewStyle().Foreground(th.Success).Background(th.Background).Render(" [ACTIVE PANE - Space to Switch, / to Search] ")
	} else {
		hdrStr += lipgloss.NewStyle().Foreground(th.Muted).Background(th.Background).Render(" [Space to Focus] ")
	}

	if searchQuery != "" && !searchActive {
		hdrStr += lipgloss.NewStyle().Foreground(th.Secondary).Background(th.Background).Render(fmt.Sprintf(" (Filter: '%s')", searchQuery))
	}

	// Exact Column Widths: MAC (21), NETINST (15), DEST / VTEP (33), TYPE (10)
	hdrTxt := fmt.Sprintf(" %s%s%s%s",
		padRight("MAC ADDRESS", 21),
		padRight("NET INSTANCE", 15),
		padRight("DESTINATION / VTEP (IP:VNI)", 33),
		padRight("TYPE", 10),
	)
	colHdrs := lipgloss.NewStyle().Bold(true).Foreground(th.Primary).Background(th.Background).Render(hdrTxt)

	var rows []string
	rows = append(rows, hdrStr)
	rows = append(rows, colHdrs)
	rows = append(rows, lipgloss.NewStyle().Foreground(th.Muted).Background(th.Background).Render(" "+strings.Repeat("─", width-4)))

	maxVisible := height - 4
	if maxVisible < 1 {
		maxVisible = 1
	}

	if len(entries) == 0 {
		rows = append(rows, lipgloss.NewStyle().Foreground(th.Muted).Background(th.Background).Render("  No matching MAC address table entries learned."))
	} else {
		if v.MACScroll >= len(entries) {
			v.MACScroll = len(entries) - 1
		}
		if v.MACScroll < 0 {
			v.MACScroll = 0
		}

		end := v.MACScroll + maxVisible
		if end > len(entries) {
			end = len(entries)
		}

		for i := v.MACScroll; i < end; i++ {
			e := entries[i]
			destDisplay := e.Interface
			if e.VTEP != "" {
				destDisplay = fmt.Sprintf("%s (%s)", e.Interface, e.VTEP)
			}

			sMAC := padRight(e.MACAddress, 21)
			sNet := padRight(e.NetInst, 15)
			sIntf := padRight(destDisplay, 33)
			sType := padRight(e.Type, 10)

			cMAC := lipgloss.NewStyle().Foreground(th.Secondary).Background(th.Background).Bold(true).Render(sMAC)
			cNet := lipgloss.NewStyle().Foreground(th.Subtext).Background(th.Background).Render(sNet)
			cIntf := lipgloss.NewStyle().Foreground(th.Warning).Background(th.Background).Render(sIntf)
			cType := lipgloss.NewStyle().Foreground(th.Success).Background(th.Background).Render(sType)

			rows = append(rows, lipgloss.NewStyle().Background(th.Background).Render(fmt.Sprintf(" %s%s%s%s", cMAC, cNet, cIntf, cType)))
		}
	}

	return boxStyle.Render(strings.Join(rows, "\n"))
}
