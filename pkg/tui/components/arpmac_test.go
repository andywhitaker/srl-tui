package components

import (
	"strings"
	"testing"

	"srl-tui/pkg/ndk"
	"srl-tui/pkg/tui/theme"
)

func TestARPMACSelectionAndModalRendering(t *testing.T) {
	state := &ndk.TelemetryState{
		ARPTables: []ndk.ARPEntry{
			{
				IPAddress:  "10.1.10.1",
				MACAddress: "00:11:22:33:44:55",
				Interface:  "ethernet-1/1.0",
				NetInst:    "default",
				EntryType:  "dynamic",
				ExpirySec:  285,
			},
			{
				IPAddress:  "172.20.20.7",
				MACAddress: "AA:BB:CC:DD:EE:FF",
				Interface:  "mgmt0.0",
				NetInst:    "mgmt",
				EntryType:  "static",
				ExpirySec:  0,
			},
		},
		MACTables: []ndk.MACTableEntry{
			{
				MACAddress: "52:54:00:12:34:56",
				NetInst:    "tenant1",
				Interface:  "vxlan0.1",
				Type:       "evpn",
				VNI:        10001,
				VTEP:       "10.10.10.10",
			},
		},
	}

	pal := theme.Cyberpunk
	view := NewARPMACView()

	// Test 1: ActivePane 0 (ARP Table) rendering with cursor selection
	renderedARP := view.Render(state, 120, 25, pal, "", false, "")
	if !strings.Contains(renderedARP, "► 10.1.10.1") {
		t.Errorf("Expected selected ARP entry cursor '► 10.1.10.1', got:\n%s", renderedARP)
	}

	// Test 2: Switch to ActivePane 1 (MAC Table)
	view.TogglePane()
	renderedMAC := view.Render(state, 120, 25, pal, "", false, "")
	if !strings.Contains(renderedMAC, "► 52:54:00:12:34:56") {
		t.Errorf("Expected selected MAC entry cursor '► 52:54:00:12:34:56', got:\n%s", renderedMAC)
	}

	// Test 3: ARP Detail Modal Rendering
	arpModalStr := RenderARPDetailModal(state.ARPTables[0], state, pal, 120, 25)
	if !strings.Contains(arpModalStr, "ARP NEIGHBOR DETAILS - 10.1.10.1") {
		t.Errorf("Expected ARP detail header in modal, got:\n%s", arpModalStr)
	}
	if !strings.Contains(arpModalStr, "00:11:22:33:44:55") {
		t.Errorf("Expected MAC address in ARP detail modal, got:\n%s", arpModalStr)
	}

	// Test 4: MAC Detail Modal Rendering
	macModalStr := RenderMACDetailModal(state.MACTables[0], state, pal, 120, 25)
	if !strings.Contains(macModalStr, "MAC TABLE ENTRY DETAILS - 52:54:00:12:34:56") {
		t.Errorf("Expected MAC detail header in modal, got:\n%s", macModalStr)
	}
	if !strings.Contains(macModalStr, "VXLAN Tunnel End Point (10.10.10.10)") {
		t.Errorf("Expected VTEP details in MAC detail modal, got:\n%s", macModalStr)
	}
}
