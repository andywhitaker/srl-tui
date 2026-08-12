package ndk

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

func createGNMIUpdateNotification(pathString string, jsonVal interface{}) *pb.Notification {
	valBytes, _ := json.Marshal(jsonVal)

	elems := []*pb.PathElem{
		{Name: "network-instance", Key: map[string]string{"name": "default"}},
		{Name: "protocols"},
		{Name: "bgp"},
		{Name: "neighbor", Key: map[string]string{"peer-address": "10.1.10.10"}},
	}

	return &pb.Notification{
		Timestamp: time.Now().UnixNano(),
		Update: []*pb.Update{
			{
				Path: &pb.Path{Elem: elems},
				Val: &pb.TypedValue{
					Value: &pb.TypedValue_JsonIetfVal{
						JsonIetfVal: valBytes,
					},
				},
			},
		},
	}
}

func TestBGPPeerDownUptimeNotEstablished(t *testing.T) {
	state := NewTelemetryState(16)
	client := NewNDKClient("unix:///tmp/dummy.sock", state)

	// Step 1: Established update
	estPayload := map[string]interface{}{
		"peer-address":     "10.1.10.10",
		"peer-as":          float64(10),
		"peer-type":        "ebgp",
		"session-state":    "established",
		"last-established": time.Now().Add(-10 * time.Minute).Format(time.RFC3339),
	}
	notif1 := createGNMIUpdateNotification("/network-instance/protocols/bgp/neighbor", estPayload)
	client.parseGNMIStreamNotification(notif1)

	if len(state.BGPPeers) != 1 {
		t.Fatalf("Expected 1 BGP peer, got %d", len(state.BGPPeers))
	}
	p1 := state.BGPPeers[0]
	if p1.SessionState != "ESTABLISHED" {
		t.Errorf("Expected ESTABLISHED state, got %s", p1.SessionState)
	}
	if p1.Uptime == "established" || p1.Uptime == "-" {
		t.Errorf("Expected formatted uptime duration (e.g. 10m00s), got %s", p1.Uptime)
	}

	// Step 2: Peer goes down (state IDLE)
	downPayload := map[string]interface{}{
		"peer-address":  "10.1.10.10",
		"session-state": "idle",
	}
	notif2 := createGNMIUpdateNotification("/network-instance/protocols/bgp/neighbor", downPayload)
	client.parseGNMIStreamNotification(notif2)

	p2 := state.BGPPeers[0]
	if p2.SessionState != "IDLE" {
		t.Errorf("Expected IDLE state, got %s", p2.SessionState)
	}
	if p2.Uptime == "established" {
		t.Errorf("BUG DETECTED: Uptime must NOT be 'established' when BGP peer is down! Got: %s", p2.Uptime)
	}
	if p2.Uptime != "-" {
		t.Errorf("Expected uptime '-' when BGP peer is down, got %s", p2.Uptime)
	}
}

func TestBGPPeerASPreservationOnFlap(t *testing.T) {
	state := NewTelemetryState(16)
	client := NewNDKClient("unix:///tmp/dummy.sock", state)

	// Step 1: Peer initialized with peer-as 10
	initPayload := map[string]interface{}{
		"peer-address":  "10.1.10.10",
		"peer-as":       float64(10),
		"peer-type":     "ebgp",
		"session-state": "established",
	}
	client.parseGNMIStreamNotification(createGNMIUpdateNotification("/neighbor", initPayload))

	if state.BGPPeers[0].PeerASN != 10 {
		t.Fatalf("Expected PeerASN 10, got %d", state.BGPPeers[0].PeerASN)
	}

	// Step 2: Peer flaps and receives update without peer-as field
	flapPayload := map[string]interface{}{
		"peer-address":  "10.1.10.10",
		"session-state": "established",
		// peer-as intentionally omitted
	}
	client.parseGNMIStreamNotification(createGNMIUpdateNotification("/neighbor", flapPayload))

	p := state.BGPPeers[0]
	if p.PeerASN == 0 {
		t.Errorf("BUG DETECTED: PeerASN reset to 0 after flap/update!")
	}
	if p.PeerASN != 10 {
		t.Errorf("Expected PeerASN 10 preserved, got %d", p.PeerASN)
	}
}

