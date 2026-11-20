--- Objects of a Grenton GATE Modbus (FwType 00000001, HwType 00000012): Modbus register
-- values Object Manager declares in the GATE's om.lua (MBV0001 = GATE:new(201, "MBV0001")).
-- The OM import registers them under "<CLU ID>.<object ID>"; their Value follows the GATE
-- when they are on the Sync list (Value is not feature 0).
--
-- Numbers (class, features, methods) from Object Manager's device interfaces
-- (clu_GATE_MODBUS_ft00000001_*), not checked on a GATE Modbus yet:
--   200 Modbus       Value@14 (num, settable); SetValue by set(14)
--   201 ModbusValue  Value@20 (string in OM); execute 0 ReadValue, 1 WriteValue
--   203 ModbusRTU    Value@22 (num, read-only feature); execute 0 ReadValue, 1 WriteValue
-- Config/bus objects (202 ModbusConfig, 204–206) are out of scope.
--
-- Guards (RemoteObject._setGuard): settings features are never written; only Value may be
-- set where OM allows it; only ReadValue / WriteValue are executed on 201/203. Raw scripts
-- sent to the GATE (clu:execute(0, "...")) and a hand-built subscription are out of reach.
-- @module objects.RemoteGateModbus
-- @inherits RemoteObject

RemoteGateModbusBase = setmetatable({}, {__index = RemoteObject})
RemoteGateModbusBase.__index = RemoteGateModbusBase

-- Method numbers vCLU sends, by name (per class); any other is refused
RemoteGateModbusBase.METHODS = {}
-- Feature numbers that may be written (Value); every other set is refused
RemoteGateModbusBase.WRITABLE = {}

local classRules = {}
local objectRules = setmetatable({}, {__mode = "k"})

local function label(self)
    return tostring(self._type) .. " " .. tostring(self._objId)
end

local function refused(self, kind, index)
    local r = objectRules[self]
    if not r then return "not a GATE Modbus object" end
    if rawget(self, "_objId") ~= r.objId or rawget(self, "_clu") ~= r.clu then
        return label(self) .. ": the object's ID or CLU changed after it was created"
    end
    if type(index) ~= "number" or index ~= index or index < 0 or index >= 10000
       or index ~= math.floor(index) then
        return label(self) .. ": " .. tostring(kind) .. " takes a whole number 0-9999, got " .. tostring(index)
    end
    if kind == "set" then
        if not r.writable[index] then
            return label(self) .. ": feature " .. tostring(index)
                .. " is a setting of the GATE; vCLU does not write it (use setValue / writeValue)"
        end
        return nil
    end
    if kind == "execute" then
        if not r.methods[index] then
            return label(self) .. ": method " .. tostring(index) .. " is not one vCLU sends to it"
        end
    end
    return nil
end

--- Why a read, write or method of this object is refused; nil when it is not.
-- @param kind string "get", "set" or "execute"
-- @param index number The feature or method number
-- @return string|nil
function RemoteGateModbusBase:_refused(kind, index)
    return refused(self, kind, index)
end

--- The value the GATE reported (the Value feature).
-- @api public
-- @return number|string|nil
function RemoteGateModbusBase:getValue()
    return self._lastState
end

--- Features to subscribe to: only Value (not 0).
-- @return table
function RemoteGateModbusBase:subscribeFeatures()
    return {self.VALUE}
end

--- Turns a reported feature into the object's value (numbers when possible).
-- @param index number Feature index
-- @param value any Reported value
-- @return any|nil
function RemoteGateModbusBase:fromFeature(index, value)
    if index ~= self.VALUE then return nil end
    local n = tonumber(value)
    if n ~= nil then return n end
    return value
end

local function class(name, typ, methods)
    local c = setmetatable({}, {__index = RemoteGateModbusBase})
    c.__index = c
    c.METHODS = {}
    c.WRITABLE = {}
    c.new = function(self, clu, objId, objName)
        local obj = RemoteObject.new(self, clu, objId, objName)
        obj._type = typ
        obj._allowedMethods = methods
        local r = classRules[c]
        objectRules[obj] = { objId = objId, clu = clu, writable = r.writable, methods = r.methods }
        RemoteObject._setGuard(obj, refused)
        return obj
    end
    _G[name] = c
    return c
end

-- ==========================================================================
-- Modbus (class 200): Value@14, SetValue by set(14)
-- ==========================================================================

--- A classic GATE Modbus register (class 200). Value is feature 14.
RemoteGateModbus = class("RemoteGateModbus", "GATE_MODBUS",
    {"getValue", "setValue"})
RemoteGateModbus.VALUE = 14
RemoteGateModbus.WRITABLE = { [14] = "Value" }

