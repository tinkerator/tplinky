# tplinky - a Go package for simple interactions with TPLink smart plugs.

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

To use this device (on already configured devices) you can try the following example:

```
$ git clone https://github.com/tinkerator/tplinky.git
$ cd tplinky
$ go build examples/tple
$ ./tple --scan=192.168.1.0/24
2024/12/01 12:53:33 192.168.1.110: F0:A7:31:xx:xx:xx on=true  "what watt" #children=0
2024/12/01 12:53:33 192.168.1.135: 50:91:E3:yy:yy:yy on=false  "outside glow" #children=0
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

## TODO

Figure out how to support plug muti-port switches. The top level
devices have children. The code can count them but not yet interact
with them.

## License info

The `tplinky` package is distributed with the same BSD 3-clause license
as that used by [golang](https://golang.org/LICENSE) itself.

## Reporting bugs and feature requests

Use the [github tplinky bug
tracker](https://github.com/tinkerator/tplinky/issues).
