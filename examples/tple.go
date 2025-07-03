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
	"strconv"
	"strings"
	"time"

	"zappem.net/pub/net/tplinky"
)

var (
	device    = flag.String("device", "", "IP address of target device")
	scan      = flag.String("scan", "", "summarize state of devices on network: <ip>/<bits>")
	timeout   = flag.Duration("timeout", 2*time.Second, "how long to wait for device")
	verbose   = flag.Bool("v", false, "list all status info from devices")
	on        = flag.Bool("on", false, "set the device to enabled")
	off       = flag.Bool("off", false, "set the device to disabled")
	stat      = flag.Bool("status", true, "get device(s) status")
	sockets   = flag.String("sockets", "", "comma separated socket indexes")
	getTime   = flag.Bool("time", false, "request time from --device")
	setNow    = flag.Bool("set-now", false, "set time on --device from time.Now()")
	alias     = flag.String("alias", "", "set alias for --device")
	factory   = flag.Bool("factory-reset", false, "factory reset --device")
	ssid      = flag.String("ssid", "", "sets the WiFi network for --device to connect to")
	password  = flag.String("password", "", "password to connect to --ssid network")
	emon      = flag.Bool("emon", false, "read the current E-Meter status")
	emonReset = flag.Bool("emon-reset", false, "reset the E-Meter state")
	poll      = flag.Duration("poll", 0, "polling time interval for E-Meter reads")
)

// status converts a device Sysinfo status into a string.
func status(dev *tplinky.Sysinfo) string {
	if *verbose {
		return fmt.Sprintf("%#v", dev)
	}
	if len(dev.Children) != 0 {
		var relays []bool
		for _, x := range dev.Children {
			relays = append(relays, x.State != 0)
		}
		return fmt.Sprintf("%s on=%v %q #children=%d", dev.Mac, relays, dev.Alias, len(dev.Children))
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

	var indexes []int
	dups := make(map[int]bool)
	if *sockets != "" {
		for _, s := range strings.Split(*sockets, ",") {
			n, err := strconv.Atoi(s)
			if err != nil {
				log.Fatalf("unrecognized socket index=%q from %q: %v", s, *sockets, err)
			}
			if dups[n] {
				log.Fatalf("duplicate socket %d vs %v", n, indexes)
			}
			dups[n] = true
			indexes = append(indexes, n)
		}
	}

	dev, err := tplinky.DialTimeout(*device, *timeout)
	if err != nil {
		log.Fatalf("failed to connect to %q: %v", *device, err)
	}
	defer dev.Close()

	if *ssid != "" {
		if err := dev.SetWiFi(*ssid, *password); err != nil {
			log.Fatalf("unable to set WiFi to %q", *ssid)
		}
		log.Printf("reconnect to device via %q WiFi network", *ssid)
		return
	}

	if *emonReset {
		if err := dev.EMonReset(); err != nil {
			log.Fatalf("failed to reset E-Monitor: %v", err)
		}
		log.Print("reset E-Monitor")
		return
	}

	if *factory {
		s, err := dev.GetStatus()
		if err != nil {
			log.Fatalf("unable to get status: %v", err)
		}
		if err := dev.FactoryReset(); err != nil {
			log.Fatalf("failed to factory reset device: %v", err)
		}
		log.Printf("factory resetting device %q (%s)...", s.Alias, *device)
		sub := s.Mac[len(s.Mac)-5:]
		log.Printf("look for WiFi SSID: 'TP-Link_Smart Plug_%s%s'", sub[0:2], sub[3:])
		return
	}

	if *alias != "" {
		if err := dev.SetAlias(*alias); err != nil {
			log.Fatalf("unable to set device alias: %v", err)
		}
		s, err := dev.GetStatus()
		if err != nil {
			log.Fatalf("unable to get status: %v", err)
		}
		log.Printf("%s: %s", *device, status(s))
		return
	}
	if *setNow {
		if err := dev.SetTime(time.Now()); err != nil {
			log.Fatalf("unable to set current time: %v", err)
		}
	}
	if *getTime || *setNow {
		t, err := dev.GetTime()
		if err != nil {
			log.Fatalf("unable to get time: %v", err)
		}
		log.Printf("device time is %v", t)
		return
	}
	if *emon {
		for {
			s, err := dev.EMonState()
			if err != nil {
				log.Fatalf("failed to get E-Monitor state: %v", err)
			}
			log.Printf("%.3fA %.3fVAC %.3fW %dWH", float64(s.CurrentMA)/1e3, float64(s.VoltageMV)/1e3, float64(s.PowerMW)/1e3, s.TotalWH)
			if *poll == 0 {
				break
			}
			time.Sleep(*poll)
		}
		return
	}
	if *on {
		if *off {
			log.Fatal("use --on or --off not both")
		}
		if len(indexes) != 0 {
			if err := dev.EnableSocket(true, indexes...); err != nil {
				log.Fatalf("failed to turn on device %q(sockets%v): %v", *device, indexes, err)
			}
		} else if err := dev.Enable(true); err != nil {
			log.Fatalf("failed to turn on device %q: %v", *device, err)
		}
	} else if *off {
		if len(indexes) != 0 {
			if err := dev.EnableSocket(false, indexes...); err != nil {
				log.Fatalf("failed to turn off device %q(sockets%v): %v", *device, indexes, err)
			}
		} else if err := dev.Enable(false); err != nil {
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
