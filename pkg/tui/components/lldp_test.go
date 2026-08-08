package components

import (
	"strings"
	"testing"

	"srl-tui/pkg/ndk"
	"srl-tui/pkg/tui/theme"
)

func TestLLDPNeighborUnknownManagementIP(t *testing.T) {
	state := ndk.TelemetryState{
		LLDPNeighbors: []ndk.LLDPNeighbor{
			{
				LocalPort:  "ethernet-1/1",
				SysName:    "spine-1",
				RemotePort: "ethernet-1/1",
				MgmtIP:     "", // Empty / Not received from neighbor
			},
		},
	}

	view := NewLLDPView()
	rendered := view.Render(&state, theme.Cyberpunk, 120, 30, "", false, "")

	if !strings.Contains(rendered, "(unknown)") {
		t.Errorf("BUG DETECTED: Expected LLDP view to render '(unknown)' when MgmtIP is empty, got:\n%s", rendered)
	}

	if strings.Contains(rendered, "10.0.0.1") {
		t.Errorf("BUG DETECTED: LLDP view rendered dummy '10.0.0.1' instead of '(unknown)'!")
	}

	modalRendered := RenderLLDPDetailModal(state.LLDPNeighbors[0], &state, theme.Cyberpunk, 120, 30)
	if !strings.Contains(modalRendered, "(unknown)") {
		t.Errorf("BUG DETECTED: Expected LLDP modal to render '(unknown)' when MgmtIP is empty, got:\n%s", modalRendered)
	}
}
