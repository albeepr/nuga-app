package qmklight

import (
	"testing"

	"github.com/mishamyrt/nuga-lib/hid"

	"github.com/mishamyrt/nuga-lib/features/light"
)

// fakeKeyboard emulates QMK via.c handling of the RGB matrix channel.
type fakeKeyboard struct {
	enabled          bool
	mode, val, speed byte
	hue, sat         byte
	saves            int

	patched                      bool // firmware has via_side.c
	sideMode, sideLight, sideSpd byte
	sideColour, sideSaves        byte
}

func (k *fakeKeyboard) Command(p []byte, _ int) ([]byte, error) {
	resp := make([]byte, 32)
	copy(resp, p)
	if len(p) > 1 && p[1] == channelCustom {
		if !k.patched {
			resp[0] = 0xFF
			return resp, hid.ErrVIAUnhandled
		}
		vals := map[byte]*byte{sideMode: &k.sideMode, sideBrightness: &k.sideLight, sideSpeed: &k.sideSpd, sideColour: &k.sideColour}
		switch p[0] {
		case cmdCustomSetValue:
			*vals[p[2]] = p[3]
		case cmdCustomGetValue:
			if v, ok := vals[p[2]]; ok {
				resp[3] = *v
			}
		case cmdCustomSave:
			k.sideSaves++
		}
		return resp, nil
	}
	switch p[0] {
	case cmdCustomSetValue:
		switch p[2] {
		case valueBrightness:
			k.val = p[3]
		case valueEffect:
			if p[3] == 0 {
				k.enabled = false
			} else {
				k.enabled, k.mode = true, p[3]
			}
		case valueSpeed:
			k.speed = p[3]
		case valueColor:
			k.hue, k.sat = p[3], p[4]
		}
	case cmdCustomGetValue:
		switch p[2] {
		case valueBrightness:
			resp[3] = k.val
		case valueEffect:
			if k.enabled {
				resp[3] = k.mode
			}
		case valueSpeed:
			resp[3] = k.speed
		case valueColor:
			resp[3], resp[4] = k.hue, k.sat
		}
	case cmdCustomSave:
		k.saves++
	}
	return resp, nil
}

func TestRoundTrip(t *testing.T) {
	kb := &fakeKeyboard{enabled: true, mode: 1, val: 255, speed: 127, hue: 85, sat: 255}
	f := New(kb, "")
	e, err := f.GetEffects()
	if err != nil {
		t.Fatal(err)
	}
	p := e.Backlight.CurrentParams()
	if e.Backlight.Mode.Code != 1 || p.Brightness != 4 || p.Speed != 2 || p.Color != 1 {
		t.Fatalf("unexpected state: %+v %+v", e.Backlight.Mode, p)
	}
	if e.Halo.Mode == nil || e.Sidelight.Mode == nil {
		t.Fatal("misc modes must be non-nil")
	}

	e.Backlight.Mode = Domain.Find(4) // breathing
	cp := e.Backlight.CurrentParams()
	cp.Color, cp.Brightness, cp.Speed = 2, 1, 4 // blue
	if err := f.SetEffects(e); err != nil {
		t.Fatal(err)
	}
	if kb.mode != 4 || kb.val != 64 || kb.speed != 255 || kb.hue != 171 || kb.sat != 255 || kb.saves != 1 {
		t.Fatalf("unexpected keyboard: %+v", kb)
	}

	e.Backlight.Mode = &light.BacklightOff
	if err := f.SetEffects(e); err != nil {
		t.Fatal(err)
	}
	e, _ = f.GetEffects()
	if e.Backlight.Mode.Code != 0 || e.Backlight.CurrentParams() != nil {
		t.Fatal("expected lights off")
	}
}

func TestNearestColorWhiteAndCustom(t *testing.T) {
	kb := &fakeKeyboard{enabled: true, mode: 1, val: 200, hue: 30, sat: 0}
	f := New(kb, "")
	e, _ := f.GetEffects()
	if e.Backlight.CurrentParams().Color != 6 {
		t.Fatalf("expected white, got %d", e.Backlight.CurrentParams().Color)
	}
	orange := light.RGB{R: 255, G: 128, B: 0}
	if err := f.SetPaletteColor(1, 3, orange); err != nil {
		t.Fatal(err)
	}
	e.Backlight.CurrentParams().Color = 3
	_ = f.SetEffects(e)
	e, _ = f.GetEffects()
	if e.Backlight.CurrentParams().Color != 3 {
		t.Fatalf("expected custom orange slot, got %d (hue %d)", e.Backlight.CurrentParams().Color, kb.hue)
	}
}

func TestSidelight(t *testing.T) {
	stock := New(&fakeKeyboard{enabled: true, mode: 1}, "")
	if stock.HasSidelight() {
		t.Fatal("stock firmware must not report side light")
	}
	e, err := stock.GetEffects()
	if err != nil || e.Sidelight.Mode == nil || e.Sidelight.Mode.Code != 0 {
		t.Fatal("stock firmware: side light off placeholder expected")
	}

	kb := &fakeKeyboard{enabled: true, mode: 1, patched: true, sideMode: 3, sideLight: 5, sideSpd: 2, sideColour: 8}
	f := New(kb, "")
	if !f.HasSidelight() {
		t.Fatal("patched firmware must report side light")
	}
	e, err = f.GetEffects()
	if err != nil {
		t.Fatal(err)
	}
	if e.Sidelight.Mode.Name != "Breathing" || e.Sidelight.Brightness != 4 || e.Sidelight.Color != 7 {
		t.Fatalf("unexpected side state %+v %+v", e.Sidelight.Mode, e.Sidelight.EffectParams)
	}
	// Change only backlight: side values must not be rewritten.
	e.Backlight.CurrentParams().Brightness = 2
	_ = f.SetEffects(e)
	if kb.sideSaves != 0 {
		t.Error("unchanged side light must not be saved again")
	}
	e.Sidelight.Mode = SideDomain.Find(3) // static
	e.Sidelight.Color = 2                 // blue
	e.Sidelight.Brightness = 1
	if err := f.SetEffects(e); err != nil {
		t.Fatal(err)
	}
	if kb.sideMode != 2 || kb.sideColour != 5 || kb.sideLight != 1 || kb.sideSaves != 1 {
		t.Fatalf("unexpected keyboard side state %+v", kb)
	}
	e.Sidelight.Mode = SideDomain.Find(0)
	_ = f.SetEffects(e)
	if kb.sideMode != fwSideOff {
		t.Error("off must map to firmware SIDE_OFF")
	}
}
