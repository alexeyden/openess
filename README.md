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

Example configuration:

```yaml
mqtt:
  sensor:
    - name: "powmr_output_power"
      state_topic: "openess/register/output_active_power"
      availability_topic: "openess/status"
      value_template: "{{value | float | round(2)}}"
      unit_of_measurement: "W"
      device_class: power
      state_class: measurement
    - name: "powmr_output_power_percent"
      availability_topic: "openess/status"
      state_topic: "openess/register/output_power_percent"
      value_template: "{{value | float | round(2)}}"
      unit_of_measurement: "%"
      device_class: power_factor
      state_class: measurement

# You have to integrate values of output power sensor over time,
# either via `inegration` or `utility_meter` to get consumption in KWh,
# otherwise it wouldnt show up in the Energy tab

sensor:
  - platform: integration
    source: sensor.powmr_output_power
    name: energy_spent
    unit_prefix: k
    round: 2
```

