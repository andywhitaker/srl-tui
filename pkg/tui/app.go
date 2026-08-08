package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"srl-tui/pkg/ndk"
	"srl-tui/pkg/tui/components"
	"srl-tui/pkg/tui/theme"
)

type TickMsg time.Time

type EVPNDetailModal struct {
	Active bool
	Entry  ndk.EVPNRouteEntry
}

type RouteDetailModal struct {
	Active bool
	Entry  ndk.RouteEntry
}

type LLDPDetailModal struct {
	Active bool
	Entry  ndk.LLDPNeighbor
}

type Model struct {
	ctx              context.Context
	state            *TelemetryStateWrapper
	theme            theme.Palette
	activeTab        components.TabID
	selectedPort     int
	evpnFilter       components.EVPNTypeFilter
	evpnSelIdx       int
	showUnimportedEVPN bool
	arpMacView       *components.ARPMACView
	lldpView         *components.LLDPView
	routeView        *components.RouteView
	showHelp         bool
	inspector        components.InspectorModal
	evpnModal        EVPNDetailModal
	routeModal       RouteDetailModal
	lldpModal        LLDPDetailModal
	pageSearchInput  textinput.Model
	pageSearchActive bool
	width            int
	height           int
}

type TelemetryStateWrapper struct {
	state *ndk.TelemetryState
}

func NewModel(ctx context.Context, state *ndk.TelemetryState, initialTheme theme.Palette) Model {
	ti := textinput.New()
	ti.Placeholder = "Search query..."
	ti.CharLimit = 40
	ti.Width = 30

	return Model{
		ctx:              ctx,
		state:            &TelemetryStateWrapper{state: state},
		theme:            initialTheme,
		activeTab:        components.TabPorts,
		selectedPort:     0,
		evpnFilter:       components.EVPNFilterAll,
		evpnSelIdx:       0,
		arpMacView:       components.NewARPMACView(),
		lldpView:         components.NewLLDPView(),
		routeView:        components.NewRouteView(),
		showHelp:         false,
		inspector:        components.NewInspectorModal(),
		evpnModal:        EVPNDetailModal{Active: false},
		routeModal:       RouteDetailModal{Active: false},
		lldpModal:        LLDPDetailModal{Active: false},
		pageSearchInput:  ti,
		pageSearchActive: false,
		width:            120,
		height:           32,
	}
}

