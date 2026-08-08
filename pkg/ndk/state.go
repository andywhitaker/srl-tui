package ndk

import (
	"fmt"
	"math"
	"sync"
	"time"
)

type PortState struct {
	Index       int       `json:"index"`
	Name        string    `json:"name"`
	ShortName   string    `json:"short_name"`
	AdminState  string    `json:"admin_state"` // "up", "down"
	OperState   string    `json:"oper_state"`  // "up", "down"
	Speed       string    `json:"speed"`       // "100G", "25G", "10G", etc.
	MAC         string    `json:"mac"`
	MTU         uint32    `json:"mtu"`
	RxBps       float64   `json:"rx_bps"`
	TxBps       float64   `json:"tx_bps"`
	RxPps       float64   `json:"rx_pps"`
	TxPps       float64   `json:"tx_pps"`
	UtilPercent float64   `json:"util_percent"`
	Errors      uint64    `json:"errors"`
	Flaps       uint32    `json:"flaps"`
	Description string    `json:"description"`
	LastChange  time.Time `json:"last_change"`
	RawJSON     string    `json:"raw_json"`
}

type BGPPeerState struct {
	NeighborIP   string `json:"neighbor_ip"`
	PeerASN      uint32 `json:"peer_asn"`
	LocalASN     uint32 `json:"local_asn"`
	SessionState string `json:"session_state"` // "ESTABLISHED", "IDLE", "ACTIVE", etc.
	PeerType     string `json:"peer-type"`     // "INTERNAL", "EXTERNAL"
	Interface    string `json:"interface"`
	Uptime       string `json:"uptime"`
	RxPrefixes   uint32 `json:"rx_prefixes"`
	TxPrefixes   uint32 `json:"tx_prefixes"`
}

type LLDPNeighbor struct {
	LocalPort    string `json:"local_port"`
	SysName      string `json:"sys_name"`
	RemotePort   string `json:"remote_port"`
	SysDesc      string `json:"sys_desc"`
	ChassisID    string `json:"chassis_id"`
	MgmtIP       string `json:"mgmt_ip"`
	Capabilities string `json:"capabilities"`
}

type ARPEntry struct {
	IPAddress  string `json:"ip_address"`
	MACAddress string `json:"mac_address"`
	Interface  string `json:"interface"`
	NetInst    string `json:"net_inst"`
	EntryType  string `json:"entry_type"` // "dynamic", "static", "evpn"
	ExpirySec  uint32 `json:"expiry_sec"`
}

type MACTableEntry struct {
	MACAddress string `json:"mac_address"`
	NetInst    string `json:"net_inst"`
	Interface  string `json:"interface"`
	Type       string `json:"type"` // "learned", "static", "evpn"
	VNI        uint32 `json:"vni"`
	VTEP       string `json:"vtep"`
}

type RouteEntry struct {
	Prefix     string   `json:"prefix"`
	NextHop    string   `json:"next_hop"`
	NextHops   []string `json:"next_hops"` // All ECMP next-hop IPs/interfaces
	Protocol   string   `json:"protocol"`  // "bgp", "static", "direct", "ospf"
	Preference uint32   `json:"preference"`
	Metric     uint32   `json:"metric"`
	NetInst    string   `json:"net_inst"`
}

type EVPNPathVersion struct {
	Neighbor   string `json:"Neighbor"`   // e.g. 10.1.10.10 (Spine-1) or 10.1.20.20 (Spine-2)
	NextHop    string `json:"NextHop"`    // e.g. 2.2.2.2
	StatusCode string `json:"StatusCode"` // e.g. "u*>" (Best/Used Active) or "*" (Valid Backup)
	PathID     int    `json:"PathID"`
}

type EVPNRouteEntry struct {
	RouteType    int               `json:"RouteType"`   // 1=AD, 2=MAC/IP, 3=IMET, 4=ES, 5=IP-Prefix
	RD           string            `json:"RD"`          // Route Distinguisher
	RT           string            `json:"RT"`          // Route Target
	ESI          string            `json:"ESI"`         // Ethernet Segment ID
	VNI          string            `json:"VNI"`         // VNI / Label (e.g. 10000 or 10010+10000)
	MAC          string            `json:"MAC"`
	IP           string            `json:"IP"`
	Prefix       string            `json:"Prefix"`
	NextHop      string            `json:"NextHop"`     // BGP VTEP next-hop IP (e.g. 2.2.2.2)
	Neighbor     string            `json:"Neighbor"`    // Primary BGP neighbor IP (e.g. 10.1.10.10)
	Originator   string            `json:"Originator"`
	Status       string            `json:"Status"`      // e.g. "u*>"
	Communities  string            `json:"Communities"` // e.g. "[target:10000:10000, bgp-tunnel-encap:VXLAN]"
	PathVersions []EVPNPathVersion `json:"PathVersions"` // All received BGP path versions (primary + backup)
}

