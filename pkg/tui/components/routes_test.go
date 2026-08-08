package components

import (
	"strings"
	"testing"

	"srl-tui/pkg/ndk"
	"srl-tui/pkg/tui/theme"
)

func TestECMPRouteNextHopResolution(t *testing.T) {
	state := ndk.NewTelemetryState(16)
	state.RouteTable = []ndk.RouteEntry{
		{
			Prefix:     "2.2.2.2/32",
			Protocol:   "bgp",
			NextHop:    "10.1.10.10, 10.1.20.20",
			NextHops:   []string{"10.1.10.10", "10.1.20.20"},
			Preference: 170,
			Metric:     0,
			NetInst:    "default",
		},
	}

	rv := NewRouteView()
	pal := theme.Cyberpunk

	rendered := rv.Render(state, pal, 140, 30, "", false, "")
	if !strings.Contains(rendered, "10.1.10.10, 10.1.20.20 [ECMP x2]") {
		t.Fatalf("Expected '[ECMP x2]' badge on route list, got: %s", rendered)
	}

	modal := RenderRouteDetailModal(state.RouteTable[0], state, pal, 140, 30)
	if !strings.Contains(modal, "Equal-Cost Multi-Path (ECMP): 2 Active Next-Hop Paths") {
		t.Fatalf("Expected ECMP breakdown section in route detail modal, got: %s", modal)
	}
	if !strings.Contains(modal, "[1] Next-Hop Path: 10.1.10.10") || !strings.Contains(modal, "[2] Next-Hop Path: 10.1.20.20") {
		t.Fatalf("Expected both ECMP next-hops in detail modal, got: %s", modal)
	}
}
