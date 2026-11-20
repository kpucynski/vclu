package lua

import (
	"strings"
	"testing"
)

// A GATE Modbus's objects as the OM import generates their proxy
// (omparser.GenerateProxy; TestGenerateProxyGateModbus there checks the same lines).
const gateModbusProxy = `
CLU511000001 = CLU_REMOTE:new(511000001, 0xC0A8010A, system:getEventBus())
local _clu = CLU511000001
modbus = {}
modbus.classic_reg = RemoteGateModbus:new(_clu, "MBS0001", "classic_reg")
modbus.tank_level = RemoteGateModbusValue:new(_clu, "MBV0001", "tank_level")
modbus.rtu_temp = RemoteGateModbusRTU:new(_clu, "MRT0001", "rtu_temp")
registerObject("CLU511000001.MBS0001", modbus.classic_reg)
registerObject("CLU511000001.MBV0001", modbus.tank_level)
registerObject("CLU511000001.MRT0001", modbus.rtu_temp)
`

func gateModbusEngine(t *testing.T) (*LuaEngine, *sentCommands, *askedCommands) {
	t.Helper()
	engine := NewLuaEngine()
	t.Cleanup(engine.Close)
	sent, asked := &sentCommands{}, &askedCommands{}
	engine.SetCommandNoReplySender(sent.send)
	engine.SetCommandSender(asked.send)
	mustExec(t, engine, gateModbusProxy)
	return engine, sent, asked
}

func TestGateModbusRegistry(t *testing.T) {
	engine, _, _ := gateModbusEngine(t)
	for path, want := range map[string]string{
		"MBS0001": "classic_reg GATE_MODBUS",
		"MBV0001": "tank_level GATE_MODBUS_VALUE",
		"MRT0001": "rtu_temp GATE_MODBUS_RTU",
	} {
		got := mustExec(t, engine, `local o = _:get("CLU511000001.`+path+`") return o._name .. " " .. o._type`)
		if got != want {
			t.Errorf("%s: %q, want %q", path, got, want)
		}
	}
}

// Sync asks for the Value feature index (not 0).
func TestGateModbusSubscribeFeatures(t *testing.T) {
	engine, _, _ := gateModbusEngine(t)
	got := mustExec(t, engine, `
		local function join(t) return table.concat(t, ",") end
		return join(modbus.classic_reg:subscribeFeatures()) .. ";"
			.. join(modbus.tank_level:subscribeFeatures()) .. ";"
			.. join(modbus.rtu_temp:subscribeFeatures())`)
	if got != "14;20;22" {
		t.Errorf("subscribeFeatures: %q, want 14;20;22", got)
	}
}

// Values follow the GATE's reports (the remote subscription).
func TestGateModbusValues(t *testing.T) {
	engine, _, _ := gateModbusEngine(t)
	mustExec(t, engine, `
		modbus.classic_reg:updateState(modbus.classic_reg:fromFeature(14, "12.5"))
		modbus.tank_level:updateState(modbus.tank_level:fromFeature(20, "42"))
		modbus.rtu_temp:updateState(modbus.rtu_temp:fromFeature(22, "21.0"))
	`)
	got := mustExec(t, engine, `
		return tostring(modbus.classic_reg:getValue()) .. ";"
			.. tostring(modbus.tank_level:getValue()) .. ";"
			.. tostring(modbus.rtu_temp:getValue())`)
	if got != "12.5;42;21" {
		t.Errorf("values: %q", got)
	}
}

// Writes go to the right feature / WriteValue; settings features are refused.
func TestGateModbusCommands(t *testing.T) {
	engine, sent, _ := gateModbusEngine(t)
	for stmt, want := range map[string]string{
		`modbus.classic_reg:setValue(7)`:  "MBS0001:set(14, 7)",
		`modbus.classic_reg:set(0, 8)`:    "MBS0001:set(14, 8)",
		`modbus.tank_level:writeValue(9)`: "MBV0001:execute(1, 9)",
		`modbus.tank_level:setValue(10)`:  "MBV0001:execute(1, 10)",
		`modbus.tank_level:set(0, 11)`:    "MBV0001:execute(1, 11)",
		`modbus.tank_level:readValue()`:   "MBV0001:execute(0)",
		`modbus.rtu_temp:writeValue(12)`:  "MRT0001:execute(1, 12)",
		`modbus.rtu_temp:set(0, 13)`:      "MRT0001:execute(1, 13)",
		`modbus.rtu_temp:readValue()`:     "MRT0001:execute(0)",
	} {
		sent.take()
		if err := try(t, engine, stmt); err != "ok" {
			t.Errorf("%s: %s", stmt, err)
			continue
		}
		if got := sent.take(); got != want {
			t.Errorf("%s: sent %q, want %q", stmt, got, want)
		}
	}
}

func TestGateModbusSettingsRefused(t *testing.T) {
	engine, sent, _ := gateModbusEngine(t)
	for _, stmt := range []string{
		`modbus.classic_reg:set(2, 100)`,
		`modbus.tank_level:set(5, 1)`,
		`modbus.rtu_temp:set(22, 1)`,
		`modbus.tank_level:execute(2)`,
		`modbus.classic_reg:execute(0)`,
	} {
		sent.take()
		if err := try(t, engine, stmt); err == "ok" {
			t.Errorf("%s: allowed, sent %q", stmt, sent.take())
		}
	}
}

// Expose type is number (HA / Sync).
func TestGateModbusExposeType(t *testing.T) {
	engine, _, _ := gateModbusEngine(t)
	got := mustExec(t, engine, `
		local TypeMap = require("TypeMap")
		return TypeMap.getExposeType(modbus.classic_reg._type) .. ";"
			.. TypeMap.getExposeType(modbus.tank_level._type) .. ";"
			.. TypeMap.getExposeType(modbus.rtu_temp._type)`)
	if got != "number;number;number" {
		t.Errorf("expose type: %q", got)
	}
}

func TestGateModbusOptimisticState(t *testing.T) {
	engine, sent, _ := gateModbusEngine(t)
	mustExec(t, engine, `modbus.tank_level:setValue(55)`)
	_ = sent.take()
	got := mustExec(t, engine, `return tostring(modbus.tank_level:getValue())`)
	if got != "55" {
		t.Errorf("optimistic state: %q", got)
	}
	if strings.Contains(got, "nil") {
		t.Fatal(got)
	}
}
