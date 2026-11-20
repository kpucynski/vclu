package omparser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// The om.lua of a GATE Modbus, lines as Object Manager writes them (anonymised).
var gateModbusOM = `-- FwType 00000001
-- FwVersion 00000456
-- HwType 00000012
-- HwVersion 00000002
CLU511000001 = GATE:new(5000, 0xC0A8010A, 0xFFFFFF00, 0xC0A80101, "", "")
-- NAME_CLU ModbusGate=CLU511000001
MBS0001 = GATE:new(200, "MBS0001")
-- NAME_PERIPHERY classic_reg=MBS0001
MBV0001 = GATE:new(201, "MBV0001")
-- NAME_PERIPHERY tank_level=MBV0001
MRT0001 = GATE:new(203, "MRT0001")
-- NAME_PERIPHERY rtu_temp=MRT0001
CFG0001 = GATE:new(202, "CFG0001")
-- NAME_PERIPHERY modbus_config=CFG0001
TIM0001 = GATE:new(6, "TIM0001")
-- NAME_PERIPHERY timer=TIM0001
MBS0001:set(0, 1)
MBS0001:set(2, 100)
MBS0001:set(14, 0)
MBV0001:set(5, 2)
MBV0001:set(8, 40001)
MBV0001:set(16, 16)
MRT0001:set(5, 3)
MRT0001:set(8, 0)
CFG0001:set(0, 19200)
TIM0001:set(0, 1000)
CLU511000001:add_event(0, OnInit)
`

var gateModbusObjects = map[string]struct {
	class ObjectClass
	name  string
}{
	"MBS0001": {ClassGateModbus, "classic_reg"},
	"MBV0001": {ClassGateModbusValue, "tank_level"},
	"MRT0001": {ClassGateModbusRTU, "rtu_temp"},
}

// A GATE Modbus's register objects are read with their names; config and timers are left out.
func TestParseGateModbus(t *testing.T) {
	parsed, err := Parse(gateModbusOM)
	if err != nil {
		t.Fatal(err)
	}
	if !parsed.CLU.IsGate || !parsed.CLU.IsGateModbus() || parsed.CLU.IsGateAlarm() || len(parsed.Warnings) != 0 {
		t.Errorf("CLU %+v warnings %q", parsed.CLU, parsed.Warnings)
	}
	if len(parsed.Objects) != len(gateModbusObjects) {
		t.Errorf("objects %+v", parsed.Objects)
	}
	for id, want := range gateModbusObjects {
		got, ok := parsed.ObjectMap[id]
		if !ok || got.Class != want.class || got.Name != want.name {
			t.Errorf("%s: %+v, want class %d name %s", id, got, want.class, want.name)
		}
		if !got.Class.IsExposable() || got.Class.MQTTType() != "number" {
			t.Errorf("%s: not exposable as number", id)
		}
	}
	if _, ok := parsed.ObjectMap["CFG0001"]; ok {
		t.Error("ModbusConfig is read; out of scope")
	}
	if _, ok := parsed.ObjectMap["TIM0001"]; ok {
		t.Error("the GATE's timer is read; vCLU has no proxy for it")
	}
	js, _ := json.Marshal(parsed)
	if strings.Contains(string(js), "40001") || strings.Contains(string(js), "19200") {
		t.Errorf("a setting reached the parsed result: %s", js)
	}
}

// The same classes on another GATE or CLU are left out with a warning.
func TestParseGateModbusClassesElsewhere(t *testing.T) {
	for name, header := range map[string]string{
		"GATE Alarm":             "-- FwType 00000002\n-- HwType 00000012\nCLU510000009 = GATE:new(5000, 0xC0A80109, 0x0, 0x0, \"\", \"\")\n",
		"HTTP GATE":              "-- FwType 00000003\n-- HwType 00000012\nCLU510000009 = GATE:new(5000, 0xC0A80109, 0x0, 0x0, \"\", \"\")\n",
		"no header":              "CLU511000009 = GATE:new(5000, 0xC0A80109, 0x0, 0x0, \"\", \"\")\n",
		"leading zeros optional": "",
	} {
		t.Run(name, func(t *testing.T) {
			om := header + "MBV0001 = GATE:new(201, \"MBV0001\")\nREQ0001 = GATE:new(121, \"REQ0001\")\nMRT0001 = GATE:new(203, \"MRT0001\")\n"
			if header == "" {
				om = "-- FwType 1\n-- HwType 12\nCLU511000009 = GATE:new(5000, 0xC0A80109, 0x0, 0x0, \"\", \"\")\n" + om
			}
			parsed, err := Parse(om)
			if err != nil {
				t.Fatal(err)
			}
			if header == "" {
				if len(parsed.Objects) != 2 || len(parsed.Warnings) != 0 || !parsed.CLU.IsGateModbus() {
					t.Errorf("objects %+v warnings %q CLU %+v", parsed.Objects, parsed.Warnings, parsed.CLU)
				}
				return
			}
			if len(parsed.Objects) != 0 {
				t.Errorf("objects %+v", parsed.Objects)
			}
			if len(parsed.Warnings) != 2 || !strings.HasPrefix(parsed.Warnings[0], "MBV0001: class 201") ||
				!strings.HasPrefix(parsed.Warnings[1], "MRT0001: class 203") {
				t.Errorf("warnings %q", parsed.Warnings)
			}
		})
	}
}

// The proxies: one per register with its name, registered under <CLU ID>.<object ID>.
func TestGenerateProxyGateModbus(t *testing.T) {
	parsed, err := Parse(gateModbusOM)
	if err != nil {
		t.Fatal(err)
	}
	out := GenerateProxy(parsed, DefaultConfig())
	for _, want := range []string{
		`modbusgate.classic_reg = RemoteGateModbus:new(_clu, "MBS0001", "classic_reg")`,
		`modbusgate.tank_level = RemoteGateModbusValue:new(_clu, "MBV0001", "tank_level")`,
		`modbusgate.rtu_temp = RemoteGateModbusRTU:new(_clu, "MRT0001", "rtu_temp")`,
		`CLU511000001._omFirmware = { type = "00000001", version = "00000456" }`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing line: %s", want)
		}
	}
	for id, o := range gateModbusObjects {
		if want := fmt.Sprintf("registerObject(\"CLU511000001.%s\", modbusgate.%s)", id, o.name); !strings.Contains(out, want) {
			t.Errorf("missing %s", want)
		}
	}
	if regexp.MustCompile(`40001|19200|CFG0001|TIM0001`).MatchString(out) {
		t.Errorf("a setting or out-of-scope object reached the proxy:\n%s", out)
	}
}
