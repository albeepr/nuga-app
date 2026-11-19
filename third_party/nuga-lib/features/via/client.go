// Package via implements the host side of the VIA raw HID protocol
// (QMK `quantum/via.h`, protocol version 12) used by NuPhy V2 keyboards.
package via

import (
	"errors"
	"fmt"

	"github.com/mishamyrt/nuga-lib/hid"
)

// Command IDs (quantum/via.h).
const (
	CmdGetProtocolVersion   = 0x01
	CmdGetKeyboardValue     = 0x02
	CmdGetKeycode           = 0x04
	CmdSetKeycode           = 0x05
	CmdKeymapReset          = 0x06
	CmdCustomSetValue       = 0x07
	CmdCustomGetValue       = 0x08
	CmdCustomSave           = 0x09
	CmdMacroGetCount        = 0x0C
	CmdMacroGetBufferSize   = 0x0D
	CmdMacroGetBuffer       = 0x0E
	CmdMacroSetBuffer       = 0x0F
	CmdMacroReset           = 0x10
	CmdGetLayerCount        = 0x11
	CmdKeymapGetBuffer      = 0x12
	CmdKeymapSetBuffer      = 0x13
	ChannelCustom           = 0
	ChannelRGBMatrix        = 3
	reportSize              = 32
	bufferChunk             = reportSize - 4
)

// ErrBufferOverflow is returned when data does not fit device buffer.
var ErrBufferOverflow = errors.New("data does not fit keyboard memory")

// Commander sends one VIA report and returns the matching response.
// Responses are matched by echoing the first `match` request bytes.
type Commander interface {
	Command(payload []byte, match int) ([]byte, error)
}

// Client is a typed VIA protocol client.
type Client struct {
	dev Commander
}

// New creates VIA client.
func New(dev Commander) *Client {
	return &Client{dev: dev}
}

// GetCustom reads custom value.
func (c *Client) GetCustom(channel, id byte, size int) ([]byte, error) {
	resp, err := c.dev.Command([]byte{CmdCustomGetValue, channel, id}, 3)
	if err != nil {
		return nil, err
	}
	return resp[3 : 3+size], nil
}

// SetCustom writes custom value.
func (c *Client) SetCustom(channel, id byte, data ...byte) error {
	_, err := c.dev.Command(append([]byte{CmdCustomSetValue, channel, id}, data...), 3)
	return err
}

// SaveCustom persists custom channel values to EEPROM.
func (c *Client) SaveCustom(channel byte) error {
	_, err := c.dev.Command([]byte{CmdCustomSave, channel}, 2)
	return err
}

// LayerCount returns dynamic keymap layer count.
func (c *Client) LayerCount() (int, error) {
	resp, err := c.dev.Command([]byte{CmdGetLayerCount}, 1)
	if err != nil {
		return 0, err
	}
	return int(resp[1]), nil
}

// ReadKeymap reads the whole dynamic keymap: layers*rows*cols keycodes.
func (c *Client) ReadKeymap(layers, rows, cols int) ([]uint16, error) {
	size := layers * rows * cols * 2
	raw := make([]byte, 0, size)
	for offset := 0; offset < size; offset += bufferChunk {
		n := minInt(bufferChunk, size-offset)
		resp, err := c.dev.Command([]byte{CmdKeymapGetBuffer, byte(offset >> 8), byte(offset), byte(n)}, 4)
		if err != nil {
			return nil, err
		}
		raw = append(raw, resp[4:4+n]...)
	}
	codes := make([]uint16, len(raw)/2)
	for i := range codes {
		codes[i] = uint16(raw[i*2])<<8 | uint16(raw[i*2+1])
	}
	return codes, nil
}

// SetKeycode writes one key.
func (c *Client) SetKeycode(layer, row, col byte, code uint16) error {
	_, err := c.dev.Command([]byte{CmdSetKeycode, layer, row, col, byte(code >> 8), byte(code)}, 4)
	return err
}

// ResetKeymap restores firmware default keymap.
func (c *Client) ResetKeymap() error {
	_, err := c.dev.Command([]byte{CmdKeymapReset}, 1)
	return err
}

// MacroCount returns number of macro slots.
func (c *Client) MacroCount() (int, error) {
	resp, err := c.dev.Command([]byte{CmdMacroGetCount}, 1)
	if err != nil {
		return 0, err
	}
	return int(resp[1]), nil
}

// MacroBufferSize returns macro storage size in bytes.
func (c *Client) MacroBufferSize() (int, error) {
	resp, err := c.dev.Command([]byte{CmdMacroGetBufferSize}, 1)
	if err != nil {
		return 0, err
	}
	return int(resp[1])<<8 | int(resp[2]), nil
}

// ReadMacroBuffer reads macro storage.
func (c *Client) ReadMacroBuffer(size int) ([]byte, error) {
	out := make([]byte, 0, size)
	for offset := 0; offset < size; offset += bufferChunk {
		n := minInt(bufferChunk, size-offset)
		resp, err := c.dev.Command([]byte{CmdMacroGetBuffer, byte(offset >> 8), byte(offset), byte(n)}, 4)
		if err != nil {
			return nil, err
		}
		out = append(out, resp[4:4+n]...)
	}
	return out, nil
}

// WriteMacroBuffer writes macro storage, zero padding to the full size.
func (c *Client) WriteMacroBuffer(data []byte, size int) error {
	if len(data) > size {
		return fmt.Errorf("%w: macros need %d bytes, keyboard has %d", ErrBufferOverflow, len(data), size)
	}
	buf := make([]byte, size)
	copy(buf, data)
	for offset := 0; offset < size; offset += bufferChunk {
		n := minInt(bufferChunk, size-offset)
		payload := append([]byte{CmdMacroSetBuffer, byte(offset >> 8), byte(offset), byte(n)}, buf[offset:offset+n]...)
		if _, err := c.dev.Command(payload, 4); err != nil {
			return err
		}
	}
	return nil
}

// ResetMacros clears all macros.
func (c *Client) ResetMacros() error {
	_, err := c.dev.Command([]byte{CmdMacroReset}, 1)
	return err
}

// IsUnhandled reports whether err means the firmware does not support a command.
func IsUnhandled(err error) bool {
	return errors.Is(err, hid.ErrVIAUnhandled)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
