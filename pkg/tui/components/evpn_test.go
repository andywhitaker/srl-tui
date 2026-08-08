package components

import (
	"strings"
	"testing"

	"srl-tui/pkg/ndk"
	"srl-tui/pkg/tui/theme"
)

func TestEVPNRouteStatusLabels(t *testing.T) {
	state := ndk.NewTelemetryState(32)
	state.EVPNRoutes = []ndk.EVPNRouteEntry{
		{
			RouteType: 2,
			RD:        "1.1.1.1:10000",
			MAC:       "00:11:22:33:44:55",
			IP:        "192.168.10.1",
			NextHop:   "10.0.0.1",
			Neighbor:  "10.0.0.1",
			Status:    "u*>",
			VNI:       "10000",
		},
		{
			RouteType: 5,
			RD:        "2.2.2.2:10020",
			Prefix:    "10.200.1.0/24",
			NextHop:   "10.0.0.2",
			Neighbor:  "10.0.0.2",
			Status:    "r*",
			VNI:       "10020",
		},
	}

	pal := theme.Cyberpunk
	output := RenderEVPNView(state.Snapshot(), EVPNFilterAll, 0, pal, 120, 30, "", false, "", true)

	if strings.Contains(output, "u*> (IMP)") {
		t.Errorf("BUG DETECTED: RenderEVPNView formatted imported route as 'u*> (IMP)' instead of 'u*>'!")
	}

	if !strings.Contains(output, "u*>") {
		t.Errorf("Expected 'u*>' for imported route status")
	}

	if !strings.Contains(output, "r* (UNIMP)") {
		t.Errorf("Expected 'r* (UNIMP)' for unimported route status")
	}
}

func TestEVPNFullRowHighlightRendering(t *testing.T) {
	state := ndk.NewTelemetryState(16)
	state.EVPNRoutes = []ndk.EVPNRouteEntry{
		{RouteType: 2, RD: "1.1.1.1:10010", MAC: "AA:BB:CC:DD:EE:FF", IP: "10.0.0.1", NextHop: "1.1.1.1", Status: "u*>"},
		{RouteType: 3, RD: "2.2.2.2:10020", VNI: "10020", NextHop: "2.2.2.2", Status: "u*>"},
	}
	pal := theme.Cyberpunk

	rendered := RenderEVPNView(state, EVPNFilterAll, 0, pal, 140, 30, "", false, "", true)
	if !strings.Contains(rendered, "TYPE-2") || !strings.Contains(rendered, "AA:BB:CC:DD:EE:FF") {
		t.Fatalf("EVPN view failed to render selected route 0: %s", rendered)
	}

	rendered2 := RenderEVPNView(state, EVPNFilterAll, 1, pal, 140, 30, "", false, "", true)
	if !strings.Contains(rendered2, "TYPE-3") || !strings.Contains(rendered2, "Multicast BUM Ingress") {
		t.Fatalf("EVPN view failed to render selected route 1: %s", rendered2)
	}
}

func TestEVPNSearchFilterSelectionReset(t *testing.T) {
	state := ndk.NewTelemetryState(16)
	state.EVPNRoutes = []ndk.EVPNRouteEntry{
		{RouteType: 2, RD: "1.1.1.1:10010", MAC: "AA:BB:CC:DD:EE:FF", IP: "10.0.0.1", NextHop: "1.1.1.1", Status: "u*>"},
		{RouteType: 3, RD: "2.2.2.2:10020", VNI: "10020", NextHop: "2.2.2.2", Status: "u*>"},
	}
	pal := theme.Cyberpunk

	// Pass out-of-bounds index (e.g. index 45) to RenderEVPNView with a search filter
	rendered := RenderEVPNView(state, EVPNFilterAll, 45, pal, 140, 30, "10020", false, "", true)
	// Should auto-clamp to index 0 (the only matching item for 10020)
	if !strings.Contains(rendered, "TYPE-3") || !strings.Contains(rendered, "2.2.2.2:10020") {
		t.Fatalf("Expected auto-clamping to matching filtered item, got: %s", rendered)
	}
}

