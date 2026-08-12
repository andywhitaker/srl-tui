package components

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
	"srl-tui/pkg/ndk"
	"srl-tui/pkg/tui/theme"
)

type InspectorModal struct {
	Active        bool
	SearchInput   textinput.Model
	TargetTitle   string
	RawJSON       string
	RawYAML       string
	Port          ndk.PortState
	IsPortModal   bool
	ScrollOffset  int
	ConfirmPrompt bool
	ConfirmAction string            // "enable" or "disable"
	KeyOrder      []string          // Stable key ordering across live telemetry updates
	SchemaKeys    map[string]string // Container Name -> Primary Schema Key Name (dynamically populated from device schema)
}


func NewInspectorModal() InspectorModal {
	ti := textinput.New()
	ti.Placeholder = "Type to fuzzy filter YANG state keys (e.g. 'oper-state', 'mtu', 'traffic')..."
	ti.CharLimit = 60
	ti.Width = 50

	return InspectorModal{
		Active:      false,
		SearchInput: ti,
	}
}

func (m *InspectorModal) ScrollUp() {
	if m.ScrollOffset > 0 {
		m.ScrollOffset--
	}
}

func (m *InspectorModal) ScrollDown(totalLines, visibleHeight int) {
	maxOffset := totalLines - visibleHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.ScrollOffset < maxOffset {
		m.ScrollOffset++
	}
}

func (m *InspectorModal) PageUp() {
	m.ScrollOffset -= 10
	if m.ScrollOffset < 0 {
		m.ScrollOffset = 0
	}
}

func (m *InspectorModal) PageDown(totalLines, visibleHeight int) {
	maxOffset := totalLines - visibleHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	m.ScrollOffset += 10
	if m.ScrollOffset > maxOffset {
		m.ScrollOffset = maxOffset
	}
}

func (m *InspectorModal) GetFilteredLineCount() int {
	filterTerm := strings.ToLower(m.SearchInput.Value())
	content := m.RawYAML
	if content == "" {
		content = m.RawJSON
	}
	lines := strings.Split(content, "\n")
	if filterTerm == "" {
		return len(lines)
	}
	filtered := m.filterYAMLLinesWithParents(lines, filterTerm)
	return len(filtered)
}



func (m *InspectorModal) OpenForPort(port ndk.PortState) {
	m.Active = true
	m.IsPortModal = true
	m.TargetTitle = fmt.Sprintf("PORT INSPECTOR & STATE DATATREE: %s", port.Name)
	m.ScrollOffset = 0
	m.ConfirmPrompt = false
	m.ConfirmAction = ""
	m.KeyOrder = nil // Reset key ordering sequence for new port
	m.SearchInput.Reset()
	m.SearchInput.Blur()

	m.UpdateYAMLFromPort(port)
}

func (m *InspectorModal) UpdateYAMLFromPort(port ndk.PortState) {
	m.Port = port

	var rawObj interface{}
	if port.RawJSON != "" {
		m.RawJSON = port.RawJSON
		_ = json.Unmarshal([]byte(port.RawJSON), &rawObj)
	}

	if rawObj != nil {
		syncRawObjWithPortState(rawObj, port)
	} else {
		adminStr := "disable"
		if strings.ToLower(port.AdminState) == "up" || strings.ToLower(port.AdminState) == "enable" {
			adminStr = "enable"
		}
		operStr := strings.ToLower(port.OperState)
		if operStr == "" {
			operStr = "down"
		}

		speedStr := port.Speed
		if speedStr == "" || speedStr == "..." {
			speedStr = "25G"
		}

		intfObj := map[string]interface{}{
			"name":        port.Name,
			"admin-state": adminStr,
			"oper-state":  operStr,
			"mtu":         port.MTU,
			"ethernet": map[string]interface{}{
				"port-speed":  speedStr,
				"mac-address": port.MAC,
			},
			"traffic-rates": map[string]interface{}{
				"in-bps":  port.RxBps,
				"out-bps": port.TxBps,
				"in-pps":  port.RxPps,
				"out-pps": port.TxPps,
			},
			"statistics": map[string]interface{}{
				"in-error-packets":    port.Errors,
				"carrier-transitions": port.Flaps,
			},
		}

		if port.Description != "" {
			intfObj["description"] = port.Description
		}

		rawObj = map[string]interface{}{
			"srl_nokia-interfaces:interface": []map[string]interface{}{intfObj},
		}

		bJSON, _ := json.MarshalIndent(rawObj, "", "  ")
		m.RawJSON = string(bJSON)
	}


	// 1. Format with stable key ordering
	bJSON, errM := json.Marshal(rawObj)
	if errM != nil {
		bJSON, _ = json.Marshal(rawObj)
	}

	var node yaml.Node
	if errY := yaml.Unmarshal(bJSON, &node); errY == nil {
		stabilizeYAMLNode(&node, "", &m.KeyOrder, m.SchemaKeys)
		if yBytes, errOut := yaml.Marshal(&node); errOut == nil {
			m.RawYAML = string(yBytes)
			return
		}
	}


	// Fallback if AST node conversion fails
	if yBytes, errY := yaml.Marshal(rawObj); errY == nil {
		m.RawYAML = string(yBytes)
	} else {
		m.RawYAML = m.RawJSON
	}
}

