package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"srl-tui/pkg/ndk"
	"srl-tui/pkg/tui/theme"
)

func RenderHeader(snap *ndk.TelemetryState, pal theme.Palette, width int) string {
	if width < 40 {
		width = 40
	}

	// Pulse Heartbeat Ticker
	heartbeatChars := []string{"♥", "⚡", "●", "◈", "◆", "★"}
	heartbeat := heartbeatChars[snap.TickCount%uint64(len(heartbeatChars))]

	heartbeatStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(pal.Secondary)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(pal.Primary).
		Background(pal.Surface).
		Padding(0, 1)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(pal.Subtext)

	statusConnectedStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(pal.Success)

	statusSyncingStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(pal.Warning)

	statusDisconnectedStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(pal.Error)

	// Animated spinner characters for sync state
	spinners := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spinnerChar := spinners[snap.TickCount%uint64(len(spinners))]

	var connStatus string
	if snap.SyncState == "INITIALIZING" {
		connStatus = statusSyncingStyle.Render(fmt.Sprintf("%s INITIALIZING... [%s]", spinnerChar, snap.SyncMessage))
	} else if snap.SyncState == "SYNCING" {
		connStatus = statusSyncingStyle.Render(fmt.Sprintf("%s SYNCING... [%s]", spinnerChar, snap.SyncMessage))
	} else if snap.NDKConnected {
		modeStr := "SR Linux State Stream"
		if snap.DemoMode {
			modeStr = "DEMO SIM"
		}
		connStatus = statusConnectedStyle.Render(fmt.Sprintf("● CONNECTED [%s]", modeStr))
	} else {
		connStatus = statusDisconnectedStyle.Render("● DISCONNECTED")
	}

	themeBadge := lipgloss.NewStyle().
		Bold(true).
		Foreground(pal.Background).
		Background(pal.Primary).
		Padding(0, 1).
		Render(pal.Name)

	// CPU & RAM Gauge Bars
	cpuGauge := renderGaugeBar("CPU", snap.CPUUsage, pal, 10)
	ramGauge := renderGaugeBar("RAM", snap.RAMUsage, pal, 10)

	// Format Switch Uptime cleanly (days, hours, minutes, seconds)
	days := int(snap.Uptime.Hours()) / 24
	hours := int(snap.Uptime.Hours()) % 24
	mins := int(snap.Uptime.Minutes()) % 60
	secs := int(snap.Uptime.Seconds()) % 60

	var uptimeStr string
	if days > 0 {
		uptimeStr = fmt.Sprintf("%dd %02dh:%02dm:%02ds", days, hours, mins, secs)
	} else {
		uptimeStr = fmt.Sprintf("%02dh:%02dm:%02ds", hours, mins, secs)
	}

	nokiaBadge := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00F5FF")).
		Background(pal.Surface).
		Padding(0, 1).
		Render("NOKIA")

	line1 := fmt.Sprintf("%s %s %s %s  %s  %s",
		nokiaBadge,
		titleStyle.Render("⚡ SRL-NDK CYBER-DASHBOARD v1.0"),
		heartbeatStyle.Render(heartbeat),
		subtitleStyle.Render(fmt.Sprintf("[%s | %s]", snap.Hostname, snap.Platform)),
		connStatus,
		themeBadge,
	)

	line2 := fmt.Sprintf("%s | Uptime: %s | Events: %d (%.0f/s) | %s  %s",
		subtitleStyle.Render("SR Linux "+snap.OSVersion),
		uptimeStr,
		snap.EventCount,
		snap.EventsPerSec,
		cpuGauge,
		ramGauge,
	)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(pal.Primary).
		Background(pal.Surface).
		Width(width - 2).
		Padding(0, 1)

	return boxStyle.Render(line1 + "\n" + line2)
}

func renderGaugeBar(label string, percent float64, pal theme.Palette, length int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filledLen := int((percent / 100.0) * float64(length))
	if filledLen > length {
		filledLen = length
	}

	color := pal.Success
	if percent > 70 {
		color = pal.Warning
	}
	if percent > 90 {
		color = pal.Error
	}

	filledStr := strings.Repeat("█", filledLen)
	emptyStr := strings.Repeat("░", length-filledLen)

	barStyle := lipgloss.NewStyle().Foreground(color)
	emptyStyle := lipgloss.NewStyle().Foreground(pal.Muted)
	labelStyle := lipgloss.NewStyle().Foreground(pal.Subtext).Bold(true)

	return fmt.Sprintf("%s:[%s%s %3.0f%%]",
		labelStyle.Render(label),
		barStyle.Render(filledStr),
		emptyStyle.Render(emptyStr),
		percent,
	)
}
