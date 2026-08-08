# Dynamic State Database Ingestion & Zero Hostname Conditionals

## Rules for Telemetry Ingestion & Protocol State Evaluation

1. **Never Use Hostname or Node-Role Conditionals for Protocol Logic**:
   - Never use hostnames, node role names (e.g. `spine`, `leaf`), or static IP addresses as conditionals to infer protocol behavior, route status, or forwarding state.
   - Switches in modern topologies (Border Spines, Leaf/Spines, DCI gateways) can import routes regardless of role name.

2. **Dynamic Forwarding & State Table Verification**:
   - Protocol attributes (such as EVPN FIB installation `u*>` vs BGP-RIB only `r*`) must **always** be evaluated dynamically by matching route payloads against actual local forwarding tables (`macMap`, `routeMap`, `arpMap`, `bridge-table`).

3. **Zero Dummy Fallbacks or Static IP Shortcuts**:
   - All telemetry data (including LLDP system descriptions, VTEP IPs, BGP peer states, and MAC entries) must be derived directly from authentic gNMI state database notifications without using dummy fallback strings or hardcoded IP arrays.