func syncRawObjWithPortState(rawObj interface{}, port ndk.PortState) {
	if rawObj == nil {
		return
	}

	if port.AdminState == "" && port.OperState == "" {
		return
	}

	var adminStr, operStr string
	if port.AdminState != "" {
		adminStr = "disable"
		if strings.ToLower(port.AdminState) == "up" || strings.ToLower(port.AdminState) == "enable" {
			adminStr = "enable"
		}
	}
	if port.OperState != "" {
		operStr = strings.ToLower(port.OperState)
	}

	updateMap := func(m map[string]interface{}) {
		if adminStr != "" {
			m["admin-state"] = adminStr
		}
		if operStr != "" {
			m["oper-state"] = operStr
		}
	}

	if m, ok := rawObj.(map[string]interface{}); ok {
		if _, hasAdmin := m["admin-state"]; hasAdmin {
			updateMap(m)
		} else if intfList, ok := m["srl_nokia-interfaces:interface"].([]interface{}); ok {
			for _, item := range intfList {
				if itemMap, ok := item.(map[string]interface{}); ok {
					updateMap(itemMap)
				}
			}
		} else {
			updateMap(m)
		}
	}
}



func stabilizeYAMLNode(node *yaml.Node, parentPath string, keyOrder *[]string, schemaKeys map[string]string) {
	if node == nil {
		return
	}
	node.Style = 0

	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			stabilizeYAMLNode(child, parentPath, keyOrder, schemaKeys)
		}

	case yaml.MappingNode:
		type keyValPair struct {
			keyNode *yaml.Node
			valNode *yaml.Node
			order   int
		}

		parentContainer := getContainerNameFromPath(parentPath)
		targetSchemaKey := ""
		if schemaKeys != nil {
			targetSchemaKey = schemaKeys[parentContainer]
		}

		var pairs []keyValPair
		for i := 0; i < len(node.Content); i += 2 {
			if i+1 >= len(node.Content) {
				break
			}
			kNode := node.Content[i]
			vNode := node.Content[i+1]
			kNode.Style = 0

			keyName := kNode.Value
			fullPath := keyName
			if parentPath != "" {
				fullPath = parentPath + "." + keyName
			}

			stabilizeYAMLNode(vNode, fullPath, keyOrder, schemaKeys)

			orderIdx := -1
			for idx, k := range *keyOrder {
				if k == fullPath {
					orderIdx = idx
					break
				}
			}
			if orderIdx == -1 {
				*keyOrder = append(*keyOrder, fullPath)
				orderIdx = len(*keyOrder) - 1
			}

			pairs = append(pairs, keyValPair{
				keyNode: kNode,
				valNode: vNode,
				order:   orderIdx,
			})
		}

		sort.SliceStable(pairs, func(i, j int) bool {
			if targetSchemaKey != "" {
				isKeyI := strings.EqualFold(pairs[i].keyNode.Value, targetSchemaKey)
				isKeyJ := strings.EqualFold(pairs[j].keyNode.Value, targetSchemaKey)
				if isKeyI && !isKeyJ {
					return true
				}
				if !isKeyI && isKeyJ {
					return false
				}
			}
			return pairs[i].order < pairs[j].order
		})

		var newContent []*yaml.Node
		for _, p := range pairs {
			newContent = append(newContent, p.keyNode, p.valNode)
		}
		node.Content = newContent

	case yaml.SequenceNode:
		for i, child := range node.Content {
			elemPath := fmt.Sprintf("%s[%d]", parentPath, i)
			stabilizeYAMLNode(child, elemPath, keyOrder, schemaKeys)
		}
	}
}

