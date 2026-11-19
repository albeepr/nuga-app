package hid

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/sstallion/go-hid"
)

// VIA raw HID interface constants (QMK `RAW_USAGE_PAGE` / `RAW_USAGE_ID`).
const (
	viaUsagePage  = 0xFF60
	viaUsage      = 0x61
	viaReportSize = 32
	viaTimeout    = 500 * time.Millisecond
	viaRetries    = 3

	// ViaUnhandled is the first byte of a response to an unknown command.
	ViaUnhandled = 0xFF
	// ViaCmdProtocolVersion is the VIA `id_get_protocol_version` command.
	ViaCmdProtocolVersion = 0x01
)

// VIAModel describes a QMK/VIA keyboard supported by the library.
type VIAModel struct {
	Model     string
	VendorID  uint16
	ProductID uint16
}

// VIAModels is the list of known QMK/VIA NuPhy keyboards.
var VIAModels = []VIAModel{
	// 0x3265: shipping NuPhy firmware; 0x3266: NuPhy's public QMK source / VIA JSON.
	{Model: "Air96V2", VendorID: 0x19F5, ProductID: 0x3265},
	{Model: "Air96V2", VendorID: 0x19F5, ProductID: 0x3266},
}

// ErrVIAUnhandled is returned when the keyboard does not understand a VIA command.
var ErrVIAUnhandled = errors.New("command is not supported by keyboard firmware")

// ErrVIATimeout is returned when the keyboard does not answer a VIA command in time.
var ErrVIATimeout = errors.New("keyboard did not respond")

// VIADevice is a handle for a QMK keyboard talking the VIA raw HID protocol.
type VIADevice struct {
	Info *DeviceInfo

	// ProtocolVersion is the VIA protocol version reported by firmware.
	ProtocolVersion uint16

	handle *hid.Device
	debug  bool
	lock   *sync.Mutex
}

// Command sends one 32-byte VIA report and waits for the matching response.
// The response is matched by echoing the first `match` bytes of the request.
func (d *VIADevice) Command(payload []byte, match int) ([]byte, error) {
	if len(payload) > viaReportSize {
		return nil, fmt.Errorf("VIA payload too long: %d", len(payload))
	}
	var err error
	var resp []byte
	for i := 0; i < viaRetries; i++ {
		resp, err = d.command(payload, match)
		if err == nil || errors.Is(err, ErrVIAUnhandled) {
			return resp, err
		}
	}
	return resp, err
}

func (d *VIADevice) command(payload []byte, match int) ([]byte, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	// Drop any stale input reports left over from an earlier timed-out request.
	stale := make([]byte, viaReportSize)
	for {
		n, err := d.handle.ReadWithTimeout(stale, time.Millisecond)
		if err != nil || n == 0 {
			break
		}
	}

	// hidapi needs the report ID (0 for raw HID) as the first byte.
	buf := make([]byte, viaReportSize+1)
	copy(buf[1:], payload)
	if d.debug {
		log.Printf("VIA send: % x", payload)
	}
	if _, err := d.handle.Write(buf); err != nil {
		return nil, err
	}

	deadline := time.Now().Add(viaTimeout)
	for time.Now().Before(deadline) {
		resp := make([]byte, viaReportSize)
		n, err := d.handle.ReadWithTimeout(resp, time.Until(deadline))
		if err != nil {
			if errors.Is(err, hid.ErrTimeout) {
				break
			}
			return nil, err
		}
		if n == 0 {
			continue
		}
		if d.debug {
			log.Printf("VIA recv: % x", resp[:n])
		}
		if resp[0] == ViaUnhandled {
			return resp, ErrVIAUnhandled
		}
		if matches(payload, resp, match) {
			return resp, nil
		}
	}
	return nil, ErrVIATimeout
}

func matches(req, resp []byte, n int) bool {
	if n > len(req) {
		n = len(req)
	}
	for i := 0; i < n; i++ {
		if req[i] != resp[i] {
			return false
		}
	}
	return true
}

// Send implements Handler. It sends a VIA command and ignores the response body.
func (d *VIADevice) Send(payload []byte) error {
	_, err := d.Command(payload, 1)
	return err
}

// Read is not meaningful for VIA (responses are bound to requests).
func (d *VIADevice) Read(_ int) ([]byte, error) {
	return nil, ErrVIAUnhandled
}

// Request implements Handler.
func (d *VIADevice) Request(payload []byte, _ int) ([]byte, error) {
	return d.Command(payload, 1)
}

// GetInfo implements Handler.
func (d *VIADevice) GetInfo() *DeviceInfo {
	return d.Info
}

// Close implements Handler.
func (d *VIADevice) Close() error {
	return d.handle.Close()
}

// Debug returns a copy of the device with verbose logging enabled.
func (d *VIADevice) Debug() *VIADevice {
	dev := *d
	dev.debug = true
	return &dev
}

// OpenVIA looks for a connected, known QMK/VIA keyboard and opens it.
func OpenVIA() (*VIADevice, error) {
	for _, m := range VIAModels {
		var path string
		err := hid.Enumerate(m.VendorID, m.ProductID, func(info *hid.DeviceInfo) error {
			if info.UsagePage == viaUsagePage && info.Usage == viaUsage {
				path = info.Path
			}
			return nil
		})
		if err != nil || path == "" {
			continue
		}
		handle, err := hid.OpenPath(path)
		if err != nil {
			return nil, err
		}
		info, err := handle.GetDeviceInfo()
		if err != nil {
			_ = handle.Close()
			return nil, err
		}
		dev := &VIADevice{
			Info: &DeviceInfo{
				Model:    m.Model,
				Path:     info.Path,
				Firmware: FormatVersion(info.ReleaseNbr),
			},
			handle: handle,
			lock:   new(sync.Mutex),
		}
		resp, err := dev.Command([]byte{ViaCmdProtocolVersion}, 1)
		if err != nil {
			_ = handle.Close()
			return nil, fmt.Errorf("VIA handshake failed: %w", err)
		}
		dev.ProtocolVersion = uint16(resp[1])<<8 | uint16(resp[2])
		return dev, nil
	}
	return nil, ErrNotFound
}
