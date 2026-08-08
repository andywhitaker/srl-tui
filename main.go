package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"srl-tui/pkg/ndk"
	"srl-tui/pkg/tui"
	"srl-tui/pkg/tui/theme"
)

func main() {
	// Force 24-bit TrueColor profile so vibrant Cyberpunk/Synthwave/Matrix/Monokai colors ALWAYS render cleanly
	lipgloss.SetColorProfile(termenv.TrueColor)

	demoFlag := flag.Bool("demo", false, "Run in simulated Cyberpunk demo mode")
	themeFlag := flag.String("theme", "cyberpunk", "Color theme: cyberpunk, synthwave, matrix, monokai, cobalt2, solarized")
	socketFlag := flag.String("socket", "unix:///opt/srlinux/var/run/sr_sdk_service_manager:50053", "NDK Unix Socket Path")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Shared State
	state := ndk.NewTelemetryState(32)

	initialTheme := theme.Cyberpunk
	switch strings.ToLower(*themeFlag) {
	case "synthwave":
		initialTheme = theme.Synthwave
	case "matrix":
		initialTheme = theme.Matrix
	case "monokai":
		initialTheme = theme.Monokai
	case "cobalt2":
		initialTheme = theme.Cobalt2
	case "solarized", "solarized-dark", "solarized_dark":
		initialTheme = theme.SolarizedDark
	}

	if *demoFlag {
		// Run Demo Simulation mode explicitly requested by user
		sim := ndk.NewSimulator(state)
		go sim.Start()
	} else {
		// Native NDK gRPC Event Subscription & Pure Go HTTP JSON-RPC Datastore Mode (No sr_cli subprocesses!)
		ndkClient := ndk.NewNDKClient(*socketFlag, state)
		_ = ndkClient.Start(ctx)
		defer ndkClient.Stop()
	}

	model := tui.NewModel(ctx, state, initialTheme)
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running SRL NDK Cyberpunk TUI: %v\n", err)
		os.Exit(1)
	}
}