func TestBGPPrefixRxTxCounters(t *testing.T) {
	state := NewTelemetryState(16)
	client := NewNDKClient("unix:///tmp/dummy.sock", state)

	payload := map[string]interface{}{
		"peer-address":  "10.1.10.10",
		"peer-as":       float64(10),
		"session-state": "established",
		"afi-safi": []interface{}{
			map[string]interface{}{
				"afi-safi-name":   "evpn",
				"received-routes": float64(39),
				"sent-routes":     float64(11),
			},
			map[string]interface{}{
				"afi-safi-name":   "ipv4-unicast",
				"received-routes": float64(4),
				"sent-routes":     float64(2),
			},
		},
	}

	client.parseGNMIStreamNotification(createGNMIUpdateNotification("/neighbor", payload))

	if len(state.BGPPeers) != 1 {
		t.Fatalf("Expected 1 peer, got %d", len(state.BGPPeers))
	}

	p := state.BGPPeers[0]
	if p.RxPrefixes != 43 {
		t.Errorf("Expected RxPrefixes 43 (39 + 4), got %d", p.RxPrefixes)
	}
	if p.TxPrefixes != 13 {
		t.Errorf("Expected TxPrefixes 13 (11 + 2), got %d", p.TxPrefixes)
	}
}

func TestEVPNRouteImportedStatusOnLeafVsSpine(t *testing.T) {
	// 1. Node with populated local forwarding datastores (e.g. active bridge table MACs / ARPs)
	leafState := NewTelemetryState(16)
	leafState.MACTables = []MACTableEntry{
		{MACAddress: "1A:46:05:FF:00:41"},
		{MACAddress: "AA:C1:AB:28:FB:B5"},
	}
	leafState.ARPTables = []ARPEntry{
		{IPAddress: "192.168.10.1", MACAddress: "AA:C1:AB:28:FB:B5"},
	}
	PrepopulateTestEVPNRoutes(leafState)
	leafClient := NewNDKClient("unix:///tmp/dummy.sock", leafState)
	leafClient.parseGNMIStreamNotification(&pb.Notification{})

	installedCount := 0
	for _, r := range leafState.EVPNRoutes {
		if r.Status == "u*>" {
			installedCount++
		}
	}
	if installedCount == 0 {
		t.Errorf("Expected installed EVPN routes for populated local MAC/ARP entries, got 0")
	}

	// 2. Node without local forwarding datastores (e.g. Route Reflector without local VRFs / MACs)
	spineState := NewTelemetryState(16)
	PrepopulateTestEVPNRoutes(spineState)
	spineClient := NewNDKClient("unix:///tmp/dummy.sock", spineState)
	spineClient.parseGNMIStreamNotification(&pb.Notification{})

	if len(spineState.EVPNRoutes) == 0 {
		t.Fatalf("Expected EVPN routes in BGP RIB-In on Route Reflector, got 0")
	}

	for _, r := range spineState.EVPNRoutes {
		if r.Status != "r*" {
			t.Errorf("EVPN Route VNI %s on node without local FIB entries should be unimported ('r*'), got status '%s'", r.VNI, r.Status)
		}
	}
}

func TestFilterSelfOriginatedEVPNRoutes(t *testing.T) {
	state := NewTelemetryState(16)
	state.RouteTable = []RouteEntry{
		{NetInst: "default", Prefix: "1.1.1.1/32", Protocol: "local"},
	}
	client := NewNDKClient("unix:///tmp/dummy.sock", state)
	client.parseGNMIStreamNotification(&pb.Notification{})

	for _, r := range state.EVPNRoutes {
		if r.NextHop == "1.1.1.1" {
			t.Errorf("BUG DETECTED: Self-originated EVPN route with NextHop '1.1.1.1' was not filtered out!")
		}
	}
}

