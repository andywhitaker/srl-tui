package theme

import (
	"github.com/charmbracelet/lipgloss"
)

type ThemeID string

const (
	CyberpunkTheme     ThemeID = "cyberpunk"
	SynthwaveTheme     ThemeID = "synthwave"
	MatrixTheme        ThemeID = "matrix"
	MonokaiTheme       ThemeID = "monokai"
	Cobalt2Theme       ThemeID = "cobalt2"
	SolarizedDarkTheme ThemeID = "solarized"
)

type Palette struct {
	ID           ThemeID
	Name         string
	Primary      lipgloss.Color // Main accent (e.g. Neon Cyan)
	Secondary    lipgloss.Color // Secondary accent (e.g. Hot Pink)
	Success      lipgloss.Color // Green / Up state
	Warning      lipgloss.Color // Yellow / Amber state
	Error        lipgloss.Color // Red / Down state
	Muted        lipgloss.Color // Dimmed text / border
	Background   lipgloss.Color // Screen background
	Surface      lipgloss.Color // Pane / Card background
	Text         lipgloss.Color // Primary text
	Subtext      lipgloss.Color // Secondary text
	Highlight    lipgloss.Color // Selection highlight
	GraphIngress lipgloss.Color // Ingress graph line
	GraphEgress  lipgloss.Color // Egress graph line
}

var Cyberpunk = Palette{
	ID:           CyberpunkTheme,
	Name:         "CYBERPUNK NEON",
	Primary:      lipgloss.Color("#00F5FF"), // Neon Cyan
	Secondary:    lipgloss.Color("#FF007F"), // Hot Pink
	Success:      lipgloss.Color("#00FF66"), // Toxic Green
	Warning:      lipgloss.Color("#FFB000"), // Amber Gold
	Error:        lipgloss.Color("#FF0055"), // Electric Crimson
	Muted:        lipgloss.Color("#4A5568"), // Slate Gray
	Background:   lipgloss.Color("#0A0E17"), // Deep Cyber Space
	Surface:      lipgloss.Color("#0A0E17"), // Consistent Background Fill
	Text:         lipgloss.Color("#E2E8F0"), // Crisp Light
	Subtext:      lipgloss.Color("#8A99AD"), // Muted Blue Gray
	Highlight:    lipgloss.Color("#7928CA"), // Neon Purple Selection
	GraphIngress: lipgloss.Color("#00F5FF"), // Cyan
	GraphEgress:  lipgloss.Color("#FF007F"), // Pink
}

var Synthwave = Palette{
	ID:           SynthwaveTheme,
	Name:         "SYNTHWAVE NIGHT",
	Primary:      lipgloss.Color("#9D4EDD"), // Sunset Purple
	Secondary:    lipgloss.Color("#F72585"), // Neon Magenta
	Success:      lipgloss.Color("#4CC9F0"), // Cyber Blue
	Warning:      lipgloss.Color("#FF9E00"), // Sunset Orange
	Error:        lipgloss.Color("#E63946"), // Red
	Muted:        lipgloss.Color("#480CA8"), // Deep Violet Muted
	Background:   lipgloss.Color("#10002B"), // Midnight Purple
	Surface:      lipgloss.Color("#10002B"), // Consistent Background Fill
	Text:         lipgloss.Color("#F8F9FA"), // Off White
	Subtext:      lipgloss.Color("#B8C0C2"), // Light Gray
	Highlight:    lipgloss.Color("#7209B7"), // Violet Highlight
	GraphIngress: lipgloss.Color("#4CC9F0"), // Blue
	GraphEgress:  lipgloss.Color("#F72585"), // Magenta
}

var Matrix = Palette{
	ID:           MatrixTheme,
	Name:         "MATRIX CODE",
	Primary:      lipgloss.Color("#00FF41"), // Bright Green
	Secondary:    lipgloss.Color("#008F11"), // Emerald
	Success:      lipgloss.Color("#00FF66"), // Pure Green
	Warning:      lipgloss.Color("#ADFF2F"), // Green Yellow
	Error:        lipgloss.Color("#FF3333"), // Red
	Muted:        lipgloss.Color("#003B00"), // Dark Phosphor
	Background:   lipgloss.Color("#050B05"), // Dark Matrix Void
	Surface:      lipgloss.Color("#050B05"), // Consistent Background Fill
	Text:         lipgloss.Color("#D0FFD0"), // Phosphor White Green
	Subtext:      lipgloss.Color("#4A804A"), // Dim Green
	Highlight:    lipgloss.Color("#005500"), // Green Box
	GraphIngress: lipgloss.Color("#00FF41"), // Bright Green
	GraphEgress:  lipgloss.Color("#39FF14"), // Neon Green
}

