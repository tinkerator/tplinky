package tplinky

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

var (
	// ErrTimeFailed is returned when a call to obtain the time failed.
	ErrTimeFailed = errors.New("get_time failed")

	// ErrNoEMeter is returned if the target device failed to perform
	// emeter commands.
	ErrNoEMeter = errors.New("no emeter responded")

	// ErrNoWiFiScan is returned for a failed wifi scan attempt.
	ErrNoWiFiScan = errors.New("wifi scan unavailable")
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
	result = make(map[string]*Sysinfo, 2)
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

// GetTime reads the time from the device.
func (c *Conn) GetTime() (time.Time, error) {
	resp, err := c.Send(Control{
		Time: &DevTime{
			GetTime: &RawNull,
		},
	})
	t := time.Now()
	if err != nil {
		return t, err
	}
	vs := resp.Time
	if vs == nil || vs.GetTime == nil {
		return t, ErrTimeFailed
	}
	x := vs.GetTime
	t = time.Date(x.Year, time.Month(x.Month), x.MDay, x.Hour, x.Min, x.Sec, 0, t.Location())
	return t, err
}

// SetTime reads the time from the device.
func (c *Conn) SetTime(t time.Time) error {
	_, err := c.Send(Control{
		Time: &DevTime{
			SetTimeZone: &TimeZone{
				Year:  t.Year(),
				Month: int(t.Month()),
				MDay:  t.Day(),
				Hour:  t.Hour(),
				Min:   t.Minute(),
				Sec:   t.Second(),
				Index: 90,
			},
		},
	})
	return err
}

// SetAlias sets the alias name for the device.
func (c *Conn) SetAlias(name string) error {
	_, err := c.Send(Control{
		System: &SystemCommands{
			SetDevAlias: &SystemCommandParameters{
				Alias: &name,
			},
		},
	})
	return err
}

// FactoryReset resets the device to its factory default
// settings. This will make the device forget its WiFi settings and
// revert it to broadcasting a self-generated WiFi network:
// `"TP-LINK_Smart Plug_XXXX"`.
func (c *Conn) FactoryReset() error {
	one := 1
	_, err := c.Send(Control{
		System: &SystemCommands{
			Reset: &SystemCommandParameters{
				Delay: &one,
			},
		},
	})
	return err
}

// SetWiFi sets the ssid and password for the preferred
// network. Performing this command will cause the device to
// disconnect from the current network, and connect with the provided
// parameters.
func (c *Conn) SetWiFi(ssid, password string) error {
	_, err := c.Send(Control{
		NetIf: &NetIfCommands{
			SetStaInfo: &StaInfoParameters{
				SSID:     ssid,
				Password: password,
				KeyType:  3,
			},
		},
	})
	return err
}

// ListWiFi gets the list of WiFi Access Points that the device can
// see. This can be useful for positioning your WiFi Router and or
// understanding the reliability of different plug placement choices.
// Less negative RSSI values imply higher signal strength. For
// example, "-51" is better than "-88".
func (c *Conn) ListWiFi() (*GetScanInfoResponse, error) {
	for {
		resp, err := c.Send(Control{
			NetIf: &NetIfCommands{
				GetScanInfo: &GetScanInfoParameters{
					Refresh: 1,
				},
			},
		})
		if err != nil {
			return nil, err
		}
		if resp.NetIf != nil {
			if detail := resp.NetIf.GetScanInfoResponse; detail != nil {
				return detail, nil
			}
		}
		break
	}
	return nil, ErrNoWiFiScan
}

// EMonReset resets the target device's E-Meter values (integrated
// energy measurement).
func (c *Conn) EMonReset() error {
	resp, err := c.Send(Control{
		EMeter: &EMeter{
			EraseEMeterStat: &EMeterResponse{},
		},
	})
	if err != nil {
		return err
	}
	if resp.EMeter == nil || resp.EMeter.EraseEMeterStat == nil {
		return ErrNoEMeter
	}
	if eCode := resp.EMeter.EraseEMeterStat.ErrCode; eCode != 0 {
		return fmt.Errorf("emeter error %d", eCode)
	}
	return nil
}

// EMonState reads a measurement of the current E-Meter values.
func (c *Conn) EMonState() (*EMeterResponse, error) {
	resp, err := c.Send(Control{
		EMeter: &EMeter{
			GetRealTime: &EMeterResponse{},
		},
	})
	if err != nil {
		return nil, err
	}
	if resp.EMeter == nil || resp.EMeter.GetRealTime == nil {
		return nil, ErrNoEMeter
	}
	if eCode := resp.EMeter.GetRealTime.ErrCode; eCode != 0 {
		return nil, fmt.Errorf("emeter error %d", eCode)
	}
	return resp.EMeter.GetRealTime, nil
}