func TestEVPNRouteStableSortingOrder(t *testing.T) {
	state := NewTelemetryState(16)
	client := NewNDKClient("unix:///tmp/dummy.sock", state)

	// Trigger initial stream parses to reach steady state
	client.parseGNMIStreamNotification(&pb.Notification{})
	client.parseGNMIStreamNotification(&pb.Notification{})

	// Record initial ordering of EVPN routes in steady state
	initialOrder := make([]string, len(state.EVPNRoutes))
	for i, r := range state.EVPNRoutes {
		initialOrder[i] = fmt.Sprintf("%d-%s-%s-%s-%s-%s", r.RouteType, r.RD, r.MAC, r.IP, r.Prefix, r.NextHop)
	}

	// Trigger 50 repeated stream parses (simulating 1-second interface traffic updates)
	for k := 0; k < 50; k++ {
		client.parseGNMIStreamNotification(&pb.Notification{})
		if len(state.EVPNRoutes) != len(initialOrder) {
			t.Fatalf("EVPNRoute count changed during stream updates: expected %d, got %d", len(initialOrder), len(state.EVPNRoutes))
		}
		for i, r := range state.EVPNRoutes {
			currentKey := fmt.Sprintf("%d-%s-%s-%s-%s-%s", r.RouteType, r.RD, r.MAC, r.IP, r.Prefix, r.NextHop)
			if currentKey != initialOrder[i] {
				t.Fatalf("BUG DETECTED: EVPN route order shifted at index %d on iteration %d! Initial: %s, Current: %s",
					i, k, initialOrder[i], currentKey)
			}
		}
	}
}

func TestEVPNRouteDeduplicationAndPathVersions(t *testing.T) {
	state := NewTelemetryState(16)
	client := NewNDKClient("unix:///tmp/dummy.sock", state)
	client.parseGNMIStreamNotification(&pb.Notification{})

	// Verify no duplicate EVPN routes for same NextHop & VNI across different neighbors
	seenKeys := make(map[string]bool)
	for _, r := range state.EVPNRoutes {
		key := fmt.Sprintf("%d-%s-%s-%s-%s-%s", r.RouteType, r.RD, r.MAC, r.IP, r.Prefix, r.NextHop)
		if seenKeys[key] {
			t.Fatalf("BUG DETECTED: Duplicate EVPN route emitted on main view for key: %s", key)
		}
		seenKeys[key] = true

		if len(r.PathVersions) < 2 {
			t.Fatalf("Expected multi-path BGP versions (at least 2), got %d for route %s", len(r.PathVersions), key)
		}
	}
}

func TestDynamicSwitchPortDiscovery(t *testing.T) {
	state := NewTelemetryState(16)
	client := NewNDKClient("unix:///tmp/dummy.sock", state)

	if len(state.Ports) != 16 {
		t.Fatalf("Expected initial slice len 16, got %d", len(state.Ports))
	}

	// Stream update for ethernet-1/58 (port index 58)
	notif := &pb.Notification{
		Timestamp: time.Now().UnixNano(),
		Update: []*pb.Update{
			{
				Path: &pb.Path{Elem: []*pb.PathElem{{Name: "interface", Key: map[string]string{"name": "ethernet-1/58"}}}},
				Val: &pb.TypedValue{
					Value: &pb.TypedValue_JsonIetfVal{
						JsonIetfVal: []byte(`{"admin-state":"enable","oper-state":"up","description":"10G Uplink","mtu":9000}`),
					},
				},
			},
		},
	}

	client.parseGNMIStreamNotification(notif)

	if len(state.Ports) < 58 {
		t.Fatalf("Expected slice to expand dynamically to at least 58 ports, got %d", len(state.Ports))
	}

	p58 := state.Ports[57]
	if p58.Name != "ethernet-1/58" || p58.AdminState != "up" || p58.OperState != "up" {
		t.Fatalf("Unexpected port 58 state: %+v", p58)
	}
}

