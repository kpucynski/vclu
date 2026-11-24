### Home Assistant

- **Contract change: GATE Alarm zones and sections are an `alarm_control_panel`, not a switch** - `expose()` on a Grenton SatelZone (partition) or JablotronSection publishes Home Assistant's alarm panel (`armed_away` / `disarmed`, `ARM_AWAY` / `ARM_HOME` / `DISARM`). A retained switch discovery from an earlier version is cleared. Satel inputs and Jablotron devices get `device_class: safety` so they appear under Security. HomeKit still shows the zone as a read-only switch. Scripts that checked `h.type == "switch"` on an exposed zone must use `"alarm"`.
