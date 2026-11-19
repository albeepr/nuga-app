// Package qmklight implements backlight control for NuPhy keyboards running
// QMK firmware with VIA support (e.g. Air96 V2). It talks to the
// `id_qmk_rgb_matrix_channel` custom-value channel used by the VIA definition
// file that NuPhy publishes for the keyboard.
package qmklight

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sync"

	"github.com/mishamyrt/nuga-lib/features/light"
)

// VIA protocol constants.
const (
	cmdCustomSetValue = 0x07
	cmdCustomGetValue = 0x08
	cmdCustomSave     = 0x09

	channelRGBMatrix = 3

	valueBrightness = 1
	valueEffect     = 2
	valueSpeed      = 3
	valueColor      = 4
)

// Commander is a VIA command transport.
type Commander interface {
	Command(payload []byte, match int) ([]byte, error)
}

// Feature is QMK RGB matrix light feature.
type Feature struct {
	dev         Commander
	paletteFile string

	lock     sync.Mutex
	palette  [][]light.RGB
	side     bool
	lastSide *light.MiscEffect
}

// New creates QMK light feature. Palette is persisted to paletteFile
// (pass empty string to keep it in memory only).
func New(dev Commander, paletteFile string) *Feature {
	f := &Feature{
		dev:         dev,
		paletteFile: paletteFile,
	}
	f.palette = f.loadPalette()
	f.side = ProbeSidelight(dev)
	return f
}

// DefaultPaletteFile returns per-user palette location for given model.
func DefaultPaletteFile(model string) string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "nuga", model+"-palette.json")
}

// BacklightDomain returns backlight modes of the QMK firmware.
func (f *Feature) BacklightDomain() light.Domain {
	return Domain
}

// GetEffects reads current RGB matrix state.
func (f *Feature) GetEffects() (*light.Effects, error) {
	effect, err := f.get(valueEffect, 1)
	if err != nil {
		return nil, err
	}
	brightness, err := f.get(valueBrightness, 1)
	if err != nil {
		return nil, err
	}
	speed, err := f.get(valueSpeed, 1)
	if err != nil {
		return nil, err
	}
	color, err := f.get(valueColor, 2)
	if err != nil {
		return nil, err
	}

	mode := Domain.Find(effect[0])
	if mode == nil {
		mode = &light.Mode{
			Name:     "Firmware effect",
			Code:     effect[0],
			Features: light.SpecificColor | light.Speed,
		}
	}
	params := light.EffectParams{
		Brightness: toLevel(brightness[0]),
		Speed:      toLevel(speed[0]),
		Color:      f.nearestColor(mode.Code, color[0], color[1]),
	}
	paramsList := make([]light.EffectParams, ModesCount)
	for i := range paramsList {
		paramsList[i] = params
	}
	effects := &light.Effects{
		Backlight: light.BacklightEffect{
			Mode:   mode,
			Params: paramsList,
		},
		Sidelight: light.MiscEffect{Mode: &SideDomain.Modes[0]},
		Halo:      light.MiscEffect{Mode: &light.HalolightOff},
	}
	if f.side {
		side, err := f.getSide()
		if err != nil {
			return nil, err
		}
		effects.Sidelight = side
	}
	return effects, nil
}

// SetEffects writes RGB matrix state and saves it to keyboard EEPROM.
func (f *Feature) SetEffects(p *light.Effects) error {
	if f.side {
		if err := f.setSide(p.Sidelight); err != nil {
			return err
		}
	}
	mode := p.Backlight.Mode
	if mode == nil || mode.Code == 0 {
		if err := f.set(valueEffect, 0); err != nil {
			return err
		}
		return f.save()
	}
	params := p.Backlight.CurrentParams()
	if params == nil {
		return light.ErrLightsOff
	}
	if err := f.set(valueEffect, mode.Code); err != nil {
		return err
	}
	if err := f.set(valueBrightness, fromLevel(params.Brightness)); err != nil {
		return err
	}
	if mode.Features.IsSet(light.Speed) {
		if err := f.set(valueSpeed, fromLevel(params.Speed)); err != nil {
			return err
		}
	}
	if mode.Features.IsSet(light.SpecificColor) && params.Color < light.ModeColorsCount {
		c := f.paletteColor(mode.Code, params.Color)
		h, s := rgbToHueSat(c)
		if err := f.set(valueColor, h, s); err != nil {
			return err
		}
	}
	return f.save()
}

