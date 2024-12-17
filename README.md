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

To use this device (on already configured devices, see TODO section)
you can try the following example:

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

## TODO

Add some support for initializing the device from a factory
reset. Notes for this (from a Raspberry Pi, networked via `eth0`):

```
$ sudo -s
# ifconfig wlan0 up
# iwlist wlan0 scan | grep ESSID
# iwconfig wlan0 essid "TP-LINK_Smart Plug_XXXX"
# dhclient wlan0
```

JSON to command it to join your local network (after the last command
here, the device will stop responding to the above ESSID) from a
Raspberry Pi:

```
{"netif":{"set_stainfo":{"ssid":"MYSSID","password":"BigSecret","key_type":3}}}
```

## License info

The `tplinky` package is distributed with the same BSD 3-clause license
as that used by [golang](https://golang.org/LICENSE) itself.

## Reporting bugs and feature requests

Use the [github tplinky bug
tracker](https://github.com/tinkerator/tplinky/issues).