func getContainerNameFromPath(path string) string {
	if path == "" {
		return ""
	}
	if idx := strings.Index(path, "["); idx != -1 {
		path = path[:idx]
	}
	if idx := strings.LastIndex(path, "."); idx != -1 {
		path = path[idx+1:]
	}
	return path
}


func (m *InspectorModal) OpenForBGP(peer ndk.BGPPeerState) {
	m.Active = true
	m.IsPortModal = false
	m.TargetTitle = fmt.Sprintf("YANG State Tree: /network-instance[name=default]/protocols/bgp/neighbor[peer-address=%s]", peer.NeighborIP)
	m.ScrollOffset = 0
	m.ConfirmPrompt = false
	m.KeyOrder = nil
	m.SearchInput.Reset()
	m.SearchInput.Blur()

	dataMap := map[string]interface{}{
		"srl_nokia-bgp:neighbor": map[string]interface{}{
			"peer-address": peer.NeighborIP,
			"peer-as":      peer.PeerASN,
			"local-as":     peer.LocalASN,
			"session-state": peer.SessionState,
			"peer-type":    peer.PeerType,
			"admin-state":  "enable",
			"description":  fmt.Sprintf("eBGP peer to AS%d", peer.PeerASN),
			"transport": map[string]interface{}{
				"local-address": "10.0.0.2",
				"remote-port":  179,
				"local-port":   49210,
			},
			"afi-safi": []map[string]interface{}{
				{
					"afi-safi-name": "ipv4-unicast",
					"admin-state":   "enable",
					"prefixes": map[string]interface{}{
						"received": peer.RxPrefixes,
						"sent":     peer.TxPrefixes,
					},
				},
			},
			"timers": map[string]interface{}{
				"hold-time":      90,
				"keepalive-interval": 30,
				"negotiated-hold-time": 90,
			},
		},
	}

	bJSON, _ := json.MarshalIndent(dataMap, "", "  ")
	m.RawJSON = string(bJSON)

	var node yaml.Node
	if errY := yaml.Unmarshal([]byte(bJSON), &node); errY == nil {
		stabilizeYAMLNode(&node, "", &m.KeyOrder, m.SchemaKeys)
		if yBytes, errOut := yaml.Marshal(&node); errOut == nil {
			m.RawYAML = string(yBytes)
			return
		}
	}


	if yBytes, errY := yaml.Marshal(dataMap); errY == nil {
		m.RawYAML = string(yBytes)
	} else {
		m.RawYAML = string(bJSON)
	}
}