func TestEVPNType1ADLabel(t *testing.T) {
	state := ndk.NewTelemetryState(16)
	state.EVPNRoutes = []ndk.EVPNRouteEntry{
		{RouteType: 1, RD: "1.1.1.1:100", ESI: "00:01:02:03:04:05:06:07:08:09", NextHop: "1.1.1.1", Status: "u*>"},
	}
	pal := theme.Cyberpunk

	rendered := RenderEVPNView(state, EVPNFilterType1, 0, pal, 140, 30, "", false, "", true)
	if !strings.Contains(rendered, "TYPE 1 (AD)") {
		t.Fatalf("Expected 'TYPE 1 (AD)' filter tab label, got: %s", rendered)
	}

	detail := RenderEVPNDetailModal(state.EVPNRoutes[0], pal, 140, 30, state)
	if !strings.Contains(detail, "Auto-Discovery (AD)") {
		t.Fatalf("Expected 'Auto-Discovery (AD)' in detail title, got: %s", detail)
	}
}

func TestEVPNUnimportedRouteToggleFilter(t *testing.T) {
	state := ndk.NewTelemetryState(16)
	state.EVPNRoutes = []ndk.EVPNRouteEntry{
		{RouteType: 2, RD: "1.1.1.1:10010", MAC: "AA:BB:CC:DD:EE:FF", IP: "10.0.0.1", NextHop: "1.1.1.1", Status: "u*>"},
		{RouteType: 5, RD: "2.2.2.2:10020", Prefix: "10.250.0.0/24", NextHop: "2.2.2.2", Status: "r*"},
	}
	pal := theme.Cyberpunk

	// 1. By default (showUnimported = false), unimported route (10.250.0.0/24) MUST be hidden
	hiddenView := RenderEVPNView(state, EVPNFilterAll, 0, pal, 140, 30, "", false, "", false)
	if strings.Contains(hiddenView, "10.250.0.0/24") {
		t.Fatalf("Expected unimported route 10.250.0.0/24 to be HIDDEN by default, but found in view")
	}
	if !strings.Contains(hiddenView, "[u] UNIMPORTED: HIDDEN") {
		t.Fatalf("Expected '[u] UNIMPORTED: HIDDEN' badge in header, got: %s", hiddenView)
	}

	// 2. When toggled ON (showUnimported = true), unimported route MUST be visible
	shownView := RenderEVPNView(state, EVPNFilterAll, 0, pal, 140, 30, "", false, "", true)
	if !strings.Contains(shownView, "10.250.0.0/24") {
		t.Fatalf("Expected unimported route 10.250.0.0/24 to be SHOWN when toggled ON")
	}
	if !strings.Contains(shownView, "[u] UNIMPORTED: SHOWN") {
		t.Fatalf("Expected '[u] UNIMPORTED: SHOWN' badge in header, got: %s", shownView)
	}
}

func TestEVPNMACIPColumnFullWidth(t *testing.T) {
	state := ndk.NewTelemetryState(16)
	state.EVPNRoutes = []ndk.EVPNRouteEntry{
		{RouteType: 2, RD: "1.1.1.1:10010", MAC: "00:11:22:33:44:55", IP: "192.168.200.254", NextHop: "1.1.1.1", Status: "u*>"},
	}
	pal := theme.Cyberpunk

	rendered := RenderEVPNView(state, EVPNFilterAll, 0, pal, 140, 30, "", false, "", true)
	expectedPayload := "00:11:22:33:44:55 [192.168.200.254]"
	if !strings.Contains(rendered, expectedPayload) {
		t.Fatalf("Expected full untruncated MAC+IP payload %q in rendered view, but was clipped/missing. Got: %s", expectedPayload, rendered)
	}
}

func TestEVPNViewConstantHeightRendering(t *testing.T) {
	state := ndk.NewTelemetryState(16)
	state.EVPNRoutes = []ndk.EVPNRouteEntry{
		{RouteType: 2, RD: "1.1.1.1:10010", MAC: "00:11:22:33:44:55", IP: "192.168.200.254", NextHop: "1.1.1.1", Status: "u*>"},
	}
	pal := theme.Cyberpunk

	targetHeight := 30
	// Render with 1 route (Type 2)
	view1Route := RenderEVPNView(state, EVPNFilterType2, 0, pal, 140, targetHeight, "", false, "", true)
	// Render with 0 routes (Type 4)
	view0Routes := RenderEVPNView(state, EVPNFilterType4, 0, pal, 140, targetHeight, "", false, "", true)

	lines1 := strings.Split(view1Route, "\n")
	lines0 := strings.Split(view0Routes, "\n")

	if len(lines1) != len(lines0) {
		t.Fatalf("BUG DETECTED: EVPN View height changes between subtabs! 1-route height=%d vs 0-route height=%d", len(lines1), len(lines0))
	}
	if len(lines1) != targetHeight {
		t.Fatalf("Expected rendered box height %d, got %d lines", targetHeight, len(lines1))
	}
}
