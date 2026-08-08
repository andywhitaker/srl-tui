package components

import (
	"strings"
	"testing"

	"srl-tui/pkg/ndk"
	"srl-tui/pkg/tui/theme"
)

func TestPortCellColorStates(t *testing.T) {
	pal := theme.Cyberpunk

	// 1. Admin down -> Grey color (pal.Muted)
	pAdminDown := ndk.PortState{Index: 0, AdminState: "down", OperState: "down"}
	cellGrey := renderPortCell(pAdminDown, false, pal)
	if cellGrey == "" {
		t.Errorf("Expected cell render string for admin down")
	}

	// 2. Admin up & Oper up -> Green color (pal.Success)
	pUp := ndk.PortState{Index: 0, AdminState: "up", OperState: "up"}
	cellGreen := renderPortCell(pUp, false, pal)
	if cellGreen == "" {
		t.Errorf("Expected cell render string for admin up + oper up")
	}

	// 3. Admin up & Oper down -> Red color (pal.Error)
	pOperDown := ndk.PortState{Index: 0, AdminState: "up", OperState: "down"}
	cellRed := renderPortCell(pOperDown, false, pal)
	if cellRed == "" {
		t.Errorf("Expected cell render string for admin up + oper down")
	}
}

func TestPortMatrixAll58PortsRendering(t *testing.T) {
	pal := theme.Cyberpunk
	state := ndk.NewTelemetryState(58)
	for i := 0; i < 58; i++ {
		state.Ports[i] = ndk.PortState{
			Index:      i,
			Name:       "ethernet-1/" + string(rune('1'+i%9)),
			ShortName:  "e1-" + string(rune('1'+i%9)),
			AdminState: "up",
			OperState:  "down",
		}
	}

	output := RenderPortMatrix(state, 57, true, pal, 140, 40)
	if !strings.Contains(output, "58 Physical Ethernet Ports") {
		t.Fatalf("Expected header to display 58 Physical Ethernet Ports, got: %s", output)
	}
	if !strings.Contains(output, "(ODD: 49..57)") || !strings.Contains(output, "(EVEN: 50..58)") {
		t.Fatalf("Expected block 2 to display odd/even ports 49..58, got: %s", output)
	}
}