func TestInterfaceStateParsingFromGNMI(t *testing.T) {
	state := NewTelemetryState(32)
	client := NewNDKClient("unix:///tmp/dummy.sock", state)

	// Notification for ethernet-1/1 (enabled/up)
	notif1 := &pb.Notification{
		Timestamp: time.Now().UnixNano(),
		Update: []*pb.Update{
			{
				Path: &pb.Path{Elem: []*pb.PathElem{{Name: "interface", Key: map[string]string{"name": "ethernet-1/1"}}}},
				Val: &pb.TypedValue{
					Value: &pb.TypedValue_JsonIetfVal{
						JsonIetfVal: []byte(`{"admin-state":"enable","oper-state":"up","description":"Uplink Spine1","mtu":9000}`),
					},
				},
			},
		},
	}
	client.parseGNMIStreamNotification(notif1)

	p1 := state.Ports[0]
	if p1.AdminState != "up" || p1.OperState != "up" {
		t.Errorf("Expected ethernet-1/1 Admin:up Oper:up, got Admin:%s Oper:%s", p1.AdminState, p1.OperState)
	}
	if p1.Description != "Uplink Spine1" {
		t.Errorf("Expected description 'Uplink Spine1', got '%s'", p1.Description)
	}

	// Notification for ethernet-1/4 (disabled/down)
	notif4 := &pb.Notification{
		Timestamp: time.Now().UnixNano(),
		Update: []*pb.Update{
			{
				Path: &pb.Path{Elem: []*pb.PathElem{{Name: "interface", Key: map[string]string{"name": "ethernet-1/4"}}}},
				Val: &pb.TypedValue{
					Value: &pb.TypedValue_JsonIetfVal{
						JsonIetfVal: []byte(`{"admin-state":"disable","oper-state":"down"}`),
					},
				},
			},
		},
	}
	client.parseGNMIStreamNotification(notif4)

	p4 := state.Ports[3]
	if p4.AdminState != "down" || p4.OperState != "down" {
		t.Errorf("Expected ethernet-1/4 Admin:down Oper:down, got Admin:%s Oper:%s", p4.AdminState, p4.OperState)
	}
}

func TestGNMIOperStateNotOverwrittenByLocalScan(t *testing.T) {
	state := NewTelemetryState(32)
	client := NewNDKClient("unix:///tmp/dummy.sock", state)

	// 1. Simulate gNMI setting AdminState:up, OperState:down on ethernet-1/1 (e.g. spine peer down)
	notifDown := &pb.Notification{
		Timestamp: time.Now().UnixNano(),
		Update: []*pb.Update{
			{
				Path: &pb.Path{Elem: []*pb.PathElem{{Name: "interface", Key: map[string]string{"name": "ethernet-1/1"}}}},
				Val: &pb.TypedValue{
					Value: &pb.TypedValue_JsonIetfVal{
						JsonIetfVal: []byte(`{"admin-state":"enable","oper-state":"down"}`),
					},
				},
			},
		},
	}
	client.parseGNMIStreamNotification(notifDown)

	state.Lock()
	state.NDKConnected = true
	state.Unlock()

	if state.Ports[0].AdminState != "up" || state.Ports[0].OperState != "down" {
		t.Fatalf("Expected ethernet-1/1 Admin:up Oper:down after gNMI update, got Admin:%s Oper:%s",
			state.Ports[0].AdminState, state.Ports[0].OperState)
	}

	// 2. Execute scanLocalInterfaces() and verify OperState remains "down" (NOT overwritten to "up")
	client.scanLocalInterfaces()

	p1 := state.Ports[0]
	if p1.AdminState != "up" || p1.OperState != "down" {
		t.Errorf("BUG DETECTED: scanLocalInterfaces() overwrote router gNMI OperState down to %s!", p1.OperState)
	}
}

func TestCPUAndRAMUtilizationParsing(t *testing.T) {
	state := NewTelemetryState(32)
	client := NewNDKClient("unix:///tmp/dummy.sock", state)

	platformJSON := []byte(`{
		"srl_nokia-platform-control:control": [
			{
				"srl_nokia-platform-memory:memory": {
					"utilization": 18
				},
				"srl_nokia-platform-cpu:cpu": [
					{
						"index": "all",
						"idle": {
							"instant": 85
						}
					}
				]
			}
		]
	}`)

	notif := &pb.Notification{
		Timestamp: time.Now().UnixNano(),
		Update: []*pb.Update{
			{
				Path: &pb.Path{Elem: []*pb.PathElem{{Name: "platform"}}},
				Val: &pb.TypedValue{
					Value: &pb.TypedValue_JsonIetfVal{
						JsonIetfVal: platformJSON,
					},
				},
			},
		},
	}

	client.parseGNMIStreamNotification(notif)

	snap := state.Snapshot()
	if snap.RAMUsage != 18 {
		t.Fatalf("Expected RAMUsage 18%%, got %.1f%%", snap.RAMUsage)
	}
	if snap.CPUUsage != 15 {
		t.Fatalf("Expected CPUUsage 15%% (100 - 85 idle), got %.1f%%", snap.CPUUsage)
	}
}

