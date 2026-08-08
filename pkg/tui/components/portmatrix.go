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
		Foreground(pal.Primary)

	subStyle := lipgloss.NewStyle().
		Foreground(pal.Subtext)

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

		blockStr := fmt.Sprintf("%s  (ODD: %d..%d)\n%s  (EVEN: %d..%d)",
			oddRow, firstOdd, lastOdd,
			evenRow, firstEven, lastEven)

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
		Background(pal.Surface).
		Width(width - 2).
		Height(height - 2).
		Padding(0, 1)

	header := fmt.Sprintf("%s %s",
		titleStyle.Render(fmt.Sprintf("█ FRONT-PANEL PORT MATRIX (%d Physical Ethernet Ports)", numPorts)),
		subStyle.Render("[Use Arrow Keys/hjkl to navigate | Press ENTER/SPACE for YANG State]"),
	)

	return panelStyle.Render(header + "\n\n" + gridStr + "\n\n" + inspectorStr)
}

func renderPortCell(p ndk.PortState, isSelected bool, pal theme.Palette) string {
	numLabel := fmt.Sprintf("%02d", p.Index+1)

	var style lipgloss.Style

	if isSelected {
		style = lipgloss.NewStyle().
			Bold(true).
			Foreground(pal.Background).
			Background(pal.Primary).
			Padding(0, 0)
		return style.Render(fmt.Sprintf("[%s]", numLabel))
	}

	adminUp := strings.ToLower(p.AdminState) == "up"
	operUp := strings.ToLower(p.OperState) == "up"

	if !adminUp {
		// When admin-state is down: grey color (pal.Muted)
		style = lipgloss.NewStyle().
			Foreground(pal.Muted).
			Background(pal.Background)
	} else if operUp {
		// When admin-state is up and oper-state is up: green (pal.Success)
		style = lipgloss.NewStyle().
			Bold(true).
			Foreground(pal.Success).
			Background(pal.Background)
	} else {
		// When admin-state is up and oper-state is down: red (pal.Error)
		style = lipgloss.NewStyle().
			Bold(true).
			Foreground(pal.Error).
			Background(pal.Background)
	}

	return style.Render(fmt.Sprintf("[%s]", numLabel))
}

func renderPortInspector(p ndk.PortState, pal theme.Palette, width int) string {
	labelStyle := lipgloss.NewStyle().Foreground(pal.Subtext).Bold(true)
	valStyle := lipgloss.NewStyle().Foreground(pal.Text)
	highlightStyle := lipgloss.NewStyle().Foreground(pal.Primary).Bold(true)

	statusStr := lipgloss.NewStyle().Foreground(pal.Error).Render("DOWN")
	if p.OperState == "up" {
		statusStr = lipgloss.NewStyle().Foreground(pal.Success).Render("UP")
	}

	rxFormatted := lipgloss.NewStyle().Foreground(pal.Success).Bold(true).Render(formatBps(p.RxBps))
	txFormatted := lipgloss.NewStyle().Foreground(pal.Secondary).Bold(true).Render(formatBps(p.TxBps))
	utilFormatted := fmt.Sprintf("%.2f%%", p.UtilPercent)
	if p.OperState != "up" || p.AdminState == "down" {
		rxFormatted = lipgloss.NewStyle().Foreground(pal.Muted).Render("0 bps")
		txFormatted = lipgloss.NewStyle().Foreground(pal.Muted).Render("0 bps")
		utilFormatted = "0.00%"
	}

	desc := p.Description
	if desc == "" {
		desc = "N/A"
	}

	col1 := fmt.Sprintf("%s %s  %s %s  %s %s",
		labelStyle.Render("Port:"), highlightStyle.Render(p.Name),
		labelStyle.Render("State:"), statusStr,
		labelStyle.Render("Speed:"), valStyle.Render(p.Speed),
	)

	col2 := fmt.Sprintf("%s %s  %s %s  %s %s  %s %d",
		labelStyle.Render("Rx Traffic:"), rxFormatted,
		labelStyle.Render("Tx Traffic:"), txFormatted,
		labelStyle.Render("Util:"), valStyle.Render(utilFormatted),
		labelStyle.Render("MTU:"), p.MTU,
	)

	col3 := fmt.Sprintf("%s %s  %s %d  %s %d",
		labelStyle.Render("Desc:"), valStyle.Render(desc),
		labelStyle.Render("Flaps:"), p.Flaps,
		labelStyle.Render("Errors:"), p.Errors,
	)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(pal.Muted).
		Background(pal.Background).
		Width(width).
		Padding(0, 1)

	return boxStyle.Render(col1 + "\n" + col2 + "\n" + col3)
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