func RenderInspectorModal(m InspectorModal, pal theme.Palette, width, height int) string {
	if !m.Active {
		return ""
	}

	if width < 60 {
		width = 60
	}
	if height < 16 {
		height = 16
	}

	modalWidth := width - 8
	modalHeight := height - 4

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(pal.Background).
		Background(pal.Secondary).
		Padding(0, 1)

	if m.IsPortModal {
		return renderSplitPanePortModal(m, pal, modalWidth, modalHeight)
	}

	// Classic single-pane modal format (for BGP / generic YANG tree)
	filterTerm := strings.ToLower(m.SearchInput.Value())
	contentSource := m.RawYAML
	if contentSource == "" {
		contentSource = m.RawJSON
	}

	rawLines := strings.Split(contentSource, "\n")
	matchedLines := m.filterYAMLLinesWithParents(rawLines, filterTerm)
	var filteredLines []string


	for _, line := range matchedLines {
		filteredLines = append(filteredLines, colorizeYAMLLine(line, pal))
	}


	totalFiltered := len(filteredLines)
	visibleLines := modalHeight - 7
	if visibleLines < 3 {
		visibleLines = 3
	}

	startIdx := m.ScrollOffset
	if startIdx > totalFiltered-visibleLines && totalFiltered > visibleLines {
		startIdx = totalFiltered - visibleLines
	}
	if startIdx < 0 {
		startIdx = 0
	}

	endIdx := startIdx + visibleLines
	if endIdx > totalFiltered {
		endIdx = totalFiltered
	}

	var visibleSlice []string
	if totalFiltered > 0 && startIdx < totalFiltered {
		visibleSlice = filteredLines[startIdx:endIdx]
	}
	contentStr := strings.Join(visibleSlice, "\n")

	scrollInfo := ""
	if totalFiltered > 0 {
		scrollInfo = fmt.Sprintf(" [Lines %d-%d of %d | ↑/↓ or PgUp/PgDn to scroll] ", startIdx+1, endIdx, totalFiltered)
	}

	searchBar := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(pal.Primary).
		Background(pal.Background).
		Padding(0, 1).
		Render(fmt.Sprintf("🔍 Filter: %s", m.SearchInput.View()))

	footer := lipgloss.NewStyle().
		Foreground(pal.Subtext).
		Render(scrollInfo + "[Press ESC / q to exit]")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(pal.Secondary).
		BorderBackground(pal.Background).
		Background(pal.Background).
		Width(modalWidth).
		Height(modalHeight).
		Padding(1, 2)

	rawContent := fmt.Sprintf("%s\n\n%s\n\n%s\n%s",
		titleStyle.Render(m.TargetTitle),
		contentStr,
		searchBar,
		footer,
	)

	contentWidth := modalWidth - 4
	r, g, b, _ := pal.Background.RGBA()
	bgSeq := fmt.Sprintf("\x1b[48;2;%d;%d;%dm", uint8(r>>8), uint8(g>>8), uint8(b>>8))
	resetSeq := fmt.Sprintf("\x1b[0m%s", bgSeq)

	var styledLines []string
	for _, l := range strings.Split(rawContent, "\n") {
		w := lipgloss.Width(l)
		if w < contentWidth {
			l += strings.Repeat(" ", contentWidth-w)
		}
		cleaned := strings.ReplaceAll(l, "\x1b[0m", resetSeq)
		styledLines = append(styledLines, bgSeq+cleaned+"\x1b[0m")
	}

	return boxStyle.Render(strings.Join(styledLines, "\n"))
}