func TestLiveTrafficRateIngestionAndSampling(t *testing.T) {
	state := NewTelemetryState(32)
	client := NewNDKClient("unix:///tmp/dummy.sock", state)

	intfJSON := []byte(`{
		"admin-state": "enable",
		"oper-state": "up",
		"traffic-rate": {
			"in-bps": "1250000",
			"out-bps": "2500000"
		}
	}`)

	notif := &pb.Notification{
		Timestamp: time.Now().UnixNano(),
		Update: []*pb.Update{
			{
				Path: &pb.Path{Elem: []*pb.PathElem{{Name: "interface", Key: map[string]string{"name": "ethernet-1/1"}}}},
				Val: &pb.TypedValue{
					Value: &pb.TypedValue_JsonIetfVal{
						JsonIetfVal: intfJSON,
					},
				},
			},
		},
	}

	client.parseGNMIStreamNotification(notif)
	client.scanLocalInterfaces()

	snap := state.Snapshot()
	p1 := snap.Ports[0]
	if p1.RxBps != 1250000 || p1.TxBps != 2500000 {
		t.Fatalf("Expected ethernet-1/1 RxBps=1250000, TxBps=2500000, got Rx=%.0f, Tx=%.0f", p1.RxBps, p1.TxBps)
	}

	if len(snap.IngressHistory) == 0 || snap.IngressHistory[len(snap.IngressHistory)-1] != 1250000 {
		t.Fatalf("Expected live traffic sample pushed to history history, got %+v", snap.IngressHistory)
	}
}

func TestBGPMaintenanceModeAndActiveAddressFamilies(t *testing.T) {
	state := NewTelemetryState(16)
	client := NewNDKClient("unix:///tmp/dummy.sock", state)

	bgpJSON := []byte(`{
		"peer-address": "10.1.10.10",
		"peer-as": 65001,
		"peer-type": "ebgp",
		"session-state": "established",
		"afi-safi": [
			{"afi-safi-name": "srl_nokia-bgp:ipv4-unicast", "received-routes": 10, "sent-routes": 5},
			{"afi-safi-name": "srl_nokia-bgp:evpn", "received-routes": 20, "sent-routes": 15}
		]
	}`)

	notif := &pb.Notification{
		Timestamp: time.Now().UnixNano(),
		Update: []*pb.Update{
			{
				Path: &pb.Path{
					Elem: []*pb.PathElem{
						{Name: "network-instance", Key: map[string]string{"name": "default"}},
						{Name: "protocols"},
						{Name: "bgp"},
						{Name: "neighbor", Key: map[string]string{"peer-address": "10.1.10.10"}},
					},
				},
				Val: &pb.TypedValue{
					Value: &pb.TypedValue_JsonIetfVal{
						JsonIetfVal: bgpJSON,
					},
				},
			},
		},
	}

	client.parseGNMIStreamNotification(notif)
	snap1 := state.Snapshot()

	if len(snap1.BGPPeers) != 1 {
		t.Fatalf("Expected 1 BGP peer, got %d", len(snap1.BGPPeers))
	}
	p1 := snap1.BGPPeers[0]
	if len(p1.AddrFamilies) != 2 || p1.AddrFamilies[0] != "ipv4-unicast" || p1.AddrFamilies[1] != "evpn" {
		t.Errorf("Expected active address families ['ipv4-unicast', 'evpn'], got %+v", p1.AddrFamilies)
	}
	if p1.InMaintenance {
		t.Errorf("Expected neighbor 10.1.10.10 NOT in maintenance mode initially")
	}

	// Toggle maintenance mode ON
	err := client.SetBGPNeighborMaintenanceMode(t.Context(), "10.1.10.10", true, "maint-bgp-10-1-10-10")
	if err != nil {
		t.Fatalf("Unexpected error setting maintenance mode: %v", err)
	}

	snap2 := state.Snapshot()
	p2 := snap2.BGPPeers[0]
	if !p2.InMaintenance {
		t.Errorf("Expected neighbor 10.1.10.10 in maintenance mode after ToggleNeighborMaintenance")
	}
	if p2.MaintenanceGroup != "maint-bgp-10-1-10-10" {
		t.Errorf("Expected maintenance group 'maint-bgp-10-1-10-10', got %s", p2.MaintenanceGroup)
	}

	// Toggle maintenance mode OFF
	_ = client.SetBGPNeighborMaintenanceMode(t.Context(), "10.1.10.10", false, "maint-bgp-10-1-10-10")
	snap3 := state.Snapshot()
	p3 := snap3.BGPPeers[0]
	if p3.InMaintenance {
		t.Errorf("Expected neighbor 10.1.10.10 NOT in maintenance mode after disabling maintenance")
	}
}

