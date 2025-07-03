# tplinky - a Go package for simple interactions with Kasa TPLink smart plugs.

## Overview

This package provides a Go API for communicating with a
[tp-link](https://www.tp-link.com/) smartplug device.

An in depth discussion of these devices is available here:
https://www.softscheck.com/en/blog/tp-link-reverse-engineering/ and a
subset of the information found there was used to develop this package
for some automation projects.

Automated documentation for this Go package is available from [![Go
Reference](https://pkg.go.dev/badge/zappem.net/pub/net/tplinky.svg)](https://pkg.go.dev/zappem.net/pub/net/tplinky).

## Example

**NOTE** The following example assumes your computer network is
  `192.168.1.0/24`. If you have configured your devices on a different
  network, you will need to substitute your network addresses instead
  as you follow along.

To use this device (on already configured devices, see [Initial Setup
section](#initial-setup-section)) you can try the following example:

```
$ git clone https://github.com/tinkerator/tplinky.git
$ cd tplinky
$ go build examples/tple.go
$ ./tple --scan=192.168.1.0/24
2024/12/01 12:53:33 192.168.1.110: F0:A7:31:xx:xx:xx on=true  "what watt" #children=0
2024/12/01 12:53:33 192.168.1.135: 50:91:E3:yy:yy:yy on=false  "no glow" #children=0
```

The `"..."` names are the aliases for the plugs that the user can
change. You can set this alias as follows:

```
$ ./tple --device=192.168.1.135 --alias="outside glow"
2024/12/01 13:00:26 192.168.1.135: 50:91:E3:yy:yy:yy on=false  "outside glow" #children=0
```

Next, to turn on the (`"outside glow"`) lights:

```
$ ./tple --device=192.168.1.135 --on
2024/12/01 13:00:28 192.168.1.135: 50:91:E3:yy:yy:yy on=true  "outside glow" #children=0
```

To turn those lights off again, but not output any text status
(suitable for a `crontab` entry):

```
$ ./tple --device=192.168.1.135 --off --status=false
```

Power strips have more than one socket per device, which enumerate as follows:

```
$ ./tple --device=192.168.1.157
2024/12/15 18:46:20 192.168.1.157: 50:91:E3:yy:yy:yy on=[true true]  "power couple" #children=2
```

Without specifying a socket for the action, the action will affect all
of the socket relay states.

To set the enabled state of the sockets individually, specify the
index of the sockets with the `--sockets=a,b` argument. In the above
example, there are two scockets, indexed `0` and `1`. For example, to
power off the 0th socket:

```
$ ./tple --device=192.168.1.157 --sockets=0 --off
2024/12/15 18:46:25 192.168.1.157: 50:91:E3:yy:yy:yy on=[false true]  "power couple" #children=2
```

To enable both sockets:

```
$ ./tple --device=192.168.1.157 --sockets=0,1 --on
2024/12/15 18:46:30 192.168.1.157: 50:91:E3:yy:yy:yy on=[true true]  "power couple" #children=2
```

The devices track time, and `tple` can initialize and read that
time. Note, the time is only settable with one second of precision, so
responses from the device are going to be up to one second wrong.

To update the time to be roughly the current time:

```
$ ./tple --device=192.168.1.157 --set-now
2024/12/17 06:30:53 device time is 2024-12-17 06:30:53 -0800 PST
```

To read the device's time:

```
$ ./tple --device=192.168.1.157 --time
2024/12/17 06:32:29 device time is 2024-12-17 06:32:29 -0800 PST
```

## Energy monitoring

Some of the TPLink devices support monitoring the energy consumption
stats for the plug. This can be used to monitor devices that are
plugged in. Not all devices support this feature. The KP110 is one of
them.

If a device is unable to measure power consumption, the following
command exits with an error:

```
$ ./tple --device=192.168.1.157 --emon
2025/07/02 20:59:46 failed to get E-Monitor state: no emeter responded
```

If, however, you use the same command on a KP110 device, you will see:

```
$ ./tple --device=192.168.1.110 --emon
2025/07/02 20:59:59 0.380A 116.213VAC 26.622W 3WH
```

Which lists:

- instantaneous current consumption (in Amps)
- instantaneous voltage detected by the plug (your AC Mains Voltage)
  - in the US, this should be somewhere in the 120 VAC range.
- instantaneous power consumption (in Watts)
- integrated energy consumption since the last `--emon-reset` (in Watt Hours)

That last value can be reset as follows:

```
$ ./tple --device=192.168.1.110 --emon-reset
2025/07/02 21:08:51 reset E-Monitor
```

To watch the power consumption over time, you can combine `--poll` and
`--emon` arguments as follows:

```
$ ./tple --device=192.168.1.110 --emon --poll=3s
2025/07/02 21:10:19 0.254A 115.152VAC 21.481W 0WH
2025/07/02 21:10:22 0.248A 115.688VAC 19.325W 0WH
2025/07/02 21:10:25 0.260A 115.444VAC 19.714W 0WH
2025/07/02 21:10:28 0.259A 115.628VAC 19.667W 0WH
2025/07/02 21:10:31 0.248A 115.879VAC 19.339W 0WH
2025/07/02 21:10:34 0.250A 115.818VAC 19.887W 0WH
^C
```

Which samples the `--emon` values once every 3 seconds until you kill the program with _Ctrl-C_.

## <a name="initial-setup-section"/>Initial Setup

When a device is newly unpacked, it has no configuration for
connecting to your WiFi network. You can use the following information
to enable fix that.

When you first plug in the smart plug to a power source, it boots in
setup mode. You can also return a device to this state with the
`--factory-reset` command line option. In setup mode, the plug acts as
WiFi network all of its own. Its SSID is of the form, `TP-LINK_Smart
Plug_XXYY`. Where `XXYY` is the last 4 hex digits of the plug's MAC
address. Typically, these are printed on the plug itself
somewhere. However, in most cases, it is unlikely you will be powering
up and configuring more than one plug at a time, so you can just use
your computer to try to connect to some WiFi network with a name like
this.

From a Rasperry Pi, networked via wired ethernet, you can find and
connect to the plug's setup network as follows:

```
$ sudo -s
# ifconfig wlan0 up
# iwlist wlan0 scan | grep ESSID
# iwconfig wlan0 essid "TP-LINK_Smart Plug_XXYY"
# dhclient wlan0
```

Alternatively, you can connect from any computer by selecting the WiFi
network of that name.

To complete this operation you will need three things:

- An alias name for your new plug. Example, `power`.
- The SSID of your network. Example, `MyWiFi`.
- The Password to connect to that network. Example, `Passphrase`.

From a computer connected to the `TP-LINK_...` network, run the
following commands (substituting your detailed parameters as
appropriate):

```
$ ./tple --scan=192.168.0.0/24
2024/12/18 13:06:48 192.168.0.1: F0:A7:31:WW:XX:YY on=true  "TP-LINK_Smart Plug_XXYY" #children=0
$ ./tple --device=192.168.0.1 --set-now
2024/12/18 13:07:22 device time is 2024-12-18 13:07:21 -0800 PST
$ ./tple --device=192.168.0.1 --alias="power"
2024/12/18 13:08:31 192.168.0.1: F0:A7:31:WW:XX:YY on=true  "power" #children=0
$ ./tple --device=192.168.0.1 --ssid="MyWiFi" --password="Passphrase"
2024/12/18 13:09:18 reconnect to device via "MyWiFi" WiFi network
$ 
```

If anything goes wrong, the plug will reestablish its `TP-LINK_...`
network, and you can try again. Note, I've found that it is required
to set an `--alias` for the networking change to take effect.

Sometimes the network you are trying to connect to is not very visible
from the location you have placed the plug. You can investigate this
with the `--wifi` command:

```
$ ./tple --wifi --device=192.168.0.1
2025/07/03 13:19:48 WiFi scan (WPA3 supported: 1)
2025/07/03 13:19:48   SSID="MyWiFi"                       KeyType: 3 RSSI=-70  dBm
2025/07/03 13:19:48   SSID="TheirWiFi"                    KeyType: 3 RSSI=-100 dBm
...
```

As noted above, you can use the `--scan` argument to locate the device
again from a computer networked to your `MyWiFi` network.

## TODO

Nothing planned.

## License info

The `tplinky` package is distributed with the same BSD 3-clause license
as that used by [golang](https://golang.org/LICENSE) itself.

## Reporting bugs and feature requests

Use the [github tplinky bug
tracker](https://github.com/tinkerator/tplinky/issues).
