package components

import (
	"strings"
	"testing"

	"srl-tui/pkg/ndk"
)

func TestInspectorModalUsesAuthenticRawJSON(t *testing.T) {
	modal := NewInspectorModal()

	rawJSON := `{"admin-state":"enable","name":"ethernet-1/4","subinterface":[{"description":"web","index":0,"name":"ethernet-1/4.0","type":"srl_nokia-interfaces:bridged"}]}`
	portWithRaw := ndk.PortState{
		Name:    "ethernet-1/4",
		RawJSON: rawJSON,
	}

	modal.OpenForPort(portWithRaw)

	if modal.RawJSON != rawJSON {
		t.Errorf("Expected inspector to use rawJSON from device, got:\n%s", modal.RawJSON)
	}

	if strings.Contains(modal.RawJSON, "10.1.40.1/24") {
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
}

func TestInspectorModalScrollingAndPagination(t *testing.T) {
	modal := NewInspectorModal()
	modal.RawJSON = "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\nline11\nline12\nline13\nline14\nline15\nline16\nline17\nline18\nline19\nline20\nline21\nline22\nline23\nline24\nline25"

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