// GetBacklightColors returns palette in Halo-compatible structure.
// Kept for interface compatibility; use GetPalette for QMK devices.
func (f *Feature) GetBacklightColors() (*light.BacklightColors, error) {
	var colors light.BacklightColors
	for i := range colors {
		for j := range colors[i] {
			colors[i][j] = defaultColors[j]
		}
	}
	return &colors, nil
}

// SetBacklightColors is not used for QMK devices.
func (f *Feature) SetBacklightColors(_ *light.BacklightColors) error {
	return nil
}

// GetPalette returns color palette for each mode (index = mode code).
func (f *Feature) GetPalette() [][]light.RGB {
	f.lock.Lock()
	defer f.lock.Unlock()
	out := make([][]light.RGB, len(f.palette))
	for i := range f.palette {
		out[i] = append([]light.RGB(nil), f.palette[i]...)
	}
	return out
}

// SetPaletteColor changes palette color for a mode.
func (f *Feature) SetPaletteColor(mode, index uint8, color light.RGB) error {
	f.lock.Lock()
	if int(mode) >= len(f.palette) || int(index) >= light.ModeColorsCount {
		f.lock.Unlock()
		return light.ErrOutOfRange
	}
	f.palette[mode][index] = color
	f.lock.Unlock()
	return f.storePalette()
}

func (f *Feature) get(id byte, size int) ([]byte, error) {
	resp, err := f.dev.Command([]byte{cmdCustomGetValue, channelRGBMatrix, id}, 3)
	if err != nil {
		return nil, err
	}
	return resp[3 : 3+size], nil
}

func (f *Feature) set(id byte, data ...byte) error {
	payload := append([]byte{cmdCustomSetValue, channelRGBMatrix, id}, data...)
	_, err := f.dev.Command(payload, 3)
	return err
}

func (f *Feature) save() error {
	_, err := f.dev.Command([]byte{cmdCustomSave, channelRGBMatrix}, 2)
	return err
}

func (f *Feature) paletteColor(mode, index uint8) light.RGB {
	f.lock.Lock()
	defer f.lock.Unlock()
	if int(mode) < len(f.palette) {
		return f.palette[mode][index]
	}
	return defaultColors[index]
}

// nearestColor finds the palette entry closest to the hue/sat reported by keyboard.
func (f *Feature) nearestColor(mode, hue, sat uint8) uint8 {
	f.lock.Lock()
	defer f.lock.Unlock()
	colors := defaultColors[:]
	if int(mode) < len(f.palette) {
		colors = f.palette[mode]
	}
	best := uint8(0)
	bestDist := math.MaxFloat64
	for i, c := range colors {
		h, s := rgbToHueSat(c)
		dh := math.Abs(float64(h) - float64(hue))
		if dh > 128 {
			dh = 256 - dh
		}
		// Hue matters less as saturation drops (white has no hue).
		weight := (float64(s) + float64(sat)) / 510
		ds := float64(s) - float64(sat)
		dist := dh*dh*weight + ds*ds
		if dist < bestDist {
			bestDist = dist
			best = uint8(i)
		}
	}
	return best
}

func (f *Feature) loadPalette() [][]light.RGB {
	palette := make([][]light.RGB, ModesCount+1)
	for i := range palette {
		palette[i] = append([]light.RGB(nil), defaultColors[:]...)
	}
	if f.paletteFile == "" {
		return palette
	}
	data, err := os.ReadFile(f.paletteFile)
	if err != nil {
		return palette
	}
	var stored [][]light.RGB
	if json.Unmarshal(data, &stored) != nil {
		return palette
	}
	for i := range stored {
		if i < len(palette) && len(stored[i]) == light.ModeColorsCount {
			palette[i] = stored[i]
		}
	}
	return palette
}

func (f *Feature) storePalette() error {
	if f.paletteFile == "" {
		return nil
	}
	data, err := json.Marshal(f.GetPalette())
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(f.paletteFile), 0o755); err != nil {
		return err
	}
	return os.WriteFile(f.paletteFile, data, 0o644)
}

// toLevel converts QMK 0-255 value to Nuga 0-4 level.
func toLevel(v byte) uint8 {
	return uint8(math.Round(float64(v) * 4 / 255))
}

