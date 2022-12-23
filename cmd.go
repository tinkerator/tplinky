package tplinky

import (
	"encoding/binary"
	"errors"
	"net"
	"sync"
	"time"
)

// GetStatus requests the status of the device.
func (c *Conn) GetStatus() (*Sysinfo, error) {
	r, err := c.Send(Control{
		System: &SystemCommands{
			GetSysinfo: &GetSysinfo{},
		},
	})
	if err != nil {
		return nil, err
	}
	if r.System == nil {
		return nil, errors.New("response did not contain sysinfo")
	}
	return r.System.GetSysinfo, nil
}

type ip4sysinfo struct {
	addr4 string
	sys   *Sysinfo
}

// Scan scans all of the IPV4 addresses on a CIDR subnet for tplink
// devices, returning a map of their current status. The scan is done
// in parallel. The network is provided in [net.ParseCIDR] format.
func Scan(network string, timeout time.Duration) (result map[string]*Sysinfo) {
	result = make(map[string]*Sysinfo)
	_, nInfo, err := net.ParseCIDR(network)
	if err != nil || len(nInfo.Mask) != 4 {
		return
	}

	mask := binary.BigEndian.Uint32(nInfo.Mask)
	first := binary.BigEndian.Uint32(nInfo.IP)
	last := (first & mask) | ^mask

	var wg0 sync.WaitGroup
	var wg sync.WaitGroup
	ch := make(chan ip4sysinfo)
	wg0.Add(1)
	go func() {
		defer wg0.Done()
		for r := range ch {
			result[r.addr4] = r.sys
		}
	}()
	for n := first + 1; n < last; n++ {
		ip := make([]byte, 4)
		binary.BigEndian.PutUint32(ip, n)
		target := net.IP(ip).String()

		wg.Add(1)
		go func() {
			defer wg.Done()
			c, err := DialTimeout(target, timeout)
			if err != nil {
				return
			}
			defer c.Close()
			sys, err := c.GetStatus()
			if err != nil {
				return
			}
			ch <- ip4sysinfo{
				addr4: target,
				sys:   sys,
			}
		}()
	}
	wg.Wait()
	close(ch)
	wg0.Wait()
	return
}

// Enable attempts to force the power-on state of a tplink device.
func (c *Conn) Enable(on bool) error {
	var en = 0
	if on {
		en = 1
	}
	_, err := c.Send(Control{
		System: &SystemCommands{
			SetRelayState: &SystemCommandParameters{
				State: &en,
			},
		},
	})
	return err
}
