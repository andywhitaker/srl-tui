package ndk

import (
	"fmt"
)

// PrepopulateTestEVPNRoutes populates mock EVPN routes for unit test suites and offline demo mode.
func PrepopulateTestEVPNRoutes(state *TelemetryState) {
	state.Lock()
	defer state.Unlock()

	macMap := make(map[string]MACTableEntry)
	for _, m := range state.MACTables {
		macMap[m.MACAddress] = m
	}
	arpMap := make(map[string]ARPEntry)
	for _, a := range state.ARPTables {
		arpMap[a.IPAddress] = a
	}
	routeMap := make(map[string]RouteEntry)
	for _, r := range state.RouteTable {
		key := fmt.Sprintf("%s-%s", r.NetInst, r.Prefix)
		routeMap[key] = r
	}

	var activeVTEPs []string
	if len(state.MACTables) > 0 || len(state.ARPTables) > 0 || len(state.RouteTable) > 0 {
		activeVTEPs = []string{"1.1.1.1", "2.2.2.2", "3.3.3.3", "4.4.4.4"}
	}

	leaves := []struct {
		VTEP     string
		Neighbor string
	}{
		{"1.1.1.1", "local"},
		{"2.2.2.2", "10.1.10.10"},
		{"3.3.3.3", "10.1.10.10"},
		{"4.4.4.4", "10.1.10.10"},
	}

	macEntries := []struct {
		MAC  string
		IP   string
		VNI  string
		VTEP string
	}{
		// Leaf 2 (2.2.2.2) - 8 unique routes
		{"00:00:5E:00:01:01", "192.168.10.254", "10010", "2.2.2.2"},
		{"1A:E5:05:FF:00:41", "", "10010", "2.2.2.2"},
		{"AA:C1:AB:DD:52:C2", "", "10010", "2.2.2.2"},
		{"AA:C1:AB:DD:52:C2", "192.168.10.2", "10010", "2.2.2.2"},
		{"00:00:5E:00:01:01", "192.168.20.254", "10020", "2.2.2.2"},
		{"1A:E5:05:FF:00:41", "", "10020", "2.2.2.2"},
		{"AA:C1:AB:D8:D6:54", "", "10020", "2.2.2.2"},
		{"AA:C1:AB:D8:D6:54", "192.168.20.2", "10020", "2.2.2.2"},

		// Leaf 3 (3.3.3.3) - 8 unique routes
		{"00:00:5E:00:01:01", "192.168.10.254", "10010", "3.3.3.3"},
		{"1A:08:06:FF:00:41", "", "10010", "3.3.3.3"},
		{"AA:C1:AB:7F:B8:93", "", "10010", "3.3.3.3"},
		{"AA:C1:AB:7F:B8:93", "192.168.10.3", "10010", "3.3.3.3"},
		{"00:00:5E:00:01:01", "192.168.20.254", "10020", "3.3.3.3"},
		{"1A:08:06:FF:00:41", "", "10020", "3.3.3.3"},
		{"AA:C1:AB:B7:E6:A4", "", "10020", "3.3.3.3"},
		{"AA:C1:AB:B7:E6:A4", "192.168.20.3", "10020", "3.3.3.3"},

		// Leaf 4 (4.4.4.4) - 8 unique routes
		{"00:00:5E:00:01:01", "192.168.10.254", "10010", "4.4.4.4"},
		{"1A:B7:07:FF:00:41", "", "10010", "4.4.4.4"},
		{"AA:C1:AB:62:06:A8", "", "10010", "4.4.4.4"},
		{"AA:C1:AB:62:06:A8", "192.168.10.4", "10010", "4.4.4.4"},
		{"00:00:5E:00:01:01", "192.168.20.254", "10020", "4.4.4.4"},
		{"1A:B7:07:FF:00:41", "", "10020", "4.4.4.4"},
		{"AA:C1:AB:31:BC:DF", "", "10020", "4.4.4.4"},
		{"AA:C1:AB:31:BC:DF", "192.168.20.4", "10020", "4.4.4.4"},

		// Local Leaf 1 (1.1.1.1) - self-originated
		{"00:00:5E:00:01:01", "192.168.10.254", "10010", "1.1.1.1"},
		{"1A:46:05:FF:00:41", "", "10010", "1.1.1.1"},
		{"AA:C1:AB:8E:D8:A1", "", "10020", "1.1.1.1"},
		{"AA:C1:AB:8E:D8:A1", "192.168.20.1", "10020", "1.1.1.1"},
	}

	evpnMap := make(map[string]EVPNRouteEntry)

	// 1. Type-2 MAC-IP Advertisement Routes
	for _, m := range macEntries {
		rd := fmt.Sprintf("%s:%s", m.VTEP, m.VNI)
		rt := fmt.Sprintf("%s:%s", m.VNI, m.VNI)
		k2 := evpnRouteKey(EVPNRouteEntry{RouteType: 2, RD: rd, MAC: m.MAC, IP: m.IP, NextHop: m.VTEP})

		vniVal := m.VNI
		if m.IP != "" && m.VNI != "10000" {
			vniVal = fmt.Sprintf("%s + 10000", m.VNI)
		}

		candidate := EVPNRouteEntry{RouteType: 2, MAC: m.MAC, IP: m.IP, VNI: m.VNI, NextHop: m.VTEP}
		st := "r*"
		if isEVPNRouteInstalled(candidate, macMap, arpMap, routeMap, activeVTEPs) {
			st = "u*>"
		}

		nbrStr := "10.1.10.10"
		if m.VTEP == "1.1.1.1" {
			nbrStr = "local"
		}

		evpnMap[k2] = EVPNRouteEntry{
			RouteType:  2,
			RD:         rd,
			RT:         rt,
			VNI:        vniVal,
			MAC:        m.MAC,
			IP:         m.IP,
			NextHop:    m.VTEP,
			Neighbor:   nbrStr,
			Originator: "default",
			Status:     st,
			PathVersions: []EVPNPathVersion{
				{Neighbor: nbrStr, NextHop: m.VTEP, StatusCode: st, PathID: 0},
				{Neighbor: "10.1.20.20", NextHop: m.VTEP, StatusCode: "*", PathID: 0},
			},
		}
	}

	for _, l := range leaves {
		// 2. Type-3 Inclusive Multicast IMET Routes
		for _, vni := range []string{"10010", "10020"} {
			rd := fmt.Sprintf("%s:%s", l.VTEP, vni)
			rt := fmt.Sprintf("%s:%s", vni, vni)
			k3 := evpnRouteKey(EVPNRouteEntry{RouteType: 3, RD: rd, NextHop: l.VTEP})

			candidate := EVPNRouteEntry{RouteType: 3, VNI: vni, NextHop: l.VTEP}
			st := "r*"
			if isEVPNRouteInstalled(candidate, macMap, arpMap, routeMap, activeVTEPs) {
				st = "u*>"
			}

			evpnMap[k3] = EVPNRouteEntry{
				RouteType:  3,
				RD:         rd,
				RT:         rt,
				VNI:        vni,
				NextHop:    l.VTEP,
				Neighbor:   "10.1.10.10, 10.1.20.20",
				Originator: "default",
				Status:     st,
				PathVersions: []EVPNPathVersion{
					{Neighbor: "10.1.10.10", NextHop: l.VTEP, StatusCode: st, PathID: 0},
					{Neighbor: "10.1.20.20", NextHop: l.VTEP, StatusCode: "*", PathID: 0},
				},
			}
		}

		// 3. Type-5 IP Prefix Routes
		for _, pfx := range []string{"192.168.10.0/24", "192.168.20.0/24"} {
			rd := fmt.Sprintf("%s:10000", l.VTEP)
			k5 := evpnRouteKey(EVPNRouteEntry{RouteType: 5, RD: rd, Prefix: pfx, NextHop: l.VTEP})

			candidate := EVPNRouteEntry{RouteType: 5, Prefix: pfx, VNI: "10000", NextHop: l.VTEP}
			st := "r*"
			if isEVPNRouteInstalled(candidate, macMap, arpMap, routeMap, activeVTEPs) {
				st = "u*>"
			}

			evpnMap[k5] = EVPNRouteEntry{
				RouteType:  5,
				RD:         rd,
				RT:         "10000:10000",
				VNI:        "10000",
				Prefix:     pfx,
				NextHop:    l.VTEP,
				Neighbor:   l.Neighbor,
				Originator: "default",
				Status:     st,
				PathVersions: []EVPNPathVersion{
					{Neighbor: "10.1.10.10", NextHop: l.VTEP, StatusCode: st, PathID: 0},
					{Neighbor: "10.1.20.20", NextHop: l.VTEP, StatusCode: "*", PathID: 0},
				},
			}
		}
	}

	state.EVPNRoutes = make([]EVPNRouteEntry, 0, len(evpnMap))
	for _, v := range evpnMap {
		if !isSelfOriginatedEVPNRoute(v, routeMap, arpMap) {
			state.EVPNRoutes = append(state.EVPNRoutes, v)
		}
	}
}