// fromLevel converts Nuga 0-4 level to QMK 0-255 value.
func fromLevel(l uint8) byte {
	if l > 4 {
		l = 4
	}
	return byte(math.Round(float64(l) * 255 / 4))
}

// rgbToHueSat converts RGB to QMK hue/saturation (both 0-255).
func rgbToHueSat(c light.RGB) (byte, byte) {
	r, g, b := float64(c.R)/255, float64(c.G)/255, float64(c.B)/255
	maxV := math.Max(r, math.Max(g, b))
	minV := math.Min(r, math.Min(g, b))
	delta := maxV - minV
	if maxV == 0 || delta == 0 {
		return 0, 0
	}
	var h float64
	switch maxV {
	case r:
		h = math.Mod((g-b)/delta, 6)
	case g:
		h = (b-r)/delta + 2
	default:
		h = (r-g)/delta + 4
	}
	h *= 60
	if h < 0 {
		h += 360
	}
	s := delta / maxV
	return byte(int(math.Round(h*256/360)) & 0xFF), byte(math.Round(s * 255))
}

// RawState is backlight and side light state in firmware units.
type RawState struct {
	Effect     byte          `json:"effect"`
	Brightness byte          `json:"brightness"`
	Speed      byte          `json:"speed"`
	Hue        byte          `json:"hue"`
	Saturation byte          `json:"saturation"`
	Side       *RawSideState `json:"side,omitempty"`
	Palette    [][]light.RGB `json:"palette"`
}

// RawSideState is side light state in firmware units.
type RawSideState struct {
	Mode       byte `json:"mode"`
	Brightness byte `json:"brightness"`
	Speed      byte `json:"speed"`
	Colour     byte `json:"colour"`
}

// GetRawState reads light state for backup.
func (f *Feature) GetRawState() (*RawState, error) {
	s := &RawState{Palette: f.GetPalette()}
	for _, v := range []struct {
		id  byte
		dst []*byte
	}{
		{valueEffect, []*byte{&s.Effect}},
		{valueBrightness, []*byte{&s.Brightness}},
		{valueSpeed, []*byte{&s.Speed}},
		{valueColor, []*byte{&s.Hue, &s.Saturation}},
	} {
		data, err := f.get(v.id, len(v.dst))
		if err != nil {
			return nil, err
		}
		for i, d := range v.dst {
			*d = data[i]
		}
	}
	if f.side {
		side := &RawSideState{}
		for _, v := range []struct {
			id  byte
			dst *byte
		}{
			{sideMode, &side.Mode}, {sideBrightness, &side.Brightness},
			{sideSpeed, &side.Speed}, {sideColour, &side.Colour},
		} {
			resp, err := f.dev.Command([]byte{cmdCustomGetValue, channelCustom, v.id}, 3)
			if err != nil {
				return nil, err
			}
			*v.dst = resp[3]
		}
		s.Side = side
	}
	return s, nil
}

// SetRawState restores light state from backup.
func (f *Feature) SetRawState(s *RawState) error {
	for i, colors := range s.Palette {
		for j, c := range colors {
			if i < len(f.palette) && j < light.ModeColorsCount {
				f.lock.Lock()
				f.palette[i][j] = c
				f.lock.Unlock()
			}
		}
	}
	_ = f.storePalette()
	for _, w := range [][]byte{
		{valueBrightness, s.Brightness},
		{valueEffect, s.Effect},
		{valueSpeed, s.Speed},
		{valueColor, s.Hue, s.Saturation},
	} {
		if err := f.set(w[0], w[1:]...); err != nil {
			return err
		}
	}
	if err := f.save(); err != nil {
		return err
	}
	if f.side && s.Side != nil {
		for _, w := range [][2]byte{
			{sideMode, s.Side.Mode}, {sideBrightness, s.Side.Brightness},
			{sideSpeed, s.Side.Speed}, {sideColour, s.Side.Colour},
		} {
			if _, err := f.dev.Command([]byte{cmdCustomSetValue, channelCustom, w[0], w[1]}, 3); err != nil {
				return err
			}
		}
		f.lastSide = nil
		if _, err := f.dev.Command([]byte{cmdCustomSave, channelCustom}, 2); err != nil {
			return err
		}
	}
	return nil
}