func renderSplitPanePortModal(m InspectorModal, pal theme.Palette, modalWidth, modalHeight int) string {
	p := m.Port

	// -------------------------------------------------------------
	// Top Pane: Detailed Port Status Overview
	// -------------------------------------------------------------
	labelStyle := lipgloss.NewStyle().Foreground(pal.Subtext).Background(pal.Background).Bold(true)
	valStyle := lipgloss.NewStyle().Foreground(pal.Text).Background(pal.Background)
	highlightStyle := lipgloss.NewStyle().Foreground(pal.Primary).Background(pal.Background).Bold(true)
	sepStyle := lipgloss.NewStyle().Foreground(pal.Primary).Background(pal.Background).Render("  │  ")
	bgStyle := lipgloss.NewStyle().Background(pal.Background)

	adminUp := strings.ToLower(p.AdminState) == "up" || strings.ToLower(p.AdminState) == "enable"
	operUp := strings.ToLower(p.OperState) == "up"

	adminBadge := lipgloss.NewStyle().Foreground(pal.Error).Background(pal.Background).Bold(true).Render("[ADMIN DISABLE]")
	if adminUp {
		adminBadge = lipgloss.NewStyle().Foreground(pal.Success).Background(pal.Background).Bold(true).Render("[ADMIN ENABLE]")
	}

	operBadge := lipgloss.NewStyle().Foreground(pal.Error).Background(pal.Background).Bold(true).Render("[OPER DOWN]")
	if operUp {
		operBadge = lipgloss.NewStyle().Foreground(pal.Success).Background(pal.Background).Bold(true).Render("[OPER UP]")
	}

	speedVal := p.Speed
	if speedVal == "" || speedVal == "..." {
		speedVal = "25G"
	}

	macVal := p.MAC
	if macVal == "" {
		macVal = "N/A"
	}

	descVal := p.Description
	if descVal == "" {
		descVal = "None"
	}

	rxStr := lipgloss.NewStyle().Foreground(pal.Success).Background(pal.Background).Bold(true).Render(formatBps(p.RxBps))
	txStr := lipgloss.NewStyle().Foreground(pal.Secondary).Background(pal.Background).Bold(true).Render(formatBps(p.TxBps))
	utilStr := fmt.Sprintf("%.2f%%", p.UtilPercent)
	if !operUp || !adminUp {
		rxStr = lipgloss.NewStyle().Foreground(pal.Muted).Background(pal.Background).Render("0 bps")
		txStr = lipgloss.NewStyle().Foreground(pal.Muted).Background(pal.Background).Render("0 bps")
		utilStr = "0.00%"
	}

	row1 := bgStyle.Render(" ") + labelStyle.Render("Port:") + bgStyle.Render(" ") + highlightStyle.Render(p.Name) + bgStyle.Render(" ") + valStyle.Render("("+p.ShortName+")") + sepStyle +
		labelStyle.Render("Admin:") + bgStyle.Render(" ") + adminBadge + sepStyle +
		labelStyle.Render("Oper:") + bgStyle.Render(" ") + operBadge + sepStyle +
		labelStyle.Render("Speed:") + bgStyle.Render(" ") + valStyle.Render(speedVal)

	row2 := bgStyle.Render(" ") + labelStyle.Render("MAC:") + bgStyle.Render(" ") + valStyle.Render(macVal) + sepStyle +
		labelStyle.Render("MTU:") + bgStyle.Render(" ") + valStyle.Render(fmt.Sprintf("%d", p.MTU)) + sepStyle +
		labelStyle.Render("Rx Traffic:") + bgStyle.Render(" ") + rxStr + sepStyle +
		labelStyle.Render("Tx Traffic:") + bgStyle.Render(" ") + txStr + sepStyle +
		labelStyle.Render("Util:") + bgStyle.Render(" ") + valStyle.Render(utilStr)

	row3 := bgStyle.Render(" ") + labelStyle.Render("In-Errors:") + bgStyle.Render(" ") + valStyle.Render(fmt.Sprintf("%d", p.Errors)) + sepStyle +
		labelStyle.Render("Flaps:") + bgStyle.Render(" ") + valStyle.Render(fmt.Sprintf("%d", p.Flaps)) + sepStyle +
		labelStyle.Render("Description:") + bgStyle.Render(" ") + valStyle.Render(descVal)

	// Quick action prompt line
	var actionHint string
	if adminUp {
		actionHint = lipgloss.NewStyle().Foreground(pal.Warning).Background(pal.Background).Bold(true).Render("⚡ Admin Controls: Press [a] to Disable Port (Admin Down)  │  Press [/] to Search Filter")
	} else {
		actionHint = lipgloss.NewStyle().Foreground(pal.Success).Background(pal.Background).Bold(true).Render("⚡ Admin Controls: Press [a] to Enable Port (Admin Up)  │  Press [/] to Search Filter")
	}

	topPaneBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(pal.Primary).
		BorderBackground(pal.Background).
		Background(pal.Background).
		Width(modalWidth - 6).
		Padding(0, 1).
		Render(fmt.Sprintf("%s\n%s\n%s\n\n%s", row1, row2, row3, actionHint))

	// -------------------------------------------------------------
	// Bottom Pane: YAML State Datastore View
	// -------------------------------------------------------------
	filterTerm := strings.ToLower(m.SearchInput.Value())
	rawLines := strings.Split(m.RawYAML, "\n")
	matchedLines := m.filterYAMLLinesWithParents(rawLines, filterTerm)
	var filteredLines []string


	for _, line := range matchedLines {
		filteredLines = append(filteredLines, colorizeYAMLLine(line, pal))
	}


	totalFiltered := len(filteredLines)
	topPaneHeight := 8
	bottomPaneHeight := modalHeight - topPaneHeight - 7
	if bottomPaneHeight < 4 {
		bottomPaneHeight = 4
	}

	visibleLines := bottomPaneHeight - 2
	if visibleLines < 2 {
		visibleLines = 2
	}

	startIdx := m.ScrollOffset
	if startIdx > totalFiltered-visibleLines && totalFiltered > visibleLines {
		startIdx = totalFiltered - visibleLines
	}
	if startIdx < 0 {
		startIdx = 0
	}

	endIdx := startIdx + visibleLines
	if endIdx > totalFiltered {
		endIdx = totalFiltered
	}

	var visibleSlice []string
	if totalFiltered > 0 && startIdx < totalFiltered {
		visibleSlice = filteredLines[startIdx:endIdx]
	}
	contentStr := strings.Join(visibleSlice, "\n")

	bottomHeader := lipgloss.NewStyle().
		Bold(true).
		Foreground(pal.Secondary).
		Render(fmt.Sprintf("📄 %s State", p.Name))

	searchBar := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(pal.Primary).
		Background(pal.Background).
		Padding(0, 1).
		Render(fmt.Sprintf("🔍 Filter: %s", m.SearchInput.View()))

	scrollInfo := ""
	if totalFiltered > 0 {
		scrollInfo = fmt.Sprintf(" [YAML Lines %d-%d of %d | ↑/↓ or PgUp/PgDn] ", startIdx+1, endIdx, totalFiltered)
	}

	footer := lipgloss.NewStyle().
		Foreground(pal.Subtext).
		Render(scrollInfo + "[Press ESC / q to exit modal]")

	outerBox := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(pal.Secondary).
		BorderBackground(pal.Background).
		Background(pal.Background).
		Width(modalWidth).
		Height(modalHeight).
		Padding(1, 2)

	modalTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(pal.Background).
		Background(pal.Secondary).
		Padding(0, 1).
		Render(m.TargetTitle)

	rawContent := fmt.Sprintf("%s\n\n%s\n\n%s\n%s\n\n%s\n%s",
		modalTitle,
		topPaneBox,
		bottomHeader,
		contentStr,
		searchBar,
		footer,
	)

	contentWidth := modalWidth - 4
	r, g, b, _ := pal.Background.RGBA()
	bgSeq := fmt.Sprintf("\x1b[48;2;%d;%d;%dm", uint8(r>>8), uint8(g>>8), uint8(b>>8))
	resetSeq := fmt.Sprintf("\x1b[0m%s", bgSeq)

	var styledLines []string
	for _, l := range strings.Split(rawContent, "\n") {
		w := lipgloss.Width(l)
		if w < contentWidth {
			l += strings.Repeat(" ", contentWidth-w)
		}
		cleaned := strings.ReplaceAll(l, "\x1b[0m", resetSeq)
		styledLines = append(styledLines, bgSeq+cleaned+"\x1b[0m")
	}

	modalContent := strings.Join(styledLines, "\n")
	renderedBase := outerBox.Render(modalContent)

	// -------------------------------------------------------------
	// Overlay Confirmation Dialog if ConfirmPrompt is active
	// -------------------------------------------------------------
	if m.ConfirmPrompt {
		confBox := renderAdminConfirmationPrompt(p.Name, m.ConfirmAction, pal, modalWidth-12)
		return lipgloss.Place(modalWidth+4, modalHeight+2, lipgloss.Center, lipgloss.Center, confBox)
	}

	return renderedBase
}