func TestType2MACIPAndDualVNIRoutes(t *testing.T) {
	state := NewTelemetryState(16)
	client := NewNDKClient("unix:///tmp/dummy.sock", state)

	// Add authentic BGP EVPN routes to state
	state.EVPNRoutes = []EVPNRouteEntry{
		{
			RouteType: 2,
			RD:        "4.4.4.4:10010",
			RT:        "10010:10010",
			VNI:       "10010",
			MAC:       "AA:C1:AB:62:06:A8",
			IP:        "",
			NextHop:   "4.4.4.4",
			Neighbor:  "10.1.10.10",
			Status:    "u*>",
			PathVersions: []EVPNPathVersion{
				{Neighbor: "10.1.10.10", NextHop: "4.4.4.4", StatusCode: "u*>", PathID: 0},
			},
		},
		{
			RouteType: 2,
			RD:        "4.4.4.4:10010",
			RT:        "10010:10010",
			VNI:       "10010 + 10000",
			MAC:       "AA:C1:AB:62:06:A8",
			IP:        "192.168.10.4",
			NextHop:   "4.4.4.4",
			Neighbor:  "10.1.10.10",
			Status:    "u*>",
			PathVersions: []EVPNPathVersion{
				{Neighbor: "10.1.10.10", NextHop: "4.4.4.4", StatusCode: "u*>", PathID: 0},
			},
		},
	}

	macJSON := []byte(`{
		"address": "AA:C1:AB:62:06:A8",
		"destination-type": "vxlan",
		"type": "evpn-static",
		"destination": "vxlan-interface:vxlan0.101 vtep:4.4.4.4 vni:10010"
	}`)

	tunnelJSON := []byte(`{
		"tunnel": [{"ipv4-prefix": "4.4.4.4/32"}]
	}`)

	arpJSON := []byte(`{
		"link-layer-address": "AA:C1:AB:62:06:A8",
		"origin": "evpn"
	}`)

	// Send single combined notification containing MAC, Tunnel, and ARP FIB updates
	client.parseGNMIStreamNotification(&pb.Notification{
		Timestamp: time.Now().UnixNano(),
		Update: []*pb.Update{
			{
				Path: &pb.Path{Elem: []*pb.PathElem{
					{Name: "network-instance", Key: map[string]string{"name": "app"}},
					{Name: "bridge-table"},
					{Name: "mac-table"},
					{Name: "mac", Key: map[string]string{"address": "AA:C1:AB:62:06:A8"}},
				}},
				Val: &pb.TypedValue{Value: &pb.TypedValue_JsonIetfVal{JsonIetfVal: macJSON}},
			},
			{
				Path: &pb.Path{Elem: []*pb.PathElem{
					{Name: "network-instance", Key: map[string]string{"name": "default"}},
					{Name: "tunnel-table"},
				}},
				Val: &pb.TypedValue{Value: &pb.TypedValue_JsonIetfVal{JsonIetfVal: tunnelJSON}},
			},
			{
				Path: &pb.Path{Elem: []*pb.PathElem{
					{Name: "network-instance", Key: map[string]string{"name": "app"}},
					{Name: "neighbor", Key: map[string]string{"ipv4-address": "192.168.10.4"}},
				}},
				Val: &pb.TypedValue{Value: &pb.TypedValue_JsonIetfVal{JsonIetfVal: arpJSON}},
			},
		},
	})

	snap := state.Snapshot()

	var macOnly, macIP *EVPNRouteEntry
	for i, r := range snap.EVPNRoutes {
		if r.RouteType == 2 && r.MAC == "AA:C1:AB:62:06:A8" {
			if r.IP == "" {
				macOnly = &snap.EVPNRoutes[i]
			} else if r.IP == "192.168.10.4" {
				macIP = &snap.EVPNRoutes[i]
			}
		}
	}

	if macOnly == nil {
		t.Fatalf("Expected MAC-Only Type 2 route for AA:C1:AB:62:06:A8, but missing")
	}
	if macIP == nil {
		t.Fatalf("Expected MAC-IP Type 2 route for AA:C1:AB:62:06:A8 (192.168.10.4), but missing")
	}
	if !strings.Contains(macIP.VNI, "10010") || !strings.Contains(macIP.VNI, "10000") {
		t.Errorf("Expected dual VNIs (e.g. '10010 + 10000') for MAC-IP route, got %s", macIP.VNI)
	}
	if macOnly.Status != "u*>" || macIP.Status != "u*>" {
		t.Errorf("Expected both authentic EVPN routes to be validated as installed in FIB (u*>), got MAC-only=%s, MAC-IP=%s", macOnly.Status, macIP.Status)
	}
}

