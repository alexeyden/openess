# OpenESS

A service that exports power consumption and other metrics from Chinese solar inverters over MQTT. It can be used as a replacement for proprietary [SmartESS mobile app](https://play.google.com/store/apps/details?id=com.eybond.smartclient.ess).

In theory it can work with most inverters supported by SmartESS, but some inverters will probably require minor improvements/fixes in code -- see limitations.

## Supported inverters

Tested and working:
- PowMr VM PLUS 5.5KW ([Aliexpress link](https://aliexpress.ru/item/1005004211405506.html?spm=a2g2w.orderdetail.0.0.147d4aa6YGHm9J))

Unsupported:
- Anenji 4kw/7.2kw (see [this issue](https://github.com/alexeyden/openess/issues/2)). If you happen to have one of these models, check out this repo by sabatex: [NetDaemonApps.InverterAnenji-4kw-7.2kw](https://github.com/sabatex/NetDaemonApps.InverterAnenji-4kw-7.2kw).

Please let me know via issues if your inverter happen to work too (and more so if it doesn't), so I can update the list here.

## Features, limitations and known issues

The service periodically polls specified Modbus registers, interprets their values based on register space descriptors pulled from SmartESS and exports interpreted human-readable values over MQTT (e.g. to Home Assistant). In addition, it can configure the datalogger (SSID and password) and the inverter itself via CLI tool which is bundled into the service.

Register values are exported at `openess/registers/{name}` topics. Additionally, the datalogger connection status is exported at `openess/status` (`online`/`offline`).

Currently only WiFi dataloggers are supported (no BLE/serial). I've only tested it with a thing called `Wi-Fi Plug Pro` ([Aliexpress link](https://aliexpress.ru/item/4000102754817.html?sku_id=12000027644368209&spm=a2g2w.productlist.search_results.0.3d667fd2ZBrSSr)) that came with my inverter, but others will probably work too.

The service depends on original register space descriptor files pulled from SmartESS APK, see `xxxx.json` files in `data/`. The appropriate file is selected based on protocol string reported by datalogger during connection process.

Some other techinal limitations:

- Registers with `valueType` other than `1` are not supported. It is not that hard to add other types, but we have to have an inverter for testing that supports them.
- Some parts of descriptor files are not currently used, e.g. `OtherCodes` sections not related to enumerations.

Known issues:
- Automation protocol detection may not work correctly. You can override the protocol file via the `Protocol` field in the config.

## Installation and configuration 

Installation:

1. Build the service with `make dist ARCH=arm64` (omit `ARCH=arm64` to build for host arch)
2. Copy `openess.tar.xz` to target machine and extract it there.
3. Install the service with `cd data/ && sudo ./install.sh`. You can uninstall later it with `sudo ./install.sh uninstall` if needed.
5. Run the service in CLI mode to check if it can properly connect to the datalogger: `openess -d 192.168.1.37:58899`. Note what descriptor file is used (look for `client: loaded protocol descriptor: 0925.json` log entry). You can also change datalogger SSID/password here (see CLI description below). 
4. Edit the config file at `/etc/openess/config.json`. You must specify datalogger address (`DeviceAddr`), MQTT broker address (`Export.Broker`) and a set of exported registers for your inverter (`Collector.Registers`). You can lookup register names in your `xxxx.json` file found out at previous step.
5. Enable and start the service: `sudo systemctl enable openess --now`

Configuration file:

```jsonc
{
    "BindPort": 8899,                   // local TCP port used for datalogger connection
    "DeviceAddr": "192.168.1.37:58899", // datalogger address
    "ProtoPath": "data/",               // a path to descriptor files (xxxx.json)
    "Protocol": "0925",                 // overrides automatic protocol detection (optional field)
    "Export": {  
        // MQTT export config
        "Broker": "tcp://127.0.0.1:1883", // broker address (required)
        "ClientId": "MyExporter",         // client id (optional)
        "User": "user",                   // auth creds (optional)
        "Password": "password"            // auth creds (optional)
    },
    "Collector": {
        "Interval": "500ms", // polling interval
        "Enabled": true,     // enable polling
        "Registers": {
            // A list of registers to poll.
            // Keys are MQTT register topic names: openess/registers/{name}
            // Values are register names from descriptor file
            "working_state": "Working State",
            "battery_capacity": "Battery capacity %",
            "battery_voltage": "Battery voltage",
            "charging_current": "Charging current",
            "discharge_current": "Battery discharge current",
            "pv_voltage": "PV input voltage",
            "pv_power": "PV input power",
            "ac_input_voltage": "AC input voltage",
            "output_voltage": "Output voltage",
            "output_power": "Output apparent power ",
            "output_active_power": "Output active power",
            "output_power_percent": "AC output Load %"
        }
    }
}
```

## Using the CLI mode

Ensure that service is stopped before running the CLI. Type `help` to get a list of supported commands. Type `exit` or `^D` to exit.

Setting datalogger SSID/password:
```
$ openess -d 192.168.1.37:58899
INFO    2024/01/24 13:52:04 client: connecting to device 192.168.1.37:58899
INFO    2024/01/24 13:52:04 client: connected to datalogger: manufacturer 37 device type 8 (protocol v1.2 props 0925,5,5,#0#)
% set-param ssid HomeWifi
% set-param password MyPassword
% set-param restart 1
% exit
```

Reading registers:
```
% read-named  "Working State"
Line Mode
% read-named "Output apparent power "
4287VA
```

Writing registers:
```
% write-named "Remove all power history" 1
010c
% write-named "LCD backlight" 0
001d
```

When writing registers, enumeration variants are represented by numeric values, so you have to look up proper values in the `xxxx.json` descrptor file.

## Integration with Home Assistant

Complete configuration example for PowMr VM PLUS 5.5KW and similar inverters.

**Note:** These inverters fail to read cumulative energy registers for some reqson (`today_energy`, `all_energy`, or `month_energy`). Use the integration sensors below to calculate energy from power measurements instead.

```yaml
mqtt:
  sensor:
    # Working State (enum: Power On / Stand By / Invert Mode / Line Mode / Bypass / Fault)
    - name: "Inverter Working State"
      unique_id: inverter_working_state
      state_topic: "openess/register/working_state"
      availability_topic: "openess/status"
      icon: mdi:state-machine

    # Battery
    - name: "Inverter Battery"
      unique_id: inverter_battery_capacity
      state_topic: "openess/register/battery_capacity"
      availability_topic: "openess/status"
      unit_of_measurement: "%"
      device_class: battery
      state_class: measurement
      value_template: "{{ value | int }}"

    - name: "Inverter Battery Voltage"
      unique_id: inverter_battery_voltage
      state_topic: "openess/register/battery_voltage"
      availability_topic: "openess/status"
      unit_of_measurement: "V"
      device_class: voltage
      state_class: measurement
      value_template: "{{ value | float | round(1) }}"

    - name: "Inverter Battery Charging Current"
      unique_id: inverter_charging_current
      state_topic: "openess/register/charging_current"
      availability_topic: "openess/status"
      unit_of_measurement: "A"
      device_class: current
      state_class: measurement
      value_template: "{{ value | float | round(1) }}"

    - name: "Inverter Battery Discharge Current"
      unique_id: inverter_discharge_current
      state_topic: "openess/register/discharge_current"
      availability_topic: "openess/status"
      unit_of_measurement: "A"
      device_class: current
      state_class: measurement
      value_template: "{{ value | float | round(1) }}"

    # Solar (PV)
    - name: "Inverter PV Voltage"
      unique_id: inverter_pv_voltage
      state_topic: "openess/register/pv_voltage"
      availability_topic: "openess/status"
      unit_of_measurement: "V"
      device_class: voltage
      state_class: measurement
      value_template: "{{ value | float | round(1) }}"

    - name: "Inverter PV Power"
      unique_id: inverter_pv_power
      state_topic: "openess/register/pv_power"
      availability_topic: "openess/status"
      unit_of_measurement: "W"
      device_class: power
      state_class: measurement
      value_template: "{{ value | float | round(0) }}"

    # Grid (AC Input)
    - name: "Inverter AC Input Voltage"
      unique_id: inverter_ac_input_voltage
      state_topic: "openess/register/ac_input_voltage"
      availability_topic: "openess/status"
      unit_of_measurement: "V"
      device_class: voltage
      state_class: measurement
      value_template: "{{ value | float | round(1) }}"

    # Output (Load)
    - name: "Inverter Output Voltage"
      unique_id: inverter_output_voltage
      state_topic: "openess/register/output_voltage"
      availability_topic: "openess/status"
      unit_of_measurement: "V"
      device_class: voltage
      state_class: measurement
      value_template: "{{ value | float | round(1) }}"

    - name: "Inverter Output Apparent Power"
      unique_id: inverter_output_apparent_power
      state_topic: "openess/register/output_power"
      availability_topic: "openess/status"
      unit_of_measurement: "VA"
      device_class: apparent_power
      state_class: measurement
      value_template: "{{ value | float | round(0) }}"

    - name: "Inverter Output Active Power"
      unique_id: inverter_output_active_power
      state_topic: "openess/register/output_active_power"
      availability_topic: "openess/status"
      unit_of_measurement: "W"
      device_class: power
      state_class: measurement
      value_template: "{{ value | float | round(0) }}"

    - name: "Inverter Output Load"
      unique_id: inverter_output_load_pct
      state_topic: "openess/register/output_power_percent"
      availability_topic: "openess/status"
      unit_of_measurement: "%"
      state_class: measurement
      value_template: "{{ value | float | round(0) }}"

template:
  - sensor:
      # Grid power: positive when consuming from grid (Line Mode)
      - name: "Inverter Grid Power"
        unique_id: inverter_grid_power
        unit_of_measurement: "W"
        device_class: power
        state_class: measurement
        state: >
          {% if states('sensor.inverter_working_state') in ['Line Mode', 'Bypass'] %}
            {{ states('sensor.inverter_output_power') | float(0) }}
          {% else %}
            0
          {% endif %}

      # Battery discharge power
      - name: "Inverter Battery Discharge Power"
        unique_id: inverter_battery_discharge_power
        unit_of_measurement: "W"
        device_class: power
        state_class: measurement
        state: >
          {{ (states('sensor.inverter_battery_discharge_current') | float(0))
             * (states('sensor.inverter_battery_voltage') | float(0)) }}

      # Battery charge power
      - name: "Inverter Battery Charge Power"
        unique_id: inverter_battery_charge_power
        unit_of_measurement: "W"
        device_class: power
        state_class: measurement
        state: >
          {{ (states('sensor.inverter_battery_charging_current') | float(0))
             * (states('sensor.inverter_battery_voltage') | float(0)) }}

sensor:
  # Convert power (W) to energy (kWh) via integration
  # These sensors calculate energy totals from power measurements
  - platform: integration
    source: sensor.inverter_pv_power
    name: "Solar Energy"
    unique_id: inverter_solar_energy
    unit_prefix: k
    round: 2
    method: left

  - platform: integration
    source: sensor.inverter_grid_power
    name: "Grid Energy Consumed"
    unique_id: inverter_grid_energy
    unit_prefix: k
    round: 2
    method: left

  - platform: integration
    source: sensor.inverter_battery_discharge_power
    name: "Battery Energy Discharged"
    unique_id: inverter_battery_energy_discharged
    unit_prefix: k
    round: 2
    method: left

  - platform: integration
    source: sensor.inverter_battery_charge_power
    name: "Battery Energy Charged"
    unique_id: inverter_battery_energy_charged
    unit_prefix: k
    round: 2
    method: left
```

This configuration provides:
- Real-time monitoring of battery, solar (PV), grid, and output metrics
- Calculated energy sensors that integrate power over time for use in Home Assistant's Energy Dashboard
- Template sensors for derived values (grid power, battery charge/discharge power)

To use this configuration, ensure your `config.json` includes the required registers as shown in the Configuration section above.