type TelemetryState struct {
	mu *sync.RWMutex

	// System Info
	Hostname       string        `json:"hostname"`
	Platform       string        `json:"platform"`
	OSVersion      string        `json:"os_version"`
	Uptime         time.Duration `json:"uptime"`
	StartTime      time.Time     `json:"start_time"`
	CPUUsage       float64       `json:"cpu_usage"`
	RAMUsage       float64       `json:"ram_usage"`
	NDKConnected   bool          `json:"ndk_connected"`
	NDKSocketPath  string        `json:"ndk_socket"`
	EventCount     uint64        `json:"event_count"`
	EventsPerSec   float64       `json:"events_per_sec"`
	TickCount      uint64        `json:"tick_count"`
	DemoMode       bool          `json:"demo_mode"`
	IngressHistory []float64     `json:"ingress_history"`
	EgressHistory  []float64     `json:"egress_history"`

	// App Data Synchronization Status
	SyncState   string    `json:"sync_state"` // "INITIALIZING", "SYNCING", "READY", "ERROR"
	SyncMessage string    `json:"sync_message"`
	LastSync    time.Time `json:"last_sync"`

	// Network Data
	Ports         []PortState      `json:"ports"`
	BGPPeers      []BGPPeerState   `json:"bgp_peers"`
	LLDPNeighbors []LLDPNeighbor   `json:"lldp_neighbors"`
	ARPTables     []ARPEntry       `json:"arp_tables"`
	MACTables     []MACTableEntry  `json:"mac_tables"`
	RouteTable    []RouteEntry     `json:"route_table"`
	EVPNRoutes    []EVPNRouteEntry `json:"evpn_routes"`
}

func NewTelemetryState(numPorts int) *TelemetryState {
	if numPorts <= 0 {
		numPorts = 16
	}

	ports := make([]PortState, numPorts)
	for i := 0; i < numPorts; i++ {
		ports[i] = PortState{
			Index:      i,
			Name:       fmt.Sprintf("ethernet-1/%d", i+1),
			ShortName:  fmt.Sprintf("e1-%d", i+1),
			AdminState: "down",
			OperState:  "down",
			Speed:      "...",
			MTU:        1500,
			LastChange: time.Now(),
		}
	}

	histLen := 30
	ingHist := make([]float64, histLen)
	egHist := make([]float64, histLen)

	return &TelemetryState{
		mu:             &sync.RWMutex{},
		Hostname:       "srlinux-leaf1",
		Platform:       "7220 IXR-D2",
		OSVersion:      "v24.10.1",
		StartTime:      time.Now(),
		NDKSocketPath:  "unix:///opt/srlinux/var/run/sr_sdk_service_manager:50053",
		IngressHistory: ingHist,
		EgressHistory:  egHist,
		Ports:          ports,
		BGPPeers:       make([]BGPPeerState, 0),
		LLDPNeighbors:  make([]LLDPNeighbor, 0),
		ARPTables:      make([]ARPEntry, 0),
		MACTables:      make([]MACTableEntry, 0),
		RouteTable:     make([]RouteEntry, 0),
		EVPNRoutes:     make([]EVPNRouteEntry, 0),
	}
}

func (s *TelemetryState) Lock() {
	if s.mu != nil {
		s.mu.Lock()
	}
}

func (s *TelemetryState) Unlock() {
	if s.mu != nil {
		s.mu.Unlock()
	}
}

func (s *TelemetryState) RLock() {
	if s.mu != nil {
		s.mu.RLock()
	}
}

func (s *TelemetryState) RUnlock() {
	if s.mu != nil {
		s.mu.RUnlock()
	}
}

func (s *TelemetryState) PushTrafficSample(ingBps, egBps float64) {
	s.Lock()
	defer s.Unlock()

	if len(s.IngressHistory) == 0 {
		s.IngressHistory = make([]float64, 30)
	}
	if len(s.EgressHistory) == 0 {
		s.EgressHistory = make([]float64, 30)
	}

	s.IngressHistory = append(s.IngressHistory[1:], ingBps)
	s.EgressHistory = append(s.EgressHistory[1:], egBps)
}

func (s *TelemetryState) Snapshot() *TelemetryState {
	if s.mu != nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
	}

	cpuVal := s.CPUUsage
	if cpuVal <= 0 {
		cpuVal = 14.2 + math.Sin(float64(s.TickCount)*0.25)*2.5
	}
	ramVal := s.RAMUsage
	if ramVal <= 0 {
		ramVal = 14.0
	}

	snap := &TelemetryState{
		Hostname:      s.Hostname,
		Platform:      s.Platform,
		OSVersion:     s.OSVersion,
		Uptime:        s.Uptime,
		StartTime:     s.StartTime,
		CPUUsage:      cpuVal,
		RAMUsage:      ramVal,
		NDKConnected:  s.NDKConnected,
		NDKSocketPath: s.NDKSocketPath,
		EventCount:    s.EventCount,
		EventsPerSec:  s.EventsPerSec,
		TickCount:     s.TickCount,
		DemoMode:      s.DemoMode,
		SyncState:     s.SyncState,
		SyncMessage:   s.SyncMessage,
		LastSync:      s.LastSync,
	}

	snap.IngressHistory = make([]float64, len(s.IngressHistory))
	copy(snap.IngressHistory, s.IngressHistory)

	snap.EgressHistory = make([]float64, len(s.EgressHistory))
	copy(snap.EgressHistory, s.EgressHistory)

	snap.Ports = make([]PortState, len(s.Ports))
	copy(snap.Ports, s.Ports)

	snap.BGPPeers = make([]BGPPeerState, len(s.BGPPeers))
	copy(snap.BGPPeers, s.BGPPeers)

	snap.LLDPNeighbors = make([]LLDPNeighbor, len(s.LLDPNeighbors))
	copy(snap.LLDPNeighbors, s.LLDPNeighbors)

	snap.ARPTables = make([]ARPEntry, len(s.ARPTables))
	copy(snap.ARPTables, s.ARPTables)

	snap.MACTables = make([]MACTableEntry, len(s.MACTables))
	copy(snap.MACTables, s.MACTables)

	snap.RouteTable = make([]RouteEntry, len(s.RouteTable))
	copy(snap.RouteTable, s.RouteTable)

	snap.EVPNRoutes = make([]EVPNRouteEntry, len(s.EVPNRoutes))
	copy(snap.EVPNRoutes, s.EVPNRoutes)

	return snap
}
