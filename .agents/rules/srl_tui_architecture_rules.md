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

## 4. BGP Active Address Family Operational Filtering
- Address families (`AddrFamilies`) MUST reflect operational state (`oper-state == "up"` / `admin-state != "disable"`).
- Address families reporting `oper-state: down` or `admin-state: disable` (e.g., `ipv6-unicast`, `route-target`) MUST be purged from `AddrFamilies` and `AFStats`.
- Negotiated OPEN capabilities (`received-afi-safi`) MUST NOT override operational down states.

## 5. Per-Family Route Counter Ingestion & Display
- BGP peer state MUST maintain per-family route statistics (`AFStats map[string]AFStats`).
- Main BGP tables and inspector modals MUST display accurate per-family sent/received route counts (e.g. `4/2, 39/13`), avoiding global prefix total overwrites on family rows.

## 6. Dynamic BGP Peer Interface Resolution
- Local BGP interfaces MUST be resolved dynamically from gNMI `local-interface` attributes, neighbor `description` fields (`via intf <name>`), or local IP route table matching.
- Static interface string fallbacks are strictly prohibited.