func renderAdminConfirmationPrompt(portName, action string, pal theme.Palette, width int) string {
	actionUpper := strings.ToUpper(action)
	actionColor := pal.Warning
	if action == "enable" {
		actionColor = pal.Success
	} else if action == "disable" {
		actionColor = pal.Error
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(pal.Background).
		Background(actionColor).
		Padding(0, 2).
		Render(fmt.Sprintf("⚠️ CONFIRM PORT ADMIN STATE CHANGE (%s)", actionUpper))

	promptMsg := fmt.Sprintf("Are you sure you want to %s port %s?\nThis will issue a gNMI SetRequest setting admin-state to '%s'.\nThe port state will update upon receiving authentic device telemetry.",
		lipgloss.NewStyle().Foreground(actionColor).Bold(true).Render(actionUpper),
		lipgloss.NewStyle().Foreground(pal.Primary).Bold(true).Render(portName),
		action,
	)

	btnConfirm := lipgloss.NewStyle().
		Bold(true).
		Foreground(pal.Background).
		Background(pal.Success).
		Padding(0, 2).
		Render("[Y / ENTER] Confirm & Apply")

	btnCancel := lipgloss.NewStyle().
		Bold(true).
		Foreground(pal.Background).
		Background(pal.Muted).
		Padding(0, 2).
		Render("[N / ESC] Cancel")

	buttons := fmt.Sprintf("%s    %s", btnConfirm, btnCancel)

	dialog := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(actionColor).
		BorderBackground(pal.Background).
		Background(pal.Background).
		Width(width).
		Padding(1, 3).
		Align(lipgloss.Center).
		Render(fmt.Sprintf("%s\n\n%s\n\n%s", title, promptMsg, buttons))

	return dialog
}

func colorizeYAMLLine(line string, pal theme.Palette) string {
	bgStyle := lipgloss.NewStyle().Background(pal.Background)
	keyStyle := lipgloss.NewStyle().Foreground(pal.Warning).Background(pal.Background).Bold(true)
	valStyle := lipgloss.NewStyle().Foreground(pal.Success).Background(pal.Background)
	bracketStyle := lipgloss.NewStyle().Foreground(pal.Primary).Background(pal.Background)
	dashStyle := lipgloss.NewStyle().Foreground(pal.Primary).Background(pal.Background).Bold(true)
	commentStyle := lipgloss.NewStyle().Foreground(pal.Subtext).Background(pal.Background)

	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") {
		return commentStyle.Render(line)
	}

	if strings.Contains(line, ":") {
		parts := strings.SplitN(line, ":", 2)
		prefix := parts[0]
		suffix := parts[1]

		if strings.HasPrefix(strings.TrimSpace(prefix), "- ") {
			dashIdx := strings.Index(prefix, "-")
			indent := prefix[:dashIdx]
			restKey := prefix[dashIdx+2:]
			return bgStyle.Render(indent) + dashStyle.Render("- ") + keyStyle.Render(restKey) + bracketStyle.Render(":") + valStyle.Render(suffix)
		}
		return keyStyle.Render(prefix) + bracketStyle.Render(":") + valStyle.Render(suffix)
	}

	return valStyle.Render(line)
}