func TestType2MACIPSingleL2VNIRoute(t *testing.T) {
	state := NewTelemetryState(16)
	route := EVPNRouteEntry{
		RouteType: 2,
		RD:        "4.4.4.4:10010",
		RT:        "10010:10010",
		VNI:       "10010",
		MAC:       "AA:BB:CC:DD:EE:11",
		IP:        "192.168.10.55",
		NextHop:   "4.4.4.4",
		Neighbor:  "10.1.10.10",
		Status:    "u*>",
	}
	state.EVPNRoutes = append(state.EVPNRoutes, route)

	snap := state.Snapshot()
	if len(snap.EVPNRoutes) != 1 {
		t.Fatalf("Expected 1 EVPN route, got %d", len(snap.EVPNRoutes))
	}

	r := snap.EVPNRoutes[0]
	if r.MAC != "AA:BB:CC:DD:EE:11" || r.IP != "192.168.10.55" {
		t.Errorf("Expected MAC-IP Type 2 route with MAC and IP, got MAC=%s, IP=%s", r.MAC, r.IP)
	}
	if r.VNI != "10010" {
		t.Errorf("Expected single L2-VNI '10010', got %s", r.VNI)
	}
}

func TestEVPNAllRouteTypesIngestionAndFIBValidation(t *testing.T) {
	state := NewTelemetryState(16)
	state.RouteTable = append(state.RouteTable, RouteEntry{NetInst: "default", Prefix: "1.1.1.1/32", Protocol: "system"})
	PrepopulateTestEVPNRoutes(state)
	client := NewNDKClient("unix:///tmp/dummy.sock", state)

	macJSON := []byte(`{
		"address": "1A:46:05:FF:00:41",
		"destination-type": "vxlan",
		"destination": "vxlan0.101 vtep:2.2.2.2 vni:10010"
	}`)

	// Trigger telemetry sync with MAC FIB update
	client.parseGNMIStreamNotification(&pb.Notification{
		Timestamp: time.Now().UnixNano(),
		Update: []*pb.Update{
			{
				Path: &pb.Path{Elem: []*pb.PathElem{
					{Name: "network-instance", Key: map[string]string{"name": "app"}},
					{Name: "bridge-table"},
					{Name: "mac-table"},
					{Name: "mac", Key: map[string]string{"address": "1A:46:05:FF:00:41"}},
				}},
				Val: &pb.TypedValue{Value: &pb.TypedValue_JsonIetfVal{JsonIetfVal: macJSON}},
			},
		},
	})

	snap := state.Snapshot()
	var type2Count, type3Count, type5Count int
	for _, r := range snap.EVPNRoutes {
		switch r.RouteType {
		case 2:
			type2Count++
		case 3:
			type3Count++
		case 5:
			type5Count++
		}
	}

	if type2Count != 24 {
		t.Errorf("Expected 24 Type-2 routes in EVPN table, got %d", type2Count)
	}
	if type3Count != 6 {
		t.Errorf("Expected 6 Type-3 IMET routes in EVPN table, got %d", type3Count)
	}
	if type5Count != 6 {
		t.Errorf("Expected 6 Type-5 IP Prefix routes in EVPN table, got %d", type5Count)
	}

	// Verify imported routes are accessible by default (Status u*>)
	importedType3Count := 0
	for _, r := range snap.EVPNRoutes {
		if r.RouteType == 3 && r.Status == "u*>" {
			importedType3Count++
		}
	}
	if importedType3Count != 6 {
		t.Errorf("Expected 6 imported Type-3 routes from remote peer VTEPs, got %d", importedType3Count)
	}

	// Verify presence of 192.168.10.3 and 192.168.20.3 MAC-IP routes
	has10_3, has20_3 := false, false
	for _, r := range snap.EVPNRoutes {
		if r.RouteType == 2 {
			if r.IP == "192.168.10.3" {
				has10_3 = true
			}
			if r.IP == "192.168.20.3" {
				has20_3 = true
			}
		}
	}
	if !has10_3 {
		t.Errorf("Expected Type-2 route for 192.168.10.3 (leaf3), but missing")
	}
	if !has20_3 {
		t.Errorf("Expected Type-2 route for 192.168.20.3 (leaf3), but missing")
	}

	t.Logf("EVPN Route Counts: Type-2=%d, Type-3=%d (imported=%d), Type-5=%d, Total=%d", type2Count, type3Count, importedType3Count, type5Count, len(snap.EVPNRoutes))
}