var Monokai = Palette{
	ID:           MonokaiTheme,
	Name:         "MONOKAI PRO",
	Primary:      lipgloss.Color("#FC9867"), // Vibrant Amber
	Secondary:    lipgloss.Color("#FF6188"), // Pink Red
	Success:      lipgloss.Color("#A9DC76"), // Mint Green
	Warning:      lipgloss.Color("#FFD866"), // Warm Yellow
	Error:        lipgloss.Color("#FF6188"), // Coral Red
	Muted:        lipgloss.Color("#5B595C"), // Medium Charcoal
	Background:   lipgloss.Color("#19181A"), // Dark Charcoal
	Surface:      lipgloss.Color("#19181A"), // Consistent Background Fill
	Text:         lipgloss.Color("#FCFCFA"), // Warm White
	Subtext:      lipgloss.Color("#939293"), // Neutral Gray
	Highlight:    lipgloss.Color("#AB9DF2"), // Purple Highlight
	GraphIngress: lipgloss.Color("#78DCE8"), // Cyan
	GraphEgress:  lipgloss.Color("#FF6188"), // Pink
}

var Cobalt2 = Palette{
	ID:           Cobalt2Theme,
	Name:         "COBALT2",
	Primary:      lipgloss.Color("#1493FF"), // Cobalt Blue
	Secondary:    lipgloss.Color("#FFC600"), // Bright Gold Yellow
	Success:      lipgloss.Color("#3AD900"), // Vibrant Green
	Warning:      lipgloss.Color("#FF9D00"), // Warm Orange
	Error:        lipgloss.Color("#FF2C70"), // Cobalt Pink Red
	Muted:        lipgloss.Color("#004B87"), // Deep Cobalt Blue Muted
	Background:   lipgloss.Color("#193549"), // Cobalt2 Navy Background
	Surface:      lipgloss.Color("#193549"), // Consistent Background Fill
	Text:         lipgloss.Color("#FFFFFF"), // Pure White
	Subtext:      lipgloss.Color("#9EBFD6"), // Soft Blue Gray
	Highlight:    lipgloss.Color("#005082"), // Cobalt Blue Highlight
	GraphIngress: lipgloss.Color("#1493FF"), // Blue
	GraphEgress:  lipgloss.Color("#FFC600"), // Gold
}

var SolarizedDark = Palette{
	ID:           SolarizedDarkTheme,
	Name:         "SOLARIZED DARK",
	Primary:      lipgloss.Color("#268BD2"), // Solarized Blue
	Secondary:    lipgloss.Color("#D33682"), // Solarized Magenta
	Success:      lipgloss.Color("#859900"), // Solarized Green
	Warning:      lipgloss.Color("#B58900"), // Solarized Yellow
	Error:        lipgloss.Color("#DC322F"), // Solarized Red
	Muted:        lipgloss.Color("#586E75"), // Base01 Muted
	Background:   lipgloss.Color("#002B36"), // Base03 Deep Dark Blue
	Surface:      lipgloss.Color("#002B36"), // Consistent Background Fill
	Text:         lipgloss.Color("#839496"), // Base0 Text
	Subtext:      lipgloss.Color("#657B83"), // Base00 Subtext
	Highlight:    lipgloss.Color("#2AA198"), // Solarized Cyan Highlight
	GraphIngress: lipgloss.Color("#268BD2"), // Solarized Blue
	GraphEgress:  lipgloss.Color("#D33682"), // Solarized Magenta
}

var AllThemes = []Palette{Cyberpunk, Synthwave, Matrix, Monokai, Cobalt2, SolarizedDark}

func GetNextTheme(current ThemeID) Palette {
	for i, t := range AllThemes {
		if t.ID == current {
			nextIdx := (i + 1) % len(AllThemes)
			return AllThemes[nextIdx]
		}
	}
	return Cyberpunk
}
