# SR Linux TUI Architecture & Telemetry Invariants

## 1. gNMI Stream Subpath JSON Parsing
- When unmarshaling gNMI stream notifications:
  - Root path notifications (`/interface[name=X]`) place fields in nested sub-maps (e.g., `dataMap["traffic-rate"]["in-bps"]`).
  - Subpath notifications (`/interface[name=X]/traffic-rate`) place fields at the top level of the unmarshaled JSON map (`dataMap["in-bps"]`).
- Telemetry parsers MUST inspect both top-level keys AND nested sub-maps before defaulting or skipping values.

## 2. TUI Fixed Box Heights & Layout Stability
- Outer Lipgloss container styles (`boxStyle`) for main view components MUST explicitly specify `.Height(height - 2)`.
- When rendering tables or lists with variable row counts (e.g. filtered views), pad remaining rows with blank lines up to `visibleRowCount`.
- Prevents `lipgloss.JoinVertical` from shifting or jumping the global bottom help bar (`footerView`) across subtabs or keystrokes.

## 3. Containerlab Execution & Compilation Targets
- Binaries deployed to Containerlab nodes run inside the Linux container OS.
- Cross-compilation for switch deployment MUST target `linux/amd64` (x86_64 hosts/switches) or `linux/arm64` (Apple Silicon Mac VMs / ARM switches).
