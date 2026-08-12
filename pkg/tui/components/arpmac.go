package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"srl-tui/pkg/ndk"
	"srl-tui/pkg/tui/theme"
)

type ARPMACView struct {
	ActivePane   int // 0 = ARP Table, 1 = MAC Table
	ARPSelIdx    int
	ARPScrollOffset int
	MACSelIdx    int
	MACScrollOffset int
}

func NewARPMACView() *ARPMACView {
	return &ARPMACView{
		ActivePane:   0,
		ARPSelIdx:    0,
		ARPScrollOffset: 0,
		MACSelIdx:    0,
		MACScrollOffset: 0,
	}
}

func (v *ARPMACView) ScrollUp() {
	if v.ActivePane == 0 {
		if v.ARPSelIdx > 0 {
			v.ARPSelIdx--
		}
	} else {
		if v.MACSelIdx > 0 {
			v.MACSelIdx--
		}
	}
}

func (v *ARPMACView) ScrollDown(totalARP, totalMAC int) {
	if v.ActivePane == 0 {
		if totalARP > 0 && v.ARPSelIdx < totalARP-1 {
			v.ARPSelIdx++
		}
	} else {
		if totalMAC > 0 && v.MACSelIdx < totalMAC-1 {
			v.MACSelIdx++
		}
	}
}

func (v *ARPMACView) PageUp() {
	for i := 0; i < 10; i++ {
		v.ScrollUp()
	}
}

func (v *ARPMACView) PageDown(totalARP, totalMAC int) {
	for i := 0; i < 10; i++ {
		v.ScrollDown(totalARP, totalMAC)
	}
}

func (v *ARPMACView) TogglePane() {
	v.ActivePane = 1 - v.ActivePane
}

func GetFilteredARP(snap *ndk.TelemetryState, searchQuery string) []ndk.ARPEntry {
	var filtered []ndk.ARPEntry
	qLower := strings.ToLower(searchQuery)
	for _, e := range snap.ARPTables {
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
		filtered = append(filtered, e)
	}
	return filtered
}

func GetFilteredMAC(snap *ndk.TelemetryState, searchQuery string) []ndk.MACTableEntry {
	var filtered []ndk.MACTableEntry
	qLower := strings.ToLower(searchQuery)
	for _, e := range snap.MACTables {
		if qLower != "" {
			match := strings.Contains(strings.ToLower(e.MACAddress), qLower) ||
				strings.Contains(strings.ToLower(e.NetInst), qLower) ||
				strings.Contains(strings.ToLower(e.Interface), qLower) ||
				strings.Contains(strings.ToLower(e.Type), qLower) ||
				strings.Contains(strings.ToLower(e.VTEP), qLower) ||
				strings.Contains(fmt.Sprintf("%d", e.VNI), qLower)
			if !match {
				continue
			}
		}
		filtered = append(filtered, e)
	}
	return filtered
}

