package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"srl-tui/pkg/tui/theme"
)

type TabID int

const (
	TabPorts TabID = iota
	TabTopology
	TabArpMac
	TabLLDP
	TabRoutes
	TabEVPN
)

type TabInfo struct {
	ID   TabID
	Name string
	Key  string
	Icon string
}

var Tabs = []TabInfo{
	{ID: TabPorts, Name: "PORTS", Key: "1", Icon: "█"},
	{ID: TabTopology, Name: "BGP", Key: "2", Icon: "🔀"},
	{ID: TabArpMac, Name: "ARP & MAC", Key: "3", Icon: "🔍"},
	{ID: TabLLDP, Name: "LLDP NEIGHBORS", Key: "4", Icon: "🤝"},
	{ID: TabRoutes, Name: "ROUTES", Key: "5", Icon: "🔀"},
	{ID: TabEVPN, Name: "EVPN VXLAN", Key: "6", Icon: "☁️"},
}

func RenderTabBar(activeTab TabID, pal theme.Palette, width int) string {
	var renderedTabs []string

	for _, t := range Tabs {
		label := fmt.Sprintf("[%s: %s %s]", t.Key, t.Icon, t.Name)

		if t.ID == activeTab {
			style := lipgloss.NewStyle().
				Bold(true).
				Foreground(pal.Background).
				Background(pal.Primary).
				Padding(0, 1)
			renderedTabs = append(renderedTabs, style.Render(label))
		} else {
			style := lipgloss.NewStyle().
				Foreground(pal.Subtext).
				Background(pal.Surface).
				Padding(0, 1)
			renderedTabs = append(renderedTabs, style.Render(label))
		}
	}

	tabBarStr := strings.Join(renderedTabs, " ")

	boxStyle := lipgloss.NewStyle().
		Background(pal.Surface).
		Width(width - 2).
		Padding(0, 1)

	return boxStyle.Render(tabBarStr)
}
