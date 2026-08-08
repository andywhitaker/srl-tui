package ndk

import (
	"math"
	"math/rand"
	"sync"
	"time"
)

type Simulator struct {
	state     *TelemetryState
	running   bool
	stopChan  chan struct{}
	mu        sync.Mutex
	tickCount uint64
}

func NewSimulator(state *TelemetryState) *Simulator {
	return &Simulator{
		state:    state,
		stopChan: make(chan struct{}),
	}
}

func (sim *Simulator) Start() {
	sim.mu.Lock()
	if sim.running {
		sim.mu.Unlock()
		return
	}
	sim.running = true
	sim.mu.Unlock()

	sim.state.Lock()
	sim.state.Hostname = "srlinux-leaf1"
	sim.state.Platform = "7220 IXR-D2"
	sim.state.OSVersion = "v24.10.1"
	sim.state.DemoMode = true
	sim.state.NDKConnected = true
	sim.state.SyncState = "READY"
	sim.state.SyncMessage = "DEMO SIMULATOR ACTIVE"
	sim.state.Unlock()

	sim.seedInitialData()
	go sim.runLoop()
}

func (sim *Simulator) seedInitialData() {
	sim.state.Lock()
	defer sim.state.Unlock()

	// Setup port statuses
	for i := range sim.state.Ports {
		if i == 0 || i == 1 || i == 2 || i == 3 || i == 4 || i == 28 {
			sim.state.Ports[i].AdminState = "up"
			sim.state.Ports[i].OperState = "up"
			sim.state.Ports[i].Speed = "25G"
			if i == 28 {
				sim.state.Ports[i].Speed = "100G"
				sim.state.Ports[i].Description = "Uplink to Spine-1"
			} else {
				sim.state.Ports[i].Description = "Access Host"
			}
		} else {
			sim.state.Ports[i].AdminState = "down"
			sim.state.Ports[i].OperState = "down"
			sim.state.Ports[i].Speed = "25G"
		}
	}

	// Pre-populate BGP Peers
	sim.state.BGPPeers = []BGPPeerState{
		{NeighborIP: "10.0.0.1", PeerASN: 65001, LocalASN: 65000, SessionState: "ESTABLISHED", PeerType: "eBGP", Interface: "ethernet-1/1.0", Uptime: "14d 02h", RxPrefixes: 43, TxPrefixes: 13},
		{NeighborIP: "10.0.0.3", PeerASN: 65002, LocalASN: 65000, SessionState: "ESTABLISHED", PeerType: "eBGP", Interface: "ethernet-1/2.0", Uptime: "14d 02h", RxPrefixes: 43, TxPrefixes: 55},
		{NeighborIP: "10.0.0.5", PeerASN: 65000, LocalASN: 65000, SessionState: "ACTIVE", PeerType: "iBGP", Interface: "ethernet-1/29.0", Uptime: "-", RxPrefixes: 0, TxPrefixes: 0},
	}

	// Pre-populate LLDP Neighbors
	sim.state.LLDPNeighbors = []LLDPNeighbor{
		{LocalPort: "ethernet-1/1", SysName: "spine-01.srl", RemotePort: "ethernet-1/31"},
		{LocalPort: "ethernet-1/2", SysName: "spine-02.srl", RemotePort: "ethernet-1/31"},
		{LocalPort: "ethernet-1/29", SysName: "leaf-02.srl", RemotePort: "ethernet-1/29"},
	}

	// Pre-populate ARP & MAC Tables
	sim.state.ARPTables = []ARPEntry{
		{IPAddress: "192.168.10.1", MACAddress: "00:50:56:A1:B2:C3", Interface: "ethernet-1/3.0", NetInst: "default", EntryType: "dynamic", ExpirySec: 280},
		{IPAddress: "192.168.10.2", MACAddress: "00:50:56:A1:B2:C4", Interface: "ethernet-1/4.0", NetInst: "default", EntryType: "dynamic", ExpirySec: 150},
		{IPAddress: "192.168.20.1", MACAddress: "00:50:56:FE:DC:BA", Interface: "ethernet-1/5.0", NetInst: "vlan-20", EntryType: "static", ExpirySec: 0},
	}

	sim.state.MACTables = []MACTableEntry{
		{MACAddress: "00:50:56:A1:B2:C3", NetInst: "default", Interface: "ethernet-1/3.0", Type: "learned"},
		{MACAddress: "00:50:56:A1:B2:C4", NetInst: "default", Interface: "ethernet-1/4.0", Type: "learned"},
		{MACAddress: "00:50:56:FE:DC:BA", NetInst: "vlan-20", Interface: "vxlan0.1", Type: "evpn"},
	}

	// Pre-populate IP Route Table
	sim.state.RouteTable = []RouteEntry{
		{Prefix: "0.0.0.0/0", NextHop: "10.0.0.1", Protocol: "bgp", Preference: 170, Metric: 0, NetInst: "default"},
		{Prefix: "10.0.0.1/32", NextHop: "ethernet-1/1.0", Protocol: "direct", Preference: 0, Metric: 0, NetInst: "default"},
		{Prefix: "10.0.0.3/32", NextHop: "ethernet-1/2.0", Protocol: "direct", Preference: 0, Metric: 0, NetInst: "default"},
		{Prefix: "10.0.0.1/32", NextHop: "ethernet-1/29.0", Protocol: "bgp", Preference: 170, Metric: 10, NetInst: "default"},
		{Prefix: "100.64.0.0/16", NextHop: "10.0.0.3", Protocol: "bgp", Preference: 170, Metric: 20, NetInst: "default"},
		{Prefix: "192.168.10.0/24", NextHop: "direct", Protocol: "direct", Preference: 0, Metric: 0, NetInst: "default"},
		{Prefix: "192.168.20.0/24", NextHop: "direct", Protocol: "direct", Preference: 0, Metric: 0, NetInst: "vlan-20"},
	}

	// Pre-populate EVPN Routes (Types 1-5)
	sim.state.EVPNRoutes = []EVPNRouteEntry{
		{RouteType: 1, RD: "10.0.0.1:100", RT: "65000:100", ESI: "00:11:22:33:44:55:66:77:88:99", VNI: "10010", NextHop: "10.0.0.1", Neighbor: "10.0.0.1", Originator: "default", Status: "u*>"},
		{RouteType: 2, RD: "10.0.0.1:100", RT: "65000:100", VNI: "10010", MAC: "00:50:56:A1:B2:C3", IP: "192.168.10.1", NextHop: "10.0.0.1", Neighbor: "10.0.0.1", Originator: "default", Status: "u*>"},
		{RouteType: 2, RD: "10.0.0.1:100", RT: "65000:100", VNI: "10010", MAC: "00:50:56:A1:B2:C4", IP: "192.168.10.2", NextHop: "10.0.0.1", Neighbor: "10.0.0.1", Originator: "default", Status: "u*>"},
		{RouteType: 3, RD: "10.0.0.1:100", RT: "65000:100", VNI: "10010", NextHop: "10.0.0.1", Neighbor: "10.0.0.1", Originator: "default", Status: "u*>"},
		{RouteType: 4, RD: "10.0.0.1:100", RT: "65000:100", ESI: "00:11:22:33:44:55:66:77:88:99", NextHop: "10.0.0.1", Neighbor: "10.0.0.1", Originator: "default", Status: "u*>"},
		{RouteType: 5, RD: "10.0.0.1:100", RT: "65000:100", VNI: "10000", Prefix: "10.200.1.0/24", NextHop: "10.0.0.1", Neighbor: "10.0.0.1", Originator: "default", Status: "u*>"},
	}
}

