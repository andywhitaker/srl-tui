package components

import (
	"strings"
	"testing"

	"srl-tui/pkg/ndk"
	"srl-tui/pkg/tui/theme"
)

func TestInspectorModalUsesAuthenticRawJSONAndYAML(t *testing.T) {
	modal := NewInspectorModal()

	rawJSON := `{"admin-state":"enable","name":"ethernet-1/4","subinterface":[{"description":"web","index":0,"name":"ethernet-1/4.0","type":"srl_nokia-interfaces:bridged"}]}`
	portWithRaw := ndk.PortState{
		Name:    "ethernet-1/4",
		RawJSON: rawJSON,
	}

	modal.OpenForPort(portWithRaw)

	if modal.RawJSON != rawJSON {
		t.Errorf("Expected inspector to store rawJSON from device, got:\n%s", modal.RawJSON)
	}

	if !strings.Contains(modal.RawYAML, "admin-state: enable") {
		t.Errorf("Expected RawYAML to contain 'admin-state: enable', got:\n%s", modal.RawYAML)
	}

	if strings.Contains(modal.RawYAML, "10.1.40.1/24") {
		t.Errorf("BUG DETECTED: Inspector output contained fake IP prefix 10.1.40.1/24!")
	}
}

func TestInspectorModalFallbackNoFakeSubinterfaces(t *testing.T) {
	modal := NewInspectorModal()

	portNoRaw := ndk.PortState{
		Index:      3,
		Name:       "ethernet-1/4",
		AdminState: "down",
		OperState:  "down",
	}

	modal.OpenForPort(portNoRaw)

	if strings.Contains(modal.RawJSON, "10.1.40.1/24") {
		t.Errorf("BUG DETECTED: Inspector fallback contained fake IP prefix 10.1.40.1/24!")
	}

	if strings.Contains(modal.RawJSON, "subinterface") {
		t.Errorf("BUG DETECTED: Inspector fallback injected fake subinterface into unconfigured port!")
	}

	if !strings.Contains(modal.RawYAML, "admin-state: disable") {
		t.Errorf("Expected YAML fallback to contain 'admin-state: disable', got:\n%s", modal.RawYAML)
	}
}

func TestSplitPanePortModalRendering(t *testing.T) {
	modal := NewInspectorModal()
	port := ndk.PortState{
		Index:       0,
		Name:        "ethernet-1/1",
		ShortName:   "e1-1",
		AdminState:  "up",
		OperState:   "up",
		Speed:       "25G",
		MAC:         "00:11:22:33:44:55",
		MTU:         1500,
		RxBps:       1000000,
		TxBps:       2000000,
		Description: "Uplink to Core",
	}
	modal.OpenForPort(port)

	pal := theme.Cyberpunk
	out := RenderInspectorModal(modal, pal, 100, 30)

	if !strings.Contains(out, "PORT INSPECTOR & STATE DATATREE: ethernet-1/1") {
		t.Errorf("Expected title in split pane modal, got:\n%s", out)
	}

	if !strings.Contains(out, "ADMIN ENABLE") {
		t.Errorf("Expected top pane to contain ADMIN ENABLE, got:\n%s", out)
	}

	if !strings.Contains(out, "ethernet-1/1 State") {
		t.Errorf("Expected bottom pane to contain ethernet-1/1 State header, got:\n%s", out)
	}

}

func TestAdminConfirmationPromptOverlay(t *testing.T) {
	modal := NewInspectorModal()
	port := ndk.PortState{
		Name:       "ethernet-1/1",
		AdminState: "up",
		OperState:  "up",
	}
	modal.OpenForPort(port)
	modal.ConfirmPrompt = true
	modal.ConfirmAction = "disable"

	pal := theme.Cyberpunk
	out := RenderInspectorModal(modal, pal, 100, 30)

	if !strings.Contains(out, "CONFIRM PORT ADMIN STATE CHANGE (DISABLE)") {
		t.Errorf("Expected confirmation prompt header, got:\n%s", out)
	}

	if !strings.Contains(out, "Are you sure you want to DISABLE port ethernet-1/1?") {
		t.Errorf("Expected prompt message for disabling port, got:\n%s", out)
	}
}

func TestInspectorModalScrollingAndPagination(t *testing.T) {
	modal := NewInspectorModal()
	modal.RawYAML = "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\nline11\nline12\nline13\nline14\nline15\nline16\nline17\nline18\nline19\nline20\nline21\nline22\nline23\nline24\nline25"

	// Test ScrollDown 1 line
	modal.ScrollDown(25, 10)
	if modal.ScrollOffset != 1 {
		t.Errorf("Expected ScrollOffset 1 after ScrollDown, got %d", modal.ScrollOffset)
	}

	// Test PageDown (+10 lines)
	modal.PageDown(25, 10)
	if modal.ScrollOffset != 11 {
		t.Errorf("Expected ScrollOffset 11 after PageDown, got %d", modal.ScrollOffset)
	}

	// Test PageUp (-10 lines)
	modal.PageUp()
	if modal.ScrollOffset != 1 {
		t.Errorf("Expected ScrollOffset 1 after PageUp, got %d", modal.ScrollOffset)
	}

	// Test ScrollUp 1 line
	modal.ScrollUp()
	if modal.ScrollOffset != 0 {
		t.Errorf("Expected ScrollOffset 0 after ScrollUp, got %d", modal.ScrollOffset)
	}
}

func TestInspectorModalFilterInputAcceptsKKey(t *testing.T) {
	modal := NewInspectorModal()
	modal.SearchInput.Focus()
	modal.SearchInput.SetValue("link")

	if modal.SearchInput.Value() != "link" {
		t.Errorf("Expected SearchInput value 'link', got '%s'", modal.SearchInput.Value())
	}
}

func TestYAMLParentContextFilterPreservation(t *testing.T) {
	yamlInput := []string{
		"srl_nokia-interfaces:interface:",
		"  - name: ethernet-1/1",
		"    admin-state: enable",
		"    ethernet:",
		"      mac-address: 00:01:02:03:04:05",
		"      port-speed: 25G",
		"    mtu: 1500",
		"    oper-state: up",
		"    subinterface:",
		"      - index: 0",
		"        admin-state: enable",
		"        name: ethernet-1/1.0",
		"        oper-state: up",
	}


	modal := NewInspectorModal()
	modal.SchemaKeys = map[string]string{
		"interface":    "name",
		"subinterface": "index",
	}
	filtered := modal.filterYAMLLinesWithParents(yamlInput, "oper-state")


	expectedLines := []string{
		"srl_nokia-interfaces:interface:",
		"name: ethernet-1/1",
		"oper-state: up",
		"subinterface:",
		"index: 0",
		"oper-state: up",
	}

	for _, expected := range expectedLines {
		found := false
		for _, line := range filtered {
			if strings.Contains(line, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected line containing '%s' to be in filter result, but was missing!\nFiltered result:\n%s", expected, strings.Join(filtered, "\n"))
		}
	}


	// Must NOT contain non-matching leaf sibling lines like admin-state, mac-address, port-speed or mtu
	for _, line := range filtered {
		if strings.Contains(line, "admin-state") || strings.Contains(line, "mac-address") || strings.Contains(line, "port-speed") || strings.Contains(line, "mtu") {
			t.Errorf("Non-matching leaf sibling line '%s' should have been filtered out!", line)
		}
	}
}


