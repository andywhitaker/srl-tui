package components

import (
	"strings"
	"testing"

	"srl-tui/pkg/ndk"
	"srl-tui/pkg/tui/theme"
)

func TestTopoMeshNoFakePeerSpineBoxes(t *testing.T) {
	state := ndk.NewTelemetryState(32)
	state.Hostname = "leaf1"
	state.Platform = "7220 IXR-D2L"

	pal := theme.Cyberpunk

	// 1. Search query excluding all options
	outputNoMatch := RenderTopoMesh(state.Snapshot(), true, pal, 100, 30, "nonexistent-query-999", false, "", 0)

	if strings.Contains(outputNoMatch, "peer-spine-01") || strings.Contains(outputNoMatch, "peer-spine-02") {
		t.Errorf("BUG DETECTED: RenderTopoMesh rendered fake peer-spine boxes when filter excluded all options!\nOutput:\n%s", outputNoMatch)
	}

	if strings.Contains(outputNoMatch, "AS65001") || strings.Contains(outputNoMatch, "AS65002") {
		t.Errorf("BUG DETECTED: RenderTopoMesh rendered made up ASes when filter excluded all options!")
	}

	if !strings.Contains(outputNoMatch, "(no BGP peers match filter)") {
		t.Errorf("Expected '(no BGP peers match filter)' notice when query excludes all peers")
	}

	// 2. Real BGP Peer
	state.BGPPeers = []ndk.BGPPeerState{
		{
			NeighborIP:   "10.0.0.1",
			PeerASN:      65000,
			SessionState: "ESTABLISHED",
			Interface:    "ethernet-1/1.0",
			AddrFamilies: []string{"ipv4-unicast", "evpn"},
		},
	}

	outputMatch := RenderTopoMesh(state.Snapshot(), true, pal, 100, 30, "", false, "", 0)

	if !strings.Contains(outputMatch, "10.0.0.1") || !strings.Contains(outputMatch, "AS65000") {
		t.Errorf("Expected real BGP peer 10.0.0.1 AS65000 in output")
	}

	if !strings.Contains(outputMatch, "ACTIVE AFs") || !strings.Contains(outputMatch, "ipv4-unicast, evpn") {
		t.Errorf("Expected ACTIVE AFs column with 'ipv4-unicast, evpn' in output")
	}
}

func TestBGPNeighborMaintenanceModeIndicators(t *testing.T) {
	state := ndk.NewTelemetryState(32)
	state.Hostname = "leaf1"

	state.BGPPeers = []ndk.BGPPeerState{
		{
			NeighborIP:       "10.1.10.10",
			PeerASN:          65001,
			SessionState:     "ESTABLISHED",
			Interface:        "ethernet-1/1.0",
			AddrFamilies:     []string{"ipv4-unicast", "evpn"},
			MaintenanceGroup: "maint-bgp-10-1-10-10",
			InMaintenance:    true,
		},
	}

	pal := theme.Cyberpunk
	output := RenderTopoMesh(state.Snapshot(), true, pal, 120, 30, "", false, "", 0)

	if !strings.Contains(output, "🚧") {
		t.Errorf("Expected road cone emoji 🚧 in summary list output when neighbor is in maintenance mode")
	}
	if !strings.Contains(output, "MAINT") {
		t.Errorf("Expected MAINT indicator in summary output when neighbor is in maintenance mode")
	}

	// Verify Detail Modal Checkered Header
	snap := state.Snapshot()
	modalOutput := RenderBGPDetailModal(snap.BGPPeers[0], pal, 100, 30, snap, false, "", "")

	if !strings.Contains(modalOutput, "🏁") || !strings.Contains(modalOutput, "🚧") {
		t.Errorf("Expected checkered header with road cones 🏁 🚧 in BGP detail modal when in maintenance mode")
	}

	if !strings.Contains(modalOutput, "NEIGHBOR IN MAINTENANCE MODE") {
		t.Errorf("Expected 'NEIGHBOR IN MAINTENANCE MODE' in detail modal header")
	}

	// Verify Confirmation Overlay Sub-modal
	confirmOutput := RenderBGPDetailModal(snap.BGPPeers[0], pal, 100, 30, snap, true, "disable", "maint-bgp-10-1-10-10")
	if !strings.Contains(confirmOutput, "MAINTENANCE MODE CONFIRMATION") {
		t.Errorf("Expected maintenance mode confirmation overlay title")
	}
}