func (sim *Simulator) runLoop() {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	tStep := 0.0

	for {
		select {
		case <-sim.stopChan:
			return
		case <-ticker.C:
			sim.tickCount++
			tStep += 0.1

			// Synthetic sine-wave traffic pattern
			ingBps := math.Abs(math.Sin(tStep*0.5)*8.5e9 + rand.Float64()*5.0e8)
			egBps := math.Abs(math.Cos(tStep*0.5)*6.2e9 + rand.Float64()*4.0e8)

			sim.state.Lock()
			sim.state.EventCount += uint64(rand.Intn(5) + 1)
			sim.state.EventsPerSec = float64(rand.Intn(12) + 8)
			sim.state.Uptime = time.Since(sim.state.StartTime)

			// Update per-port simulated stats
			for i := range sim.state.Ports {
				if sim.state.Ports[i].OperState == "up" {
					sim.state.Ports[i].RxBps = ingBps * (0.8 + rand.Float64()*0.4) / 4.0
					sim.state.Ports[i].TxBps = egBps * (0.8 + rand.Float64()*0.4) / 4.0
					sim.state.Ports[i].RxPps = sim.state.Ports[i].RxBps / 1200.0
					sim.state.Ports[i].TxPps = sim.state.Ports[i].TxBps / 1200.0
					sim.state.Ports[i].UtilPercent = math.Min(100.0, (sim.state.Ports[i].RxBps/4.0e11)*100.0)
				}
			}

			sim.state.Unlock()
		}
	}
}

func (sim *Simulator) Stop() {
	sim.mu.Lock()
	defer sim.mu.Unlock()
	if !sim.running {
		return
	}
	sim.running = false
	close(sim.stopChan)
}
