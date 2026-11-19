package qmklight

import (
	"math"

	"github.com/mishamyrt/nuga-lib/features/light"
)

// Side light values on VIA custom channel 0. Requires NuPhy firmware with
// the via_side.c patch (stock firmware answers "unhandled").
const (
	channelCustom    = 0
	sideBrightness   = 1
	sideMode         = 2
	sideSpeed        = 3
	sideColour       = 4
	sideValueBattery = 5
	sideValueLink    = 6
	sideValueOS      = 7

	fwSideOff     = 4
	fwSideRainbow = 8
)

// SideDomain lists side light modes. Nuga code = firmware mode + 1, off = 0.
var SideDomain = light.Domain{
	Name: "Sidelight",
	Modes: []light.Mode{
		{Name: "Lights off", Code: 0, Features: 0},
		{Name: "Wave", Code: 1, Features: light.SpecificColor | light.RandomColor | light.Speed},
		{Name: "Spectrum", Code: 2, Features: light.Speed},
		{Name: "Static", Code: 3, Features: light.SpecificColor | light.RandomColor},
		{Name: "Breathing", Code: 4, Features: light.SpecificColor | light.RandomColor | light.Speed},
	},
}

// Nuga misc colour index (red, green, blue, yellow, purple, cyan, white) to
// firmware colour_lib index (red, orange, yellow, green, cyan, blue, purple, white).
var nugaToFwColour = [7]byte{0, 3, 5, 2, 6, 4, 7}

func fwToNugaColour(c byte) uint8 {
	if c >= fwSideRainbow {
		return 7
	}
	if c == 1 { // orange has no Nuga slot; closest is red
		return 0
	}
	for i, v := range nugaToFwColour {
		if v == c {
			return uint8(i)
		}
	}
	return 0
}

func modeToFw(code byte) byte {
	if code == 0 || code > 4 {
		return fwSideOff
	}
	return code - 1
}

func modeFromFw(m byte) byte {
	if m >= fwSideOff {
		return 0
	}
	return m + 1
}

// side brightness: Nuga 0..4 <-> firmware 0..5
func sideLevelToFw(l uint8) byte   { return byte(math.Round(float64(l) * 5 / 4)) }
func sideLevelFromFw(v byte) uint8 { return uint8(math.Round(float64(v) * 4 / 5)) }

// SidelightDomain returns side light modes.
func (f *Feature) SidelightDomain() light.Domain { return SideDomain }

// HasSidelight reports whether firmware exposes side light control.
func (f *Feature) HasSidelight() bool { return f.side }

// ProbeSidelight checks whether firmware handles side light values.
func ProbeSidelight(dev Commander) bool {
	_, err := dev.Command([]byte{cmdCustomGetValue, channelCustom, sideMode}, 3)
	return err == nil
}

func (f *Feature) getSide() (light.MiscEffect, error) {
	vals := make([]byte, 4)
	for i, id := range []byte{sideMode, sideBrightness, sideSpeed, sideColour} {
		resp, err := f.dev.Command([]byte{cmdCustomGetValue, channelCustom, id}, 3)
		if err != nil {
			return light.MiscEffect{}, err
		}
		vals[i] = resp[3]
	}
	mode := SideDomain.Find(modeFromFw(vals[0]))
	e := light.MiscEffect{
		Mode: mode,
		EffectParams: light.EffectParams{
			Brightness: sideLevelFromFw(vals[1]),
			Speed:      vals[2],
			Color:      fwToNugaColour(vals[3]),
		},
	}
	f.lastSide = &e
	return e, nil
}

func (f *Feature) setSide(e light.MiscEffect) error {
	if f.lastSide != nil && sameSide(*f.lastSide, e) {
		return nil
	}
	code := byte(0)
	if e.Mode != nil {
		code = e.Mode.Code
	}
	colour := byte(fwSideRainbow)
	if e.Color < 7 {
		colour = nugaToFwColour[e.Color]
	}
	speed := e.Speed
	if speed > 4 {
		speed = 4
	}
	writes := [][2]byte{{sideMode, modeToFw(code)}}
	if code != 0 {
		writes = append(writes,
			[2]byte{sideBrightness, sideLevelToFw(e.Brightness)},
			[2]byte{sideSpeed, speed},
		)
		if e.Mode.Features.IsAnySet(light.SpecificColor, light.RandomColor) {
			writes = append(writes, [2]byte{sideColour, colour})
		}
	}
	for _, w := range writes {
		if _, err := f.dev.Command([]byte{cmdCustomSetValue, channelCustom, w[0], w[1]}, 3); err != nil {
			return err
		}
	}
	if _, err := f.dev.Command([]byte{cmdCustomSave, channelCustom}, 2); err != nil {
		return err
	}
	saved := e
	f.lastSide = &saved
	return nil
}

func sameSide(a, b light.MiscEffect) bool {
	if a.Mode == nil || b.Mode == nil {
		return a.Mode == b.Mode
	}
	return a.Mode.Code == b.Mode.Code && a.EffectParams == b.EffectParams
}

// Status is extra keyboard status exposed by patched firmware.
type Status struct {
	Battery  uint8 `json:"battery"`
	Charging bool  `json:"charging"`
	// Link: 0 2.4G, 1-3 Bluetooth, 4 USB (NuPhy LINK_* values)
	Link uint8 `json:"link"`
	// OS: "mac" or "win" from the physical switch
	OS string `json:"os"`
}

// GetStatus reads battery, link and OS switch (patched firmware only).
func (f *Feature) GetStatus() (*Status, error) {
	bat, err := f.dev.Command([]byte{cmdCustomGetValue, channelCustom, sideValueBattery}, 3)
	if err != nil {
		return nil, err
	}
	link, err := f.dev.Command([]byte{cmdCustomGetValue, channelCustom, sideValueLink}, 3)
	if err != nil {
		return nil, err
	}
	osv, err := f.dev.Command([]byte{cmdCustomGetValue, channelCustom, sideValueOS}, 3)
	if err != nil {
		return nil, err
	}
	s := &Status{Battery: bat[3], Charging: bat[4]&0x01 != 0, Link: link[3], OS: "win"}
	if osv[3] == 0xA2 {
		s.OS = "mac"
	}
	return s, nil
}
