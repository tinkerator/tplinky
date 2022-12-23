// Package tplinky provides an API for accessing and controlling TP-link smart plug
// devices.
package tplinky

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"time"
)

// Int converts a number value into a pointer to this number value.
func Int(n int) *int {
	return &n
}

// String converts a string value into a pointer to this string value.
func String(t string) *string {
	return &t
}

// SystemCommandParameters holds the command parameters for conversing
// with a tp-link smart plug.
type SystemCommandParameters struct {
	Delay *int    `json:"delay,omitempty"`
	State *int    `json:"state,omitempty"`
	Off   *int    `json:"off,omitempty"`
	Alias *string `json:"alias,omitempty"`
}

// MacAddr holds the tp-link preferred structure for representing a
// device's MAC address.
type MacAddr struct {
	Mac string `json:"mac"`
}

// DeviceID holds the tp-link preferred structure for representing a
// device's text identifier.
type DeviceID struct {
	DeviceID string `json:"deviceId"`
}

// HWID holds the tp-link preferred structure for representing a
// device's HW ID.
type HWID struct {
	HWID string `json:"hwId"`
}

// DevLocation captures the tp-link device's sense of where it is in
// the physical world.
type DevLocation struct {
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
}

// ActionType holds the tp-link preferred numeric action type value.
type ActionType struct {
	Type int `json:"type"`
}

// RawNull is used to ensure a null is in the output.
var RawNull = json.RawMessage(`null`)

// SystemCommands holds a superset of the command structure for
// communicating with the tp-link smartplug device.
type SystemCommands struct {
	GetSysinfo     *GetSysinfo              `json:"get_sysinfo,omitempty"`
	Reboot         *SystemCommandParameters `json:"reboot,omitempty"`
	Reset          *SystemCommandParameters `json:"reset,omitempty"`
	SetRelayState  *SystemCommandParameters `json:"set_relay_state,omitempty"`
	SetLEDOff      *SystemCommandParameters `json:"set_led_off,omitempty"`
	SetDevAlias    *SystemCommandParameters `json:"set_dev_alias,omitempty"`
	SetMacAddr     *MacAddr                 `json:"set_mac_addr,omitempty"`
	SetDeviceID    *DeviceID                `json:"set_device_id,omitempty"`
	SetHWID        *HWID                    `json:"set_hw_id,omitempty"`
	SetDevLocation *DevLocation             `json:"set_dev_location,omitempty"`
}

// Control is a structure containing the TP-link control syntax as
// described here:
//
//	https://github.com/softScheck/tplink-smartplug/blob/master/tplink-smarthome-commands.txt
type Control struct {
	System *SystemCommands `json:"system,omitempty"`
}

// GetSysinfo holds the empty request for obtaining Sysinfo from the
// target tp-link smartplug.
type GetSysinfo struct{}

// Sysinfo holds the detailed data response from that target tp-link
// device.
type Sysinfo struct {
	SWVer      string     `json:"sw_ver,omitempty"`
	HWVer      string     `json:"hw_ver,omitempty"`
	Type       string     `json:"type,omitempty"`
	Model      string     `json:"model,omitempty"`
	Mac        string     `json:"mac,omitempty"`
	DevName    string     `json:"dev_name,omitempty"`
	Alias      string     `json:"alias,omitempty"`
	RelayState int        `json:"relay_state"`
	OnTime     int        `json:"on_time"`
	ActiveMode string     `json:"active_mode"`
	Feature    string     `json:"feature"`
	Updating   int        `json:"updating"`
	IconHash   string     `json:"icon_hash"`
	RSSI       int        `json:"rssi"`
	LEDOff     int        `json:"led_off"`
	LongitudeI int        `json:"longitude_i"`
	LatitudeI  int        `json:"latitude_i"`
	HWID       string     `json:"hwId"`
	FWID       string     `json:"fwId"`
	DeviceID   string     `json:"deviceId"`
	OEMID      string     `json:"oemId"`
	NextAction ActionType `json:"next_action"`
	ErrCode    int        `json:"err_code"`
}

// SystemResponse wraps Sysinfo.
type SystemResponse struct {
	GetSysinfo *Sysinfo `json:"get_sysinfo,omitempty"`
}

// Response is a structure containing the TP-link control response.
type Response struct {
	System *SystemResponse `json:"system,omitempty"`
}

// Conn holds an open connection to a TP-Link device. It uses the port
// 9999 TCP protocol for communication.
type Conn struct {
	target string
	conn   net.Conn
}

// Encode translates to and from the obfuscation format of the tp-link
// TCP protocol. This same function is used to Read and Write the
// device.
//
// Detailed discussion here:
//
//	https://www.softscheck.com/en/reverse-engineering-tp-link-hs110/
func Encode(p []byte) *bytes.Buffer {
	b := &bytes.Buffer{}
	binary.Write(b, binary.BigEndian, int32(len(p)))
	key := byte(171)
	for _, c := range p {
		key = key ^ c
		b.WriteByte(key)
	}
	return b
}

// Decode unpacks a reply from the TP-Link device.
func Decode(p []byte) *bytes.Buffer {
	input := bytes.NewBuffer(p)
	var n int32
	binary.Read(input, binary.BigEndian, &n)
	b := &bytes.Buffer{}
	key := byte(171)
	for {
		c, err := input.ReadByte()
		if err != nil {
			break
		}
		b.WriteByte(key ^ c)
		key = c
	}
	return b
}

// Read reads and decodes upto len(p) bytes from the target.
func (c *Conn) Read(p []byte) (n int, err error) {
	if n, err = c.conn.Read(p); err != nil {
		return n, err
	}
	Encode(p[:n])
	return n, nil
}

// Write writes some bytes encoded to the target.
func (c *Conn) Write(p []byte) (int, error) {
	Encode(p)
	return c.conn.Write(p)
}

// ErrNotOpen is an error that indicates that the target device does
// not have an open connection.
var ErrNotOpen = errors.New("not open")

// Close closes an open connection to a tp-link device on the network.
// Once closed the connection will no longer function for
// communication purposes.
func (c *Conn) Close() error {
	if c == nil || c.target == "" {
		return ErrNotOpen
	}
	c.target = ""
	return c.conn.Close()
}

// Dial the TP-link target with a custom dial timeout, returning an
// open connection or an error.
func DialTimeout(target string, timeout time.Duration) (*Conn, error) {
	if !strings.Contains(target, ":") {
		target += ":9999"
	}
	opt := net.Dialer{Timeout: timeout}
	conn, err := opt.Dial("tcp", target)
	if err != nil {
		return nil, err
	}
	return &Conn{
		target: target,
		conn:   conn,
	}, nil
}

// Dial the TP-link target with a 2 second dial timeout.
func Dial(target string) (*Conn, error) {
	return DialTimeout(target, 2*time.Second)
}

// Send a command to the device and decode the response.
func (c *Conn) Send(cmd Control) (*Response, error) {
	j, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	json.Compact(&b, j)
	if _, err := c.conn.Write(Encode(b.Bytes()).Bytes()); err != nil {
		return nil, err
	}
	d := make([]byte, 4096)
	n, err := c.Read(d)
	if err != nil {
		return nil, err
	}
	x := Decode(d[:n])
	var r Response
	if err := json.Unmarshal(x.Bytes(), &r); err != nil {
		return nil, err
	}
	return &r, nil
}
