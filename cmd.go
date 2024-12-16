package tplinky

import (
	"encoding/binary"
	"errors"
	"fmt"
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
	// Quirk of this API. If the connected device has more than
	// one socket, alias the returned *Sysinfo field RelayState to
	// the logical OR of all of the socket states. Empirically,
	// this value is always 0 in such systems, and does not change
	// if you attempt to set it directly.
	if len(r.System.GetSysinfo.Children) != 0 {
		r.System.GetSysinfo.RelayState = 0
		for _, child := range r.System.GetSysinfo.Children {
			if child.State != 0 {
				r.System.GetSysinfo.RelayState = 1
				break
			}
		}
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
	current, err := c.GetStatus()
	if err != nil {
		return err
	}
	if len(current.Children) != 0 {
		// For a power strip, force all sockets to desired state.
		sockets := make([]int, len(current.Children))
		for i := range sockets {
			sockets[i] = i
		}
		return c.EnableSocket(on, sockets...)
	}
	var en = 0
	if on {
		en = 1
	}
	if current.RelayState == en {
		// No need to do anything if device is currently in
		// desired state.
		return nil
	}
	_, err = c.Send(Control{
		System: &SystemCommands{
			SetRelayState: &SystemCommandParameters{
				State: &en,
			},
		},
	})
	return err
}

// EnableSocket attempts to force the power-on state of the specified
// sockets of a power strip.
func (c *Conn) EnableSocket(on bool, sockets ...int) error {
	current, err := c.GetStatus()
	if err != nil {
		return err
	}
	var en = 0
	if on {
		en = 1
	}
	var children []string
	for _, i := range sockets {
		if i < 0 || i >= len(current.Children) {
			return fmt.Errorf("socket=%d not found in %d sockets", i, len(current.Children))
		}
		if en != current.Children[i].State {
			children = append(children, current.Children[i].ID)
		}
	}
	for i := range children {
		_, err = c.Send(Control{
			Context: &ControlContext{
				ChildIDs: children[i : i+1],
			},
			System: &SystemCommands{
				SetRelayState: &SystemCommandParameters{
					State: &en,
				},
			},
		})
		if err != nil {
			break
		}
	}
	return err
}
