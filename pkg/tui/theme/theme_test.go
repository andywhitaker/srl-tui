package theme

import (
	"testing"
)

func TestThemeCyclingAndPalettes(t *testing.T) {
	if len(AllThemes) != 6 {
		t.Fatalf("Expected 6 themes, got %d", len(AllThemes))
	}

	// Test cycling from Cyberpunk -> Synthwave -> Matrix -> Monokai -> Cobalt2 -> Solarized Dark -> Cyberpunk
	t1 := GetNextTheme(CyberpunkTheme)
	if t1.ID != SynthwaveTheme {
		t.Errorf("Expected SynthwaveTheme, got %s", t1.ID)
	}

	t2 := GetNextTheme(SynthwaveTheme)
	if t2.ID != MatrixTheme {
		t.Errorf("Expected MatrixTheme, got %s", t2.ID)
	}

	t3 := GetNextTheme(MatrixTheme)
	if t3.ID != MonokaiTheme {
		t.Errorf("Expected MonokaiTheme, got %s", t3.ID)
	}

	t4 := GetNextTheme(MonokaiTheme)
	if t4.ID != Cobalt2Theme {
		t.Errorf("Expected Cobalt2Theme, got %s", t4.ID)
	}

	t5 := GetNextTheme(Cobalt2Theme)
	if t5.ID != SolarizedDarkTheme {
		t.Errorf("Expected SolarizedDarkTheme, got %s", t5.ID)
	}

	t6 := GetNextTheme(SolarizedDarkTheme)
	if t6.ID != CyberpunkTheme {
		t.Errorf("Expected CyberpunkTheme, got %s", t6.ID)
	}
}