func (m Model) Init() tea.Cmd {
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case TickMsg:
		return m, tickCmd()

	case tea.KeyMsg:
		if m.pageSearchActive {
			switch msg.String() {
			case "enter":
				m.pageSearchActive = false
				m.pageSearchInput.Blur()
				return m, nil
			case "esc":
				m.pageSearchActive = false
				m.pageSearchInput.SetValue("")
				m.pageSearchInput.Blur()
				m.resetViewSelections()
				return m, nil
			default:
				oldVal := m.pageSearchInput.Value()
				var cmd tea.Cmd
				m.pageSearchInput, cmd = m.pageSearchInput.Update(msg)
				if m.pageSearchInput.Value() != oldVal {
					m.resetViewSelections()
				}
				return m, cmd
			}
		}

		if m.evpnModal.Active {
			if msg.String() == "esc" || msg.String() == "enter" || msg.String() == "q" {
				m.evpnModal.Active = false
			}
			return m, nil
		}

		if m.routeModal.Active {
			if msg.String() == "esc" || msg.String() == "enter" || msg.String() == "q" {
				m.routeModal.Active = false
			}
			return m, nil
		}

		if m.lldpModal.Active {
			if msg.String() == "esc" || msg.String() == "enter" || msg.String() == "q" {
				m.lldpModal.Active = false
			}
			return m, nil
		}

		if m.inspector.Active {
			visibleLines := m.height - 13
			if visibleLines < 3 {
				visibleLines = 3
			}
			totalLines := m.inspector.GetFilteredLineCount()

			switch msg.String() {
			case "esc", "ctrl+c":
				m.inspector.Active = false
				return m, nil
			case "down":
				m.inspector.ScrollDown(totalLines, visibleLines)
				return m, nil
			case "up":
				m.inspector.ScrollUp()
				return m, nil
			case "pgdown", "ctrl+d":
				m.inspector.PageDown(totalLines, visibleLines)
				return m, nil
			case "pgup", "ctrl+u":
				m.inspector.PageUp()
				return m, nil
			default:
				oldQuery := m.inspector.SearchInput.Value()
				var cmd tea.Cmd
				m.inspector.SearchInput, cmd = m.inspector.SearchInput.Update(msg)
				if m.inspector.SearchInput.Value() != oldQuery {
					m.inspector.ScrollOffset = 0
				}
				return m, cmd
			}
		}

		if m.showHelp {
			if msg.String() == "esc" || msg.String() == "?" || msg.String() == "q" {
				m.showHelp = false
			}
			return m, nil
		}

		switch msg.String() {
		case "esc":
			if m.pageSearchInput.Value() != "" {
				m.pageSearchInput.SetValue("")
			}

		case "q", "ctrl+c":
			return m, tea.Quit

		case "?":
			m.showHelp = !m.showHelp

		case "u", "U":
			m.showUnimportedEVPN = !m.showUnimportedEVPN
			m.evpnSelIdx = 0

		case "c", "t":
			m.theme = theme.GetNextTheme(m.theme.ID)

		case "tab":
			m.activeTab = (m.activeTab + 1) % 6

		case "shift+tab":
			m.activeTab = (m.activeTab + 5) % 6

		case "1":
			m.activeTab = components.TabPorts
		case "2":
			m.activeTab = components.TabTopology
		case "3":
			m.activeTab = components.TabArpMac
		case "4":
			m.activeTab = components.TabLLDP
		case "5":
			m.activeTab = components.TabRoutes
		case "6":
			m.activeTab = components.TabEVPN

		case "enter":
			snap := m.state.state.Snapshot()
			if m.activeTab == components.TabPorts && m.selectedPort >= 0 && m.selectedPort < len(snap.Ports) {
				m.inspector.OpenForPort(snap.Ports[m.selectedPort])
			} else if m.activeTab == components.TabLLDP {
				neighbors := components.GetFilteredLLDP(snap, m.pageSearchInput.Value())
				if m.lldpView.SelectedIdx >= 0 && m.lldpView.SelectedIdx < len(neighbors) {
					m.lldpModal = LLDPDetailModal{
						Active: true,
						Entry:  neighbors[m.lldpView.SelectedIdx],
					}
				}
			} else if m.activeTab == components.TabRoutes {
				routes := components.GetFilteredRoutes(snap, m.pageSearchInput.Value())
				if m.routeView.SelectedIdx >= 0 && m.routeView.SelectedIdx < len(routes) {
					m.routeModal = RouteDetailModal{
						Active: true,
						Entry:  routes[m.routeView.SelectedIdx],
					}
				}
			} else if m.activeTab == components.TabEVPN {
				routes := components.GetFilteredEVPNRoutes(snap, m.evpnFilter, m.pageSearchInput.Value(), m.showUnimportedEVPN)
				if m.evpnSelIdx >= 0 && m.evpnSelIdx < len(routes) {
					m.evpnModal = EVPNDetailModal{
						Active: true,
						Entry:  routes[m.evpnSelIdx],
					}
				}
			}

		case " ", "space", "a", "m":
			if m.activeTab == components.TabArpMac {
				m.arpMacView.TogglePane()
			}

		case "/":
			snap := m.state.state.Snapshot()
			if m.activeTab == components.TabPorts && m.selectedPort >= 0 && m.selectedPort < len(snap.Ports) {
				m.inspector.OpenForPort(snap.Ports[m.selectedPort])
			} else {
				m.pageSearchActive = true
				m.pageSearchInput.Focus()
			}

		// Navigation depending on Active Tab
		case "right", "l":
			if m.activeTab == components.TabPorts {
				if m.selectedPort+2 < len(m.state.state.Ports) {
					m.selectedPort += 2
				}
			} else if m.activeTab == components.TabEVPN {
				m.evpnFilter = (m.evpnFilter + 1) % 6
				m.evpnSelIdx = 0
			}

		case "left", "h":
			if m.activeTab == components.TabPorts {
				if m.selectedPort >= 2 {
					m.selectedPort -= 2
				}
			} else if m.activeTab == components.TabEVPN {
				m.evpnFilter = (m.evpnFilter + 5) % 6
				m.evpnSelIdx = 0
			}

		case "down", "j":
			if m.activeTab == components.TabPorts {
				if m.selectedPort%2 == 0 && m.selectedPort+1 < len(m.state.state.Ports) {
					m.selectedPort += 1
				}
			} else if m.activeTab == components.TabArpMac {
				m.arpMacView.ScrollDown()
			} else if m.activeTab == components.TabLLDP {
				snap := m.state.state.Snapshot()
				neighbors := components.GetFilteredLLDP(snap, m.pageSearchInput.Value())
				m.lldpView.ScrollDown(len(neighbors))
			} else if m.activeTab == components.TabRoutes {
				snap := m.state.state.Snapshot()
				routes := components.GetFilteredRoutes(snap, m.pageSearchInput.Value())
				m.routeView.ScrollDown(len(routes))
			} else if m.activeTab == components.TabEVPN {
				snap := m.state.state.Snapshot()
				filteredCount := components.GetFilteredEVPNCount(snap, m.evpnFilter, m.pageSearchInput.Value(), m.showUnimportedEVPN)
				m.clampEVPNSelection(filteredCount)
				if m.evpnSelIdx < filteredCount-1 {
					m.evpnSelIdx++
				}
			}

		case "up", "k":
			if m.activeTab == components.TabPorts {
				if m.selectedPort%2 == 1 {
					m.selectedPort -= 1
				}
			} else if m.activeTab == components.TabArpMac {
				m.arpMacView.ScrollUp()
			} else if m.activeTab == components.TabLLDP {
				m.lldpView.ScrollUp()
			} else if m.activeTab == components.TabRoutes {
				m.routeView.ScrollUp()
			} else if m.activeTab == components.TabEVPN {
				snap := m.state.state.Snapshot()
				filteredCount := components.GetFilteredEVPNCount(snap, m.evpnFilter, m.pageSearchInput.Value(), m.showUnimportedEVPN)
				m.clampEVPNSelection(filteredCount)
				if m.evpnSelIdx > 0 {
					m.evpnSelIdx--
				}
			}

		case "pgdown", "ctrl+d":
			if m.activeTab == components.TabArpMac {
				m.arpMacView.PageDown()
			} else if m.activeTab == components.TabLLDP {
				snap := m.state.state.Snapshot()
				neighbors := components.GetFilteredLLDP(snap, m.pageSearchInput.Value())
				for i := 0; i < 10; i++ {
					m.lldpView.ScrollDown(len(neighbors))
				}
			} else if m.activeTab == components.TabRoutes {
				snap := m.state.state.Snapshot()
				routes := components.GetFilteredRoutes(snap, m.pageSearchInput.Value())
				for i := 0; i < 10; i++ {
					m.routeView.ScrollDown(len(routes))
				}
			} else if m.activeTab == components.TabEVPN {
				snap := m.state.state.Snapshot()
				filteredCount := components.GetFilteredEVPNCount(snap, m.evpnFilter, m.pageSearchInput.Value(), m.showUnimportedEVPN)
				m.evpnSelIdx += 10
				if m.evpnSelIdx >= filteredCount {
					m.evpnSelIdx = filteredCount - 1
				}
			}

		case "pgup", "ctrl+u":
			if m.activeTab == components.TabArpMac {
				m.arpMacView.PageUp()
			} else if m.activeTab == components.TabLLDP {
				for i := 0; i < 10; i++ {
					m.lldpView.ScrollUp()
				}
			} else if m.activeTab == components.TabRoutes {
				for i := 0; i < 10; i++ {
					m.routeView.ScrollUp()
				}
			} else if m.activeTab == components.TabEVPN {
				m.evpnSelIdx -= 10
				if m.evpnSelIdx < 0 {
					m.evpnSelIdx = 0
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func overlayModal(base, modal string, width, height int) string {
	if width <= 0 || height <= 0 {
		return modal
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modal)
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing Cyber TUI terminal engine..."
	}

	snap := m.state.state.Snapshot()

	// 1. Header
	headerView := components.RenderHeader(snap, m.theme, m.width)

	// 2. Navigation Tabs
	tabsView := components.RenderTabBar(m.activeTab, m.theme, m.width)

	// 3. Main Workspace Area
	workspaceHeight := m.height - 7
	if workspaceHeight < 10 {
		workspaceHeight = 10
	}

	var mainView string
	searchQuery := m.pageSearchInput.Value()

	switch m.activeTab {
	case components.TabPorts:
		mainView = components.RenderPortMatrix(snap, m.selectedPort, true, m.theme, m.width, workspaceHeight)

	case components.TabTopology:
		mainView = components.RenderTopoMesh(snap, true, m.theme, m.width, workspaceHeight, searchQuery, m.pageSearchActive, m.pageSearchInput.View())

	case components.TabArpMac:
		mainView = m.arpMacView.Render(snap, m.width, workspaceHeight, m.theme, searchQuery, m.pageSearchActive, m.pageSearchInput.View())

	case components.TabLLDP:
		mainView = m.lldpView.Render(snap, m.theme, m.width, workspaceHeight, searchQuery, m.pageSearchActive, m.pageSearchInput.View())

	case components.TabRoutes:
		mainView = m.routeView.Render(snap, m.theme, m.width, workspaceHeight, searchQuery, m.pageSearchActive, m.pageSearchInput.View())

	case components.TabEVPN:
		mainView = components.RenderEVPNView(snap, m.evpnFilter, m.evpnSelIdx, m.theme, m.width, workspaceHeight, searchQuery, m.pageSearchActive, m.pageSearchInput.View(), m.showUnimportedEVPN)
	}

	// 4. Global Footer Help Bar
	footerView := RenderHelpBar(m.theme, m.width)

	fullView := lipgloss.JoinVertical(
		lipgloss.Left,
		headerView,
		tabsView,
		mainView,
		footerView,
	)

	// Render Modals Overlays
	if m.lldpModal.Active {
		modal := components.RenderLLDPDetailModal(m.lldpModal.Entry, snap, m.theme, m.width, m.height)
		return overlayModal(fullView, modal, m.width, m.height)
	}

	if m.routeModal.Active {
		modal := components.RenderRouteDetailModal(m.routeModal.Entry, snap, m.theme, m.width, m.height)
		return overlayModal(fullView, modal, m.width, m.height)
	}

	if m.evpnModal.Active {
		modal := components.RenderEVPNDetailModal(m.evpnModal.Entry, m.theme, m.width, m.height, snap)
		return overlayModal(fullView, modal, m.width, m.height)
	}

	if m.inspector.Active {
		inspectorView := components.RenderInspectorModal(m.inspector, m.theme, m.width, m.height)
		return overlayModal(fullView, inspectorView, m.width, m.height)
	}

	if m.showHelp {
		helpModal := RenderHelpOverlay(m.theme, m.width, m.height)
		return overlayModal(fullView, helpModal, m.width, m.height)
	}

	return fullView
}

func (m *Model) resetViewSelections() {
	m.evpnSelIdx = 0
	if m.routeView != nil {
		m.routeView.SelectedIdx = 0
	}
	if m.lldpView != nil {
		m.lldpView.SelectedIdx = 0
	}
}

func (m *Model) clampEVPNSelection(totalFilteredCount int) {
	if totalFilteredCount <= 0 {
		m.evpnSelIdx = 0
		return
	}
	if m.evpnSelIdx >= totalFilteredCount {
		m.evpnSelIdx = totalFilteredCount - 1
	}
	if m.evpnSelIdx < 0 {
		m.evpnSelIdx = 0
	}
}
