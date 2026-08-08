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
	outputNoMatch := RenderTopoMesh(state.Snapshot(), true, pal, 100, 30, "nonexistent-query-999", false, "")

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
		},
	}

	outputMatch := RenderTopoMesh(state.Snapshot(), true, pal, 100, 30, "", false, "")

	if !strings.Contains(outputMatch, "10.0.0.1") || !strings.Contains(outputMatch, "AS65000") {
		t.Errorf("Expected real BGP peer 10.0.0.1 AS65000 in output")
	}

	if strings.Contains(outputMatch, "peer-spine") {
		t.Errorf("BUG DETECTED: RenderTopoMesh rendered fake peer-spine boxes alongside real BGP peer!")
	}
}
