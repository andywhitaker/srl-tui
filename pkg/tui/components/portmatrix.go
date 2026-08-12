package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"srl-tui/pkg/ndk"
	"srl-tui/pkg/tui/theme"
)

func RenderPortMatrix(snap *ndk.TelemetryState, selectedIdx int, focused bool, pal theme.Palette, width, height int) string {
	if width < 50 {
		width = 50
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(pal.Primary).
		Background(pal.Background)

	subStyle := lipgloss.NewStyle().
		Foreground(pal.Subtext).
		Background(pal.Background)

	numPorts := len(snap.Ports)
	if numPorts == 0 {
		numPorts = 16
	}

	// Calculate ports per row dynamically (each port cell is 4 chars wide)
	portsPerBlockRow := (width - 24) / 4
	if portsPerBlockRow < 8 {
		portsPerBlockRow = 8
	}
	if portsPerBlockRow > 24 {
		portsPerBlockRow = 24
	}
	blockSize := portsPerBlockRow * 2

	var blockStrings []string
	for bStart := 0; bStart < numPorts; bStart += blockSize {
		bEnd := bStart + blockSize
		if bEnd > numPorts {
			bEnd = numPorts
		}

		var oddRowCells []string
		var evenRowCells []string

		for i := bStart; i < bEnd; i++ {
			p := snap.Ports[i]
			cellStr := renderPortCell(p, i == selectedIdx, pal)

			if i%2 == 0 {
				oddRowCells = append(oddRowCells, cellStr)
			} else {
				evenRowCells = append(evenRowCells, cellStr)
			}
		}

		oddRow := lipgloss.JoinHorizontal(lipgloss.Top, oddRowCells...)
		evenRow := lipgloss.JoinHorizontal(lipgloss.Top, evenRowCells...)

		firstOdd := bStart + 1
		lastOdd := bEnd - 1
		if bEnd%2 == 1 {
			lastOdd = bEnd
		}

		firstEven := bStart + 2
		lastEven := bEnd
		if bEnd%2 == 1 {
			lastEven = bEnd - 1
		}

		lblOdd := lipgloss.NewStyle().Foreground(pal.Subtext).Background(pal.Background).Render(fmt.Sprintf("  (ODD: %d..%d)", firstOdd, lastOdd))
		lblEven := lipgloss.NewStyle().Foreground(pal.Subtext).Background(pal.Background).Render(fmt.Sprintf("  (EVEN: %d..%d)", firstEven, lastEven))

		blockStr := fmt.Sprintf("%s%s\n%s%s", oddRow, lblOdd, evenRow, lblEven)

		blockStrings = append(blockStrings, blockStr)
	}

	gridStr := strings.Join(blockStrings, "\n\n")

	// Split Pane Inspector for Selected Port
	var inspectorStr string
	if selectedIdx >= 0 && selectedIdx < len(snap.Ports) {
		selPort := snap.Ports[selectedIdx]
		inspectorStr = renderPortInspector(selPort, pal, width-6)
	}

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

	header := lipgloss.NewStyle().Background(pal.Background).Render(fmt.Sprintf("%s %s",
		titleStyle.Render(fmt.Sprintf("█ FRONT-PANEL PORT MATRIX (%d Physical Ethernet Ports)", numPorts)),
		subStyle.Render("[Use Arrow Keys/hjkl to navigate | Press ENTER/SPACE for YANG State]"),
	))

	return panelStyle.Render(header + "\n\n" + gridStr + "\n\n" + inspectorStr)
}

func renderPortCell(p ndk.PortState, isSelected bool, pal theme.Palette) string {
	numLabel := fmt.Sprintf("%02d", p.Index+1)

	if isSelected {
		style := lipgloss.NewStyle().
			Bold(true).
			Foreground(pal.Background).
			Background(pal.Primary).
			Padding(0, 0)
		return style.Render(fmt.Sprintf("[%s]", numLabel))
	}

	adminUp := strings.ToLower(p.AdminState) == "up"
	operUp := strings.ToLower(p.OperState) == "up"

	style := lipgloss.NewStyle().Background(pal.Background)

	if !adminUp {
		// When admin-state is down: grey color (pal.Muted)
		style = style.Foreground(pal.Muted)
	} else if operUp {
		// When admin-state is up and oper-state is up: green (pal.Success)
		style = style.Bold(true).Foreground(pal.Success)
	} else {
		// When admin-state is up and oper-state is down: red (pal.Error)
		style = style.Bold(true).Foreground(pal.Error)
	}

	return style.Render(fmt.Sprintf("[%s]", numLabel))
}

func renderPortInspector(p ndk.PortState, pal theme.Palette, width int) string {
	lbl := lipgloss.NewStyle().Foreground(pal.Subtext).Background(pal.Background).Bold(true)
	val := lipgloss.NewStyle().Foreground(pal.Text).Background(pal.Background)
	hl := lipgloss.NewStyle().Foreground(pal.Primary).Background(pal.Background).Bold(true)

	statusStr := lipgloss.NewStyle().Foreground(pal.Error).Background(pal.Background).Bold(true).Render(padRight("DOWN", 6))
	if p.OperState == "up" {
		statusStr = lipgloss.NewStyle().Foreground(pal.Success).Background(pal.Background).Bold(true).Render(padRight("UP", 6))
	}

	rxFormatted := lipgloss.NewStyle().Foreground(pal.Success).Background(pal.Background).Bold(true).Render(padRight(formatBps(p.RxBps), 12))
	txFormatted := lipgloss.NewStyle().Foreground(pal.Secondary).Background(pal.Background).Bold(true).Render(padRight(formatBps(p.TxBps), 12))
	utilFormatted := fmt.Sprintf("%.2f%%", p.UtilPercent)
	if p.OperState != "up" || p.AdminState == "down" {
		rxFormatted = lipgloss.NewStyle().Foreground(pal.Muted).Background(pal.Background).Render(padRight("0 bps", 12))
		txFormatted = lipgloss.NewStyle().Foreground(pal.Muted).Background(pal.Background).Render(padRight("0 bps", 12))
		utilFormatted = "0.00%"
	}

	desc := p.Description
	if desc == "" {
		desc = "N/A"
	}

	c1 := lipgloss.JoinHorizontal(lipgloss.Top,
		lbl.Render("Port: "), hl.Render(padRight(p.Name, 16)),
		lbl.Render("State: "), statusStr,
		lbl.Render("Speed: "), val.Render(p.Speed),
	)

	c2 := lipgloss.JoinHorizontal(lipgloss.Top,
		lbl.Render("Rx Traffic: "), rxFormatted,
		lbl.Render("Tx Traffic: "), txFormatted,
		lbl.Render("Util: "), val.Render(padRight(utilFormatted, 8)),
		lbl.Render("MTU: "), val.Render(fmt.Sprintf("%d", p.MTU)),
	)

	c3 := lipgloss.JoinHorizontal(lipgloss.Top,
		lbl.Render("Desc: "), val.Render(padRight(desc, 24)),
		lbl.Render("Flaps: "), val.Render(padRight(fmt.Sprintf("%d", p.Flaps), 6)),
		lbl.Render("Errors: "), val.Render(fmt.Sprintf("%d", p.Errors)),
	)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(pal.Muted).
		BorderBackground(pal.Background).
		Background(pal.Background).
		Width(width).
		Padding(0, 1)

	rawContent := c1 + "\n" + c2 + "\n" + c3
	var styledLines []string
	bgStyle := lipgloss.NewStyle().Background(pal.Background)
	for _, l := range strings.Split(rawContent, "\n") {
		styledLines = append(styledLines, bgStyle.Render(l))
	}
	return boxStyle.Render(strings.Join(styledLines, "\n"))
}

func formatBps(bps float64) string {
	if bps >= 1e9 {
		return fmt.Sprintf("%.2f Gbps", bps/1e9)
	} else if bps >= 1e6 {
		return fmt.Sprintf("%.2f Mbps", bps/1e6)
	} else if bps >= 1e3 {
		return fmt.Sprintf("%.2f Kbps", bps/1e3)
	}
	return fmt.Sprintf("%.0f bps", bps)
}