// getFirstScalarKeyInBlock dynamically identifies the primary key line of a map block or list item
// from the received schema payload structure without any hardcoded key names.
func getFirstScalarKeyInBlock(lines []string, startLineIdx, endLineIdx int, parentContainer string, schemaKeys map[string]string) (int, bool) {
	targetSchemaKey := ""
	if schemaKeys != nil && parentContainer != "" {
		targetSchemaKey = schemaKeys[parentContainer]
	}

	// 1. Lookup registered schema key for parentContainer
	if targetSchemaKey != "" {
		for k := startLineIdx; k <= endLineIdx; k++ {
			line := lines[k]
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "- ") {
				trimmed = strings.TrimSpace(trimmed[2:])
			}
			if strings.Contains(trimmed, ":") {
				parts := strings.SplitN(trimmed, ":", 2)
				keyName := strings.TrimSpace(parts[0])
				if strings.EqualFold(keyName, targetSchemaKey) {
					val := strings.TrimSpace(parts[1])
					if val != "" && val != "{" && val != "[" {
						return k, true
					}
				}
			}
		}
	}

	// 2. Fallback ONLY if sequence list item block has list hyphen '- '
	if isListHyphenStartInBlock(lines, startLineIdx, endLineIdx) {
		for k := startLineIdx; k <= endLineIdx; k++ {
			line := lines[k]
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "- ") {
				parts := strings.SplitN(trimmed[2:], ":", 2)
				if len(parts) == 2 {
					val := strings.TrimSpace(parts[1])
					if val != "" && val != "{" && val != "[" {
						return k, true
					}
				}
			}
		}
	}
	return -1, false
}

func (m *InspectorModal) filterYAMLLinesWithParents(lines []string, filterTerm string) []string {
	if filterTerm == "" {
		return lines
	}

	filterTerm = strings.ToLower(filterTerm)
	matchedIndices := make(map[int]bool)
	primaryKeyIndices := make(map[int]bool)
	listItemHyphenNeeded := make(map[int]bool)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if strings.Contains(strings.ToLower(line), filterTerm) {
			matchedIndices[i] = true

			// Find ancestor container headers above line i
			currIndent := getYAMLIndent(line)
			for j := i - 1; j >= 0; j-- {
				parentLine := lines[j]
				if strings.TrimSpace(parentLine) == "" {
					continue
				}

				if isContainerHeaderLine(parentLine) {
					pIndent := getYAMLIndent(parentLine)
					if pIndent < currIndent {
						matchedIndices[j] = true
						parentContainer := getContainerNameFromLine(parentLine)

						// Dynamically identify primary key line of list item/container from schema payload
						if pkLine, ok := getFirstScalarKeyInBlock(lines, j+1, i, parentContainer, m.SchemaKeys); ok {
							matchedIndices[pkLine] = true
							primaryKeyIndices[pkLine] = true
							if strings.HasPrefix(strings.TrimSpace(lines[pkLine]), "- ") || isListHyphenStartInBlock(lines, j+1, pkLine) {
								listItemHyphenNeeded[pkLine] = true
							}
						} else if strings.HasPrefix(strings.TrimSpace(lines[i]), "- ") || isListHyphenStartInBlock(lines, j+1, i) {
							listItemHyphenNeeded[i] = true
						}

						currIndent = pIndent
						if currIndent == 0 {
							break
						}
					}
				}
			}
		}
	}



	var rawResult []string
	for i, line := range lines {
		if matchedIndices[i] {
			outLine := line
			if listItemHyphenNeeded[i] {
				trimmed := strings.TrimSpace(outLine)
				if strings.HasPrefix(trimmed, "- ") {
					trimmed = strings.TrimSpace(trimmed[2:])
				}
				indent := getYAMLIndent(line)
				outLine = strings.Repeat(" ", indent) + "- " + trimmed
			} else if primaryKeyIndices[i] {
				trimmed := strings.TrimSpace(outLine)
				if strings.HasPrefix(trimmed, "- ") {
					trimmed = strings.TrimSpace(trimmed[2:])
					indent := getYAMLIndent(line)
					outLine = strings.Repeat(" ", indent) + trimmed
				}
			}
			rawResult = append(rawResult, outLine)
		}
	}

	return alignYAMLLines(rawResult)
}

