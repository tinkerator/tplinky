// Program tple is a simple command line wrapper example for using the
// tplinky package.
//
// For help using this tool, invoke it with the --help argument.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"zappem.net/pub/net/tplinky"
)

var (
	device  = flag.String("device", "", "IP address of target device")
	scan    = flag.String("scan", "", "summarize state of devices on network: <ip>/<bits>")
	timeout = flag.Duration("timeout", 2*time.Second, "how long to wait for device")
	verbose = flag.Bool("v", false, "list all status info from devices")
	on      = flag.Bool("on", false, "set the device to enabled")
	off     = flag.Bool("off", false, "set the device to disabled")
	stat    = flag.Bool("status", true, "get device(s) status")
)

// status converts a device Sysinfo status into a string.
func status(dev *tplinky.Sysinfo) string {
	if *verbose {
		return fmt.Sprintf("%#v", dev)
	}
	return fmt.Sprintf("%s on=%-5v %q #children=%d", dev.Mac, dev.RelayState != 0, dev.Alias, len(dev.Children))
}

func main() {
	flag.Parse()

	if *scan != "" {
		devices := tplinky.Scan(*scan, *timeout)
		if len(devices) == 0 {
			log.Fatal("no devices found")
		}
		for ip, dev := range devices {
			log.Printf("%s: %s", ip, status(dev))
		}
		os.Exit(0)
	}

	dev, err := tplinky.DialTimeout(*device, *timeout)
	if err != nil {
		log.Fatalf("failed to connect to %q: %v", *device, err)
	}
	defer dev.Close()

	if *on {
		if *off {
			log.Fatal("use --on or --off not both")
		}
		if err := dev.Enable(true); err != nil {
			log.Fatalf("failed to turn on device %q: %v", *device, err)
		}
	} else if *off {
		if err := dev.Enable(false); err != nil {
			log.Fatalf("failed to turn off device %q: %v", *device, err)
		}
	}
	if *stat {
		sys, err := dev.GetStatus()
		if err != nil {
			log.Fatalf("failed to get status for %q: %v", *device, err)
		}
		log.Printf("%s: %s", *device, status(sys))
	}
}
