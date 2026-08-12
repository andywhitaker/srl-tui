# Lipgloss TUI Background & Border Rendering Guidelines

## 1. Always Set `BorderBackground` on Bordered Styles
When defining bordered containers or cards with `lipgloss.NewStyle().Border(...)`:
- You MUST set `.BorderBackground(bg)` alongside `.Background(bg)`.
- **Reason**: Lipgloss's `.Background(bg)` only colors internal content. Without `.BorderBackground(bg)`, border outline characters (`╭`, `─`, `╮`, `│`, `╰`, `╯`, `╔`, `═`, `╗`, `║`, `╚`, `╝`) render without background color codes, allowing raw terminal default background colors to bleed through underneath borders.

```go
// CORRECT
boxStyle := lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(pal.Primary).
	BorderBackground(pal.Background). // <--- Required to color border characters
	Background(pal.Background).
	Padding(0, 1)
```

## 2. Multi-Line Block Padding & Gap Alignment
When joining multi-line blocks horizontally (`lipgloss.JoinHorizontal`) or vertically (`lipgloss.JoinVertical`):
- Gap strings between multi-line blocks must be multi-line strings matching the exact height of adjacent blocks.
- Wrap joined output line-by-line with `lipgloss.NewStyle().Background(bg).Render(line)` to ensure right-margin padding inserted by `JoinVertical` inherits the expected background color.

```go
// Multi-line gap string matching box height (4 lines)
gapLine := lipgloss.NewStyle().Background(pal.Background).Render("   ")
gapStr := fmt.Sprintf("%s\n%s\n%s\n%s", gapLine, gapLine, gapLine, gapLine)

// Line-by-line re-wrapping after JoinVertical
var styledLines []string
bgStyle := lipgloss.NewStyle().Background(pal.Background)
for _, line := range strings.Split(joinedDiagram, "\n") {
	styledLines = append(styledLines, bgStyle.Render(line))
}
```

## 3. Container Deployment & Stale Process Memory
When deploying updated TUI binaries to Docker containers (`docker cp`):
- Always execute `docker exec <container> pkill -9 -f <binary_name> || true` before or after copying binaries.
- **Reason**: Linux process memory retains the previously executed binary in RAM for active terminal sessions (`pts/*`) until the process is explicitly killed and re-executed.

## 4. Prevent Unstyled Background Gaps in Detail Views & Modals
When formatting key-value lines or padded text lines inside modal boxes:
- DO NOT use unstyled `fmt.Sprintf` format specifiers like `fmt.Sprintf("  %-20s : %s", keyStyle.Render(k), valStyle.Render(v))`. Unstyled spaces and colons outside Lipgloss style calls emit no ANSI background code.
- Explicitly set `.Background(pal.Background)` on EVERY label, value, colon, and space segment.
- Replace ANSI reset sequences `\x1b[0m` inside lines with `\x1b[0m\x1b[48;2;<R>;<G>;<B>m` so trailing `\x1b[0m` from styled values does not reset background color on right-side padding spaces.
- Pad lines to `contentWidth` prior to line wrapping so Lipgloss `Width()` does not insert unstyled space padding inside borders.

```go
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
```

## 5. Real-Time Telemetry Counters & Dynamic Ticking
When managing system uptime, BGP neighbor uptime, or CPU/RAM utilization metrics:
- **Continuous Ticking**: Counters MUST tick dynamically in real-time between telemetry events using background tickers or render-time duration calculations (`time.Since(s.StartTime)` for system uptime, `time.Since(p.LastEstablished)` for BGP neighbor uptime).
- **State Priority**: Incoming gNMI telemetry updates MUST take priority by immediately updating `StartTime`, `LastEstablished` timestamps, or platform CPU/RAM utilization values whenever received from the device.
- **Local System Metric Fallback**: CPU and RAM utilization metrics should periodically sample `/proc/stat` and `/proc/meminfo` (or `/sys`) inside the container to maintain live visual responsiveness when gNMI event streams are quiet.
