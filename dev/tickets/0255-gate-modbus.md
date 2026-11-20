# #255: Grenton GATE Modbus register objects in vclu

- **Type:** feature
- **Area:** grenton
- **Size:** M
- **Epic:** —
- **Blocked by:** —
- **Skill:** —
- **Needs hardware:** Grenton GATE Modbus with ModbusValue / ModbusRTU objects

> State (who, status) lives in the GitHub issue, not here. To take the ticket, ask in the issue;
> a maintainer assigns it. This file is the specification: change it by PR.

## Goal
After an Object Manager import, a GATE Modbus's register objects (Modbus, ModbusValue, ModbusRTU)
are in vclu under `<CLU>.<object>`, on the Sync list, with Value following the GATE; writes go
through WriteValue / SetValue. Settings features are never written; `:set` lines stay out of the
stored om.lua (already covered by StripGateSettings).

## Context
- GATE Modbus: `FwType 00000001`, `HwType 00000012`. Same HwType as GATE Alarm/HTTP; FwType tells
  them apart (Alarm `02`, HTTP `03`).
- Objects in om.lua: `MBV1234 = GATE:new(201, "MBV1234")` with `-- NAME_PERIPHERY …`.
- Value is not feature 0:
  | Class | Type | Value feature | Write |
  |---|---|---|---|
  | 200 | Modbus | 14 | set 14 SetValue |
  | 201 | ModbusValue | 20 (string in OM) | execute 1 WriteValue |
  | 203 | ModbusRTU | 22 (read-only feature) | execute 1 WriteValue |
- Config/bus objects (202, 204, 205, 206) are out of scope for this ticket.
- Pattern: #214 GATE Alarm (`RemoteGateAlarm.lua`, `IsGateAlarm`).

## Files
- `pkg/omparser/` (types, parser, generator, tests)
- `pkg/lua/runtime/objects/RemoteGateModbus.lua`
- `pkg/lua/runtime/utils/TypeMap.lua`, embed/init, Sync UI quick-select
- `changelog.d/`

## Acceptance criteria
- [ ] Parse keeps classes 200/201/203 only on a GATE Modbus; other CLUs get a warning.
- [ ] Proxy registers them; Sync subscribes to the Value feature index (not 0).
- [ ] `getValue` / `setValue` / `writeValue` / `readValue` work; settings features refused.
- [ ] `go test ./pkg/omparser/ ./pkg/lua/ -run GateModbus` and `go test ./...` pass.
- [ ] Changelog entry.

## Contract impact
New registry objects; no existing contracts change.