func TestNoSlash32HostAddressesInType5Routes(t *testing.T) {
	state := NewTelemetryState(16)
	client := NewNDKClient("unix:///tmp/dummy.sock", state)

	// Simulate gNMI route update containing a /32 host IP route
	hostRouteJSON := []byte(`{
		"route-owner": "bgp_mgr",
		"active-next-hop": "4.4.4.4",
		"preference": 170,
		"metric": 0
	}`)
	client.parseGNMIStreamNotification(&pb.Notification{
		Timestamp: time.Now().UnixNano(),
		Update: []*pb.Update{
			{
				Path: &pb.Path{Elem: []*pb.PathElem{
					{Name: "network-instance", Key: map[string]string{"name": "default"}},
					{Name: "route-table"},
					{Name: "ipv4-unicast"},
					{Name: "route", Key: map[string]string{"ipv4-prefix": "192.168.10.4/32"}},
				}},
				Val: &pb.TypedValue{Value: &pb.TypedValue_JsonIetfVal{JsonIetfVal: hostRouteJSON}},
			},
		},
	})

	snap := state.Snapshot()
	for _, r := range snap.EVPNRoutes {
		if r.RouteType == 5 && strings.HasSuffix(r.Prefix, "/32") {
			t.Fatalf("BUG DETECTED: Synthesized Type-5 EVPN route for /32 host IP: %s", r.Prefix)
		}
	}
}

func TestDynamicUptimeAndCPURAMCounters(t *testing.T) {
	state := NewTelemetryState(16)
	state.StartTime = time.Now().Add(-10 * time.Minute)
	state.CPUUsage = 25.5
	state.RAMUsage = 42.0

	pastEst := time.Now().Add(-5 * time.Minute)
	state.BGPPeers = []BGPPeerState{
		{
			NeighborIP:      "10.1.10.10",
			SessionState:    "ESTABLISHED",
			LastEstablished: pastEst,
			Uptime:          FormatUptimeDuration(time.Since(pastEst)),
		},
	}

	snap1 := state.Snapshot()
	if snap1.Uptime < 9*time.Minute {
		t.Fatalf("Expected system uptime >= 9m, got %v", snap1.Uptime)
	}
	if snap1.BGPPeers[0].Uptime == "" || snap1.BGPPeers[0].Uptime == "-" {
		t.Fatalf("Expected formatted BGP peer uptime, got %s", snap1.BGPPeers[0].Uptime)
	}

	time.Sleep(100 * time.Millisecond)
	snap2 := state.Snapshot()
	if snap2.Uptime <= snap1.Uptime {
		t.Fatalf("System uptime did not count up dynamically: snap1=%v, snap2=%v", snap1.Uptime, snap2.Uptime)
	}
}