--- Writes the register (set 14 SetValue). set(0, v) and HA/MCP setValue map here.
-- @api public
-- @param value number The value
function RemoteGateModbus:setValue(value)
    RemoteObject.set(self, self.VALUE, value)
    self:updateState(tonumber(value) or value)
end

--- set(0, v) and set(14, v) write Value; other features are refused by the guard.
-- @param index number Feature number
-- @param value any The value
function RemoteGateModbus:set(index, value)
    if index == 0 or index == self.VALUE then
        return self:setValue(value)
    end
    return RemoteObject.set(self, index, value)
end

-- ==========================================================================
-- ModbusValue (class 201): Value@20, ReadValue / WriteValue
-- ==========================================================================

--- A GATE ModbusValue register (class 201). Value is feature 20 (string in OM).
RemoteGateModbusValue = class("RemoteGateModbusValue", "GATE_MODBUS_VALUE",
    {"getValue", "setValue", "writeValue", "readValue"})
RemoteGateModbusValue.VALUE = 20
RemoteGateModbusValue.READ_VALUE = 0
RemoteGateModbusValue.WRITE_VALUE = 1
RemoteGateModbusValue.METHODS = { [0] = "ReadValue", [1] = "WriteValue" }
RemoteGateModbusValue.WRITABLE = { [20] = "Value" }

--- Asks the GATE to read the register now (execute 0 ReadValue).
-- @api public
function RemoteGateModbusValue:readValue()
    self:execute(self.READ_VALUE)
end

--- Writes the register (execute 1 WriteValue). Prefer this over set(20).
-- @api public
-- @param value number|string The value
function RemoteGateModbusValue:writeValue(value)
    self:execute(self.WRITE_VALUE, value)
    self:updateState(tonumber(value) or value)
end

--- Writes the register (WriteValue). set(0, v) and HA/MCP setValue map here.
-- @api public
-- @param value number|string The value
function RemoteGateModbusValue:setValue(value)
    self:writeValue(value)
end

--- set(0, v) writes via WriteValue; set(20, v) is allowed (OM SetValue); other features refused.
-- @param index number Feature number
-- @param value any The value
function RemoteGateModbusValue:set(index, value)
    if index == 0 then
        return self:writeValue(value)
    end
    if index == self.VALUE then
        RemoteObject.set(self, self.VALUE, value)
        self:updateState(tonumber(value) or value)
        return
    end
    return RemoteObject.set(self, index, value)
end

-- ==========================================================================
-- ModbusRTU (class 203): Value@22 read-only, ReadValue / WriteValue
-- ==========================================================================

--- A GATE ModbusRTU register (class 203). Value is feature 22 (read-only); writes use WriteValue.
RemoteGateModbusRTU = class("RemoteGateModbusRTU", "GATE_MODBUS_RTU",
    {"getValue", "setValue", "writeValue", "readValue"})
RemoteGateModbusRTU.VALUE = 22
RemoteGateModbusRTU.READ_VALUE = 0
RemoteGateModbusRTU.WRITE_VALUE = 1
RemoteGateModbusRTU.METHODS = { [0] = "ReadValue", [1] = "WriteValue" }
RemoteGateModbusRTU.WRITABLE = {}

RemoteGateModbusRTU.readValue = RemoteGateModbusValue.readValue
RemoteGateModbusRTU.writeValue = RemoteGateModbusValue.writeValue
RemoteGateModbusRTU.setValue = RemoteGateModbusValue.setValue

--- set(0, v) writes via WriteValue; every feature set is refused (Value is read-only).
-- @param index number Feature number
-- @param value any The value
function RemoteGateModbusRTU:set(index, value)
    if index == 0 then
        return self:writeValue(value)
    end
    return RemoteObject.set(self, index, value)
end

-- ==========================================================================
-- The guard's rules: a copy for the guard, read-only tables for scripts
-- ==========================================================================

local function copy(t)
    local out = {}
    for k, v in pairs(t) do out[k] = v end
    return out
end

local function readOnly(t, what)
    return setmetatable({}, {
        __index = t,
        __newindex = function() error(what .. " is read-only", 2) end,
        __metatable = false,
    })
end

for _, name in ipairs({ "RemoteGateModbus", "RemoteGateModbusValue", "RemoteGateModbusRTU" }) do
    local c = _G[name]
    local r = { writable = copy(c.WRITABLE), methods = copy(c.METHODS) }
    classRules[c] = r
    c.WRITABLE = readOnly(r.writable, name .. ".WRITABLE")
    c.METHODS = readOnly(r.methods, name .. ".METHODS")
end

return RemoteGateModbus