func alignYAMLLines(lines []string) []string {
	var aligned []string

	type stackEntry struct {
		rawIndent  int
		baseIndent int
		isListItem bool
	}
	stack := []stackEntry{{rawIndent: -1, baseIndent: 0, isListItem: false}}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		rawIndent := getYAMLIndent(line)
		hasHyphen := strings.HasPrefix(trimmed, "- ")

		for len(stack) > 1 && rawIndent < stack[len(stack)-1].rawIndent {
			stack = stack[:len(stack)-1]
		}

		top := stack[len(stack)-1]

		if hasHyphen {
			content := strings.TrimSpace(trimmed[2:])
			itemBaseIndent := top.baseIndent
			if top.rawIndent != -1 && rawIndent > top.rawIndent {
				itemBaseIndent = top.baseIndent + 2
			} else if top.rawIndent == -1 {
				itemBaseIndent = 2
			}
			alignedLine := strings.Repeat(" ", itemBaseIndent-2) + "- " + content
			aligned = append(aligned, alignedLine)

			stack = append(stack, stackEntry{
				rawIndent:  rawIndent,
				baseIndent: itemBaseIndent,
				isListItem: true,
			})
		} else if isContainerHeaderLine(line) {
			headerIndent := top.baseIndent
			if top.rawIndent != -1 && rawIndent > top.rawIndent {
				if top.isListItem {
					headerIndent = top.baseIndent
				} else {
					headerIndent = top.baseIndent + 2
				}
			}
			alignedLine := strings.Repeat(" ", headerIndent) + trimmed
			aligned = append(aligned, alignedLine)

			stack = append(stack, stackEntry{
				rawIndent:  rawIndent,
				baseIndent: headerIndent,
				isListItem: false,
			})
		} else {
			propIndent := top.baseIndent
			if top.isListItem {
				propIndent = top.baseIndent
			} else if top.rawIndent != -1 && rawIndent > top.rawIndent {
				propIndent = top.baseIndent + 2
			}
			alignedLine := strings.Repeat(" ", propIndent) + trimmed
			aligned = append(aligned, alignedLine)
		}
	}
	return aligned
}



func isListHyphenStartInBlock(lines []string, start, end int) bool {
	for k := start; k <= end; k++ {
		if strings.HasPrefix(strings.TrimSpace(lines[k]), "- ") {
			return true
		}
	}
	return false
}

func isContainerHeaderLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	if strings.HasPrefix(trimmed, "- ") {
		trimmed = strings.TrimSpace(trimmed[2:])
	}
	return strings.HasSuffix(trimmed, ":")
}

func getContainerNameFromLine(line string) string {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "- ") {
		trimmed = strings.TrimSpace(trimmed[2:])
	}
	if idx := strings.Index(trimmed, ":"); idx != -1 {
		return strings.TrimSpace(trimmed[:idx])
	}
	return trimmed
}

func getYAMLIndent(line string) int {

	indent := 0
	for _, ch := range line {
		if ch == ' ' {
			indent++
		} else if ch == '\t' {
			indent += 2
		} else {
			break
		}
	}
	return indent
}




