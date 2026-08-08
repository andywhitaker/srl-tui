package components

import (
	"fmt"
	"math"

	"github.com/charmbracelet/lipgloss"
	"srl-tui/pkg/ndk"
	"srl-tui/pkg/tui/theme"
)

func formatThroughput(bps float64) string {
	if bps < 1000.0 {
		return fmt.Sprintf("%.0f bps", bps)
	} else if bps < 1000000.0 {
		return fmt.Sprintf("%.1f Kbps", bps/1000.0)
	} else if bps < 1000000000.0 {
		return fmt.Sprintf("%.1f Mbps", bps/1000000.0)
	}
	return fmt.Sprintf("%.2f Gbps", bps/1000000000.0)
}

func getScaleUnit(peakBps float64) (string, float64) {
	if peakBps < 1000.0 {
		return "bps", 1.0
	} else if peakBps < 1000000.0 {
		return "Kbps", 1000.0
	} else if peakBps < 1000000000.0 {
		return "Mbps", 1000000.0
	}
	return "Gbps", 1000000000.0
}

func RenderBrailleChart(snap *ndk.TelemetryState, focused bool, pal theme.Palette, width, height int) string {
	if width < 40 {
		width = 40
	}

	chartWidth := width - 18
	if chartWidth < 20 {
		chartWidth = 20
	}

	chartHeight := height - 6
	if chartHeight < 4 {
		chartHeight = 4
	}

	// 1. Calculate Peak Traffic for Dynamic Peak Auto-Scaling
	peakIng := 0.0
	for _, v := range snap.IngressHistory {
		if v > peakIng {
			peakIng = v
		}
	}
	peakEg := 0.0
	for _, v := range snap.EgressHistory {
		if v > peakEg {
			peakEg = v
		}
	}

	overallPeak := peakIng
	if peakEg > overallPeak {
		overallPeak = peakEg
	}
	if overallPeak <= 0 {
		overallPeak = 1000.0 // Default floor: 1 Kbps
	}

	// Dynamic Unit Scaling (bps -> Kbps -> Mbps -> Gbps)
	unitName, unitFactor := getScaleUnit(overallPeak)
	maxScaleVal := math.Ceil(overallPeak/unitFactor*1.1) * unitFactor

	// 2. Draw Dual Braille Waveform with Normalized Dynamic Scale
	canvas := drawDualBrailleWaveform(snap.IngressHistory, snap.EgressHistory, maxScaleVal, chartWidth, chartHeight)

	ingStyle := lipgloss.NewStyle().Foreground(pal.GraphIngress).Bold(true)
	egStyle := lipgloss.NewStyle().Foreground(pal.GraphEgress).Bold(true)
	axisStyle := lipgloss.NewStyle().Foreground(pal.Subtext)

	// Combine Y-axis labels with Braille canvas
	var chartLines []string
	for r := 0; r < chartHeight; r++ {
		yVal := maxScaleVal * (1.0 - float64(r)/float64(chartHeight))
		yLabel := fmt.Sprintf("%7.1f %-4s", yVal/unitFactor, unitName)
		if r == chartHeight-1 {
			yLabel = fmt.Sprintf("%7.0f %-4s", 0.0, unitName)
		}

		lineStr := ingStyle.Render(canvas[r])
		chartLines = append(chartLines, fmt.Sprintf("%s │ %s", axisStyle.Render(yLabel), lineStr))
	}

	chartStr := lipgloss.JoinVertical(lipgloss.Left, chartLines...)

	// Current & Peak Metrics
	currIng := 0.0
	if len(snap.IngressHistory) > 0 {
		currIng = snap.IngressHistory[len(snap.IngressHistory)-1]
	}
	currEg := 0.0
	if len(snap.EgressHistory) > 0 {
		currEg = snap.EgressHistory[len(snap.EgressHistory)-1]
	}

	legendIng := ingStyle.Render(fmt.Sprintf("━━ INGRESS (Cyan): %s [PEAK: %s]", formatThroughput(currIng), formatThroughput(peakIng)))
	legendEg := egStyle.Render(fmt.Sprintf("━━ EGRESS (Pink):  %s [PEAK: %s]", formatThroughput(currEg), formatThroughput(peakEg)))

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(pal.Primary)
	subStyle := lipgloss.NewStyle().Foreground(pal.Subtext)

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
		titleStyle.Render("📊 PORT THROUGHPUT MONITOR"),
		subStyle.Render("[Dynamic Peak Auto-Scaling Waveform: bps → Kbps → Mbps → Gbps]"),
	)

	return panelStyle.Render(header + "\n" + legendIng + "  " + legendEg + "\n" + chartStr)
}

func drawDualBrailleWaveform(ingData, egData []float64, maxScale float64, charWidth, charHeight int) []string {
	if charWidth <= 0 {
		charWidth = 40
	}
	if charHeight <= 0 {
		charHeight = 4
	}

	dotWidth := charWidth * 2
	dotHeight := charHeight * 4

	grid := make([][]bool, dotHeight)
	for r := range grid {
		grid[r] = make([]bool, dotWidth)
	}

	// Map Ingress Data scaled to maxScale
	plotDatasetScaled(grid, ingData, maxScale, dotWidth, dotHeight)

	// Map Egress Data scaled to maxScale
	plotDatasetScaled(grid, egData, maxScale, dotWidth, dotHeight)

	// Convert grid to Braille characters
	lines := make([]string, charHeight)
	for row := 0; row < charHeight; row++ {
		var lineRunes []rune
		for col := 0; col < charWidth; col++ {
			x0 := col * 2
			y0 := row * 4

			var charVal rune = 0x2800

			// Dot mapping
			if getGrid(grid, y0+0, x0+0) { charVal |= 0x01 }
			if getGrid(grid, y0+1, x0+0) { charVal |= 0x02 }
			if getGrid(grid, y0+2, x0+0) { charVal |= 0x04 }
			if getGrid(grid, y0+0, x0+1) { charVal |= 0x08 }
			if getGrid(grid, y0+1, x0+1) { charVal |= 0x10 }
			if getGrid(grid, y0+2, x0+1) { charVal |= 0x20 }
			if getGrid(grid, y0+3, x0+0) { charVal |= 0x40 }
			if getGrid(grid, y0+3, x0+1) { charVal |= 0x80 }

			lineRunes = append(lineRunes, charVal)
		}
		lines[row] = string(lineRunes)
	}

	return lines
}

func plotDatasetScaled(grid [][]bool, data []float64, maxScale float64, dotWidth, dotHeight int) {
	if len(data) == 0 || maxScale <= 0 {
		return
	}

	dataLen := len(data)
	for x := 0; x < dotWidth; x++ {
		idx := (x * dataLen) / dotWidth
		if idx >= dataLen {
			idx = dataLen - 1
		}

		val := data[idx]
		norm := val / maxScale
		if norm > 1.0 {
			norm = 1.0
		}
		if norm < 0.0 {
			norm = 0.0
		}

		y := int(math.Round(norm * float64(dotHeight-1)))
		gridY := (dotHeight - 1) - y
		if gridY >= 0 && gridY < dotHeight {
			grid[gridY][x] = true
		}
	}
}

func getGrid(grid [][]bool, r, c int) bool {
	if r >= 0 && r < len(grid) && c >= 0 && c < len(grid[r]) {
		return grid[r][c]
	}
	return false
}