func (v *ARPMACView) Render(state *ndk.TelemetryState, width, height int, th theme.Palette, searchQuery string, searchActive bool, searchInputView string) string {
	if width < 30 || height < 10 {
		return ""
	}

	halfHeight := (height - 3) / 2
	if halfHeight < 4 {
		halfHeight = 4
	}

	filteredARP := GetFilteredARP(state, searchQuery)
	filteredMAC := GetFilteredMAC(state, searchQuery)

	// Clamp selections
	if v.ARPSelIdx >= len(filteredARP) {
		v.ARPSelIdx = len(filteredARP) - 1
	}
	if v.ARPSelIdx < 0 {
		v.ARPSelIdx = 0
	}

	if v.MACSelIdx >= len(filteredMAC) {
		v.MACSelIdx = len(filteredMAC) - 1
	}
	if v.MACSelIdx < 0 {
		v.MACSelIdx = 0
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
		currStr := "0/0"
		if len(entries) > 0 {
			currStr = fmt.Sprintf("%d/%d", v.ARPSelIdx+1, len(entries))
		}
		hdrStr += lipgloss.NewStyle().Foreground(th.Success).Background(th.Background).Render(fmt.Sprintf(" [ACTIVE PANE (%s) - ↑/↓ to scroll, ENTER for details, Space to Switch] ", currStr))
	} else {
		hdrStr += lipgloss.NewStyle().Foreground(th.Muted).Background(th.Background).Render(" [Space to Focus] ")
	}

	if searchQuery != "" && !searchActive {
		hdrStr += lipgloss.NewStyle().Foreground(th.Secondary).Background(th.Background).Render(fmt.Sprintf(" (Filter: '%s')", searchQuery))
	}

	// Dynamic Column Widths for ARP Table: Cursor (2) + IP (17) + MAC (20) + NETINST (14) + TYPE (14) + INTF (dynamic)
	ipWidth := 17
	macWidth := 20
	netWidth := 14
	typeWidth := 14

	intfWidth := width - (2 + ipWidth + macWidth + netWidth + typeWidth + 4)
	if intfWidth < 20 {
		intfWidth = 20
	}

	hdrTxt := fmt.Sprintf("  %s%s%s%s%s",
		padRight("IP ADDRESS", ipWidth),
		padRight("MAC ADDRESS", macWidth),
		padRight("NET INSTANCE", netWidth),
		padRight("TYPE", typeWidth),
		padRight("INTERFACE", intfWidth),
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
		if v.ARPSelIdx >= v.ARPScrollOffset+maxVisible {
			v.ARPScrollOffset = v.ARPSelIdx - maxVisible + 1
		}
		if v.ARPSelIdx < v.ARPScrollOffset {
			v.ARPScrollOffset = v.ARPSelIdx
		}

		end := v.ARPScrollOffset + maxVisible
		if end > len(entries) {
			end = len(entries)
		}

		for i := v.ARPScrollOffset; i < end; i++ {
			e := entries[i]
			sIP := padRight(e.IPAddress, ipWidth)
			sMAC := padRight(e.MACAddress, macWidth)
			sNet := padRight(e.NetInst, netWidth)
			sType := padRight(e.EntryType, typeWidth)
			sIntf := padRight(e.Interface, intfWidth)

			var row string
			if active && i == v.ARPSelIdx {
				rowStr := fmt.Sprintf("► %s%s%s%s%s", sIP, sMAC, sNet, sType, sIntf)
				row = lipgloss.NewStyle().Foreground(th.Background).Background(th.Highlight).Bold(true).Render(rowStr)
			} else {
				cIP := lipgloss.NewStyle().Foreground(th.Text).Background(th.Background).Render(sIP)
				cMAC := lipgloss.NewStyle().Foreground(th.Secondary).Background(th.Background).Render(sMAC)
				cNet := lipgloss.NewStyle().Foreground(th.Subtext).Background(th.Background).Render(sNet)
				cType := lipgloss.NewStyle().Foreground(th.Success).Background(th.Background).Render(sType)
				cIntf := lipgloss.NewStyle().Foreground(th.Warning).Background(th.Background).Render(sIntf)

				row = lipgloss.NewStyle().Background(th.Background).Render(fmt.Sprintf("  %s%s%s%s%s", cIP, cMAC, cNet, cType, cIntf))
			}
			rows = append(rows, row)
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
		currStr := "0/0"
		if len(entries) > 0 {
			currStr = fmt.Sprintf("%d/%d", v.MACSelIdx+1, len(entries))
		}
		hdrStr += lipgloss.NewStyle().Foreground(th.Success).Background(th.Background).Render(fmt.Sprintf(" [ACTIVE PANE (%s) - ↑/↓ to scroll, ENTER for details, Space to Switch] ", currStr))
	} else {
		hdrStr += lipgloss.NewStyle().Foreground(th.Muted).Background(th.Background).Render(" [Space to Focus] ")
	}

	if searchQuery != "" && !searchActive {
		hdrStr += lipgloss.NewStyle().Foreground(th.Secondary).Background(th.Background).Render(fmt.Sprintf(" (Filter: '%s')", searchQuery))
	}

	// Dynamic Column Widths for MAC Table: Cursor (2) + MAC (20) + NETINST (14) + TYPE (14) + DEST / VTEP (dynamic)
	macWidth := 20
	netWidth := 14
	typeWidth := 14

	destWidth := width - (2 + macWidth + netWidth + typeWidth + 4)
	if destWidth < 30 {
		destWidth = 30
	}

	hdrTxt := fmt.Sprintf("  %s%s%s%s",
		padRight("MAC ADDRESS", macWidth),
		padRight("NET INSTANCE", netWidth),
		padRight("TYPE", typeWidth),
		padRight("DESTINATION / VTEP (IP:VNI)", destWidth),
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
		if v.MACSelIdx >= v.MACScrollOffset+maxVisible {
			v.MACScrollOffset = v.MACSelIdx - maxVisible + 1
		}
		if v.MACSelIdx < v.MACScrollOffset {
			v.MACScrollOffset = v.MACSelIdx
		}

		end := v.MACScrollOffset + maxVisible
		if end > len(entries) {
			end = len(entries)
		}

		for i := v.MACScrollOffset; i < end; i++ {
			e := entries[i]
			destDisplay := e.Interface
			if e.VTEP != "" {
				if e.VNI > 0 {
					destDisplay = fmt.Sprintf("%s (%s:%d)", e.Interface, e.VTEP, e.VNI)
				} else {
					destDisplay = fmt.Sprintf("%s (%s)", e.Interface, e.VTEP)
				}
			} else if e.VNI > 0 {
				destDisplay = fmt.Sprintf("%s (VNI: %d)", e.Interface, e.VNI)
			}

			sMAC := padRight(e.MACAddress, macWidth)
			sNet := padRight(e.NetInst, netWidth)
			sType := padRight(e.Type, typeWidth)
			sIntf := padRight(destDisplay, destWidth)

			var row string
			if active && i == v.MACSelIdx {
				rowStr := fmt.Sprintf("► %s%s%s%s", sMAC, sNet, sType, sIntf)
				row = lipgloss.NewStyle().Foreground(th.Background).Background(th.Highlight).Bold(true).Render(rowStr)
			} else {
				cMAC := lipgloss.NewStyle().Foreground(th.Secondary).Background(th.Background).Bold(true).Render(sMAC)
				cNet := lipgloss.NewStyle().Foreground(th.Subtext).Background(th.Background).Render(sNet)
				cType := lipgloss.NewStyle().Foreground(th.Success).Background(th.Background).Render(sType)
				cIntf := lipgloss.NewStyle().Foreground(th.Warning).Background(th.Background).Render(sIntf)

				row = lipgloss.NewStyle().Background(th.Background).Render(fmt.Sprintf("  %s%s%s%s", cMAC, cNet, cType, cIntf))
			}
			rows = append(rows, row)
		}
	}

	return boxStyle.Render(strings.Join(rows, "\n"))
}

func RenderARPDetailModal(entry ndk.ARPEntry, snap *ndk.TelemetryState, th theme.Palette, width, height int) string {
	modalWidth := 72
	if width < modalWidth {
		modalWidth = width - 4
	}

	statusBadge := lipgloss.NewStyle().Bold(true).Foreground(th.Background).Background(th.Success).Padding(0, 1).Render("● ARP ADJACENCY ACTIVE / REACHABLE")

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(th.Secondary).Background(th.Background)

	expiryStr := "Active (300s)"
	if entry.ExpirySec > 0 {
		expiryStr = fmt.Sprintf("%ds remaining", entry.ExpirySec)
	}

	entryTypeStr := entry.EntryType
	if entryTypeStr == "" {
		entryTypeStr = "dynamic"
	}

	fmtRow := func(lbl string, val string) string {
		l := lipgloss.NewStyle().Foreground(th.Subtext).Background(th.Background).Render(padRight(lbl, 24))
		gap := lipgloss.NewStyle().Foreground(th.Muted).Background(th.Background).Render(" : ")
		v := lipgloss.NewStyle().Foreground(th.Text).Background(th.Background).Render(val)
		return lipgloss.NewStyle().Background(th.Background).Render("  " + l + gap + v)
	}

	// Subinterface / port physical oper state lookup
	portOperState := "Unknown / N/A"
	if snap != nil {
		for _, p := range snap.Ports {
			if strings.HasPrefix(entry.Interface, p.Name) || strings.HasPrefix(entry.Interface, p.ShortName) {
				portOperState = fmt.Sprintf("Admin: %s, Oper: %s", p.AdminState, p.OperState)
				break
			}
		}
	}

	lines := []string{
		titleStyle.Render("🔍 ARP NEIGHBOR DETAILS - " + entry.IPAddress),
		statusBadge,
		lipgloss.NewStyle().Foreground(th.Muted).Background(th.Background).Render(strings.Repeat("─", modalWidth-4)),
		fmtRow("IPv4 Address", entry.IPAddress),
		fmtRow("Hardware MAC Address", entry.MACAddress),
		fmtRow("Bound Interface", entry.Interface),
		fmtRow("Network Instance VRF", entry.NetInst),
		fmtRow("Entry Origin / Type", strings.ToUpper(entryTypeStr)),
		fmtRow("Cache Expiry Timer", expiryStr),
		fmtRow("Physical Link Status", portOperState),
		"",
		lipgloss.NewStyle().Foreground(th.Muted).Background(th.Background).Render("Press ESC or ENTER to close detail window"),
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(th.Secondary).
		BorderBackground(th.Background).
		Background(th.Background).
		Padding(1, 2).
		Width(modalWidth)

	rawContent := strings.Join(lines, "\n")
	contentWidth := modalWidth - 4
	r, g, b, _ := th.Background.RGBA()
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

func RenderMACDetailModal(entry ndk.MACTableEntry, snap *ndk.TelemetryState, th theme.Palette, width, height int) string {
	modalWidth := 72
	if width < modalWidth {
		modalWidth = width - 4
	}

	statusBadge := lipgloss.NewStyle().Bold(true).Foreground(th.Background).Background(th.Success).Padding(0, 1).Render("● MAC FORWARDING ENTRY ACTIVE")

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(th.Secondary).Background(th.Background)

	macTypeStr := entry.Type
	if macTypeStr == "" {
		macTypeStr = "learned"
	}

	vniStr := "Local L2 (No VNI)"
	if entry.VNI > 0 {
		vniStr = fmt.Sprintf("%d", entry.VNI)
	}

	vtepStr := "Local Port Attachment"
	if entry.VTEP != "" {
		vtepStr = fmt.Sprintf("VXLAN Tunnel End Point (%s)", entry.VTEP)
	}

	fmtRow := func(lbl string, val string) string {
		l := lipgloss.NewStyle().Foreground(th.Subtext).Background(th.Background).Render(padRight(lbl, 24))
		gap := lipgloss.NewStyle().Foreground(th.Muted).Background(th.Background).Render(" : ")
		v := lipgloss.NewStyle().Foreground(th.Text).Background(th.Background).Render(val)
		return lipgloss.NewStyle().Background(th.Background).Render("  " + l + gap + v)
	}

	lines := []string{
		titleStyle.Render("🏷️ MAC TABLE ENTRY DETAILS - " + entry.MACAddress),
		statusBadge,
		lipgloss.NewStyle().Foreground(th.Muted).Background(th.Background).Render(strings.Repeat("─", modalWidth-4)),
		fmtRow("Hardware MAC Address", entry.MACAddress),
		fmtRow("Network Instance VRF", entry.NetInst),
		fmtRow("Destination Interface", entry.Interface),
		fmtRow("MAC Learning Type", strings.ToUpper(macTypeStr)),
		fmtRow("VXLAN VNI", vniStr),
		fmtRow("Remote Destination", vtepStr),
		"",
		lipgloss.NewStyle().Foreground(th.Muted).Background(th.Background).Render("Press ESC or ENTER to close detail window"),
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(th.Secondary).
		BorderBackground(th.Background).
		Background(th.Background).
		Padding(1, 2).
		Width(modalWidth)

	rawContent := strings.Join(lines, "\n")
	contentWidth := modalWidth - 4
	r, g, b, _ := th.Background.RGBA()
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
