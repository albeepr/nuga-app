package qmkkeys

import (
	"testing"

	"github.com/mishamyrt/nuga-lib/features/keys"
	"github.com/mishamyrt/nuga-lib/features/keys/layout"
	"github.com/mishamyrt/nuga-lib/features/via"
)

// fakeVIA emulates QMK via.c dynamic keymap and macro commands.
type fakeVIA struct {
	keymap []uint16
	macros []byte
	writes int
}

func newFake() *fakeVIA {
	return &fakeVIA{
		keymap: append([]uint16(nil), air96V2DefaultKeymap...),
		macros: make([]byte, 1024),
	}
}

func (k *fakeVIA) Command(p []byte, _ int) ([]byte, error) {
	r := make([]byte, 32)
	copy(r, p)
	switch p[0] {
	case via.CmdGetLayerCount:
		r[1] = 5
	case via.CmdKeymapGetBuffer:
		off, n := int(p[1])<<8|int(p[2]), int(p[3])
		for i := 0; i < n; i++ {
			c := k.keymap[(off+i)/2]
			if (off+i)%2 == 0 {
				r[4+i] = byte(c >> 8)
			} else {
				r[4+i] = byte(c)
			}
		}
	case via.CmdSetKeycode:
		k.keymap[(int(p[1])*6+int(p[2]))*21+int(p[3])] = uint16(p[4])<<8 | uint16(p[5])
		k.writes++
	case via.CmdKeymapReset:
		copy(k.keymap, air96V2DefaultKeymap)
	case via.CmdMacroGetCount:
		r[1] = 16
	case via.CmdMacroGetBufferSize:
		r[1], r[2] = byte(len(k.macros)>>8), byte(len(k.macros))
	case via.CmdMacroGetBuffer:
		off, n := int(p[1])<<8|int(p[2]), int(p[3])
		copy(r[4:4+n], k.macros[off:off+n])
	case via.CmdMacroSetBuffer:
		off, n := int(p[1])<<8|int(p[2]), int(p[3])
		copy(k.macros[off:off+n], p[4:4+n])
	case via.CmdMacroReset:
		for i := range k.macros {
			k.macros[i] = 0
		}
	}
	return r, nil
}

func TestReadFactoryKeymap(t *testing.T) {
	f := NewAir96V2(via.New(newFake()))
	mac, err := f.GetMac()
	if err != nil {
		t.Fatal(err)
	}
	m := *mac
	expect := map[layout.KeyName]layout.KeyName{
		layout.KeyA:      layout.KeyA,
		layout.KeyFn:     layout.KeyFn,
		layout.KeyF8:     layout.KeyPlayPause,
		"fn_f8":          layout.KeyF8,
		layout.KeyF4:     layout.KeySpotlight,
		"fn_lshift":      KeySideLayer,
		layout.KeyEsc:    layout.KeyEsc,
		"fn_esc":         KeyTransparent,
		"fn_num1":        "nuphy_link_ble1",
		layout.KeyLMeta:  layout.KeyLMeta,
		layout.KeyPgDn:   layout.KeyPgDn,
		layout.NumPadDot: layout.NumPadDot,
	}
	for pos, name := range expect {
		got := m[pos]
		if got.Keystroke == nil || got.Keystroke.Name != name {
			t.Errorf("%s: expected %s, got %+v", pos, name, got.Keystroke)
		}
	}
	win, _ := f.GetWin()
	if (*win)[layout.KeyLAlt].Keystroke.Name != layout.KeyLMeta {
		t.Errorf("win layer: lalt position should be GUI, got %s", (*win)[layout.KeyLAlt].Keystroke.Name)
	}
	if (*win)[layout.KeyFn].Keystroke.Name != layout.KeyFn {
		t.Errorf("win fn should be MO(3)")
	}
	// Every key on the factory map must have a name (no "QMK code" leftovers)
	for pos, a := range m {
		if a.Type == layout.ActionKeystroke {
			if _, unknown := parseUnknownName(a.Keystroke.Name); unknown {
				t.Errorf("%s has unnamed code %s", pos, a.Keystroke.Name)
			}
		}
	}
}

func TestRemapPreservesUntouchedKeys(t *testing.T) {
	kb := newFake()
	kb.keymap[(0*6+2)*21+1] = 0x5262 // TG(2) on Q: unknown to Nuga
	f := NewAir96V2(via.New(kb))
	m, _ := f.GetMac()
	km := *m
	if km[layout.KeyQ].Keystroke.Name != UnknownName(0x5262) {
		t.Fatalf("unknown code should round trip by name, got %s", km[layout.KeyQ].Keystroke.Name)
	}
	idx := uint8(2)
	km[layout.KeyCapsLock] = layout.Key{Type: layout.ActionKeystroke, Keystroke: &layout.KeystrokeParams{
		Name: layout.KeyEsc, Modifiers: &layout.Modifiers{Ctrl: true, Shift: true}}}
	km["fn_j"] = layout.Key{Type: layout.ActionMacro, MacroIndex: &idx}
	km[layout.KeyA] = layout.Key{Type: layout.ActionNone}
	if err := f.SetMac(&km); err != nil {
		t.Fatal(err)
	}
	if kb.writes != 3 {
		t.Errorf("expected 3 writes, got %d", kb.writes)
	}
	if got := kb.keymap[(0*6+3)*21+0]; got != 0x0329 {
		t.Errorf("caps -> C(S(esc)) expected 0x0329, got %04X", got)
	}
	if got := kb.keymap[(1*6+3)*21+7]; got != 0x7702 {
		t.Errorf("fn+j -> macro 2 expected 0x7702, got %04X", got)
	}
	if got := kb.keymap[(0*6+2)*21+1]; got != 0x5262 {
		t.Errorf("unknown code on Q must be preserved, got %04X", got)
	}
	if kb.keymap[(2*6+3)*21+0] != 0x39 {
		t.Error("win layer must not change")
	}
	descs := f.KeyDescriptions()
	found := false
	for _, d := range descs {
		if d.Name == UnknownName(0x5262) {
			found = true
		}
	}
	if !found {
		t.Error("unknown code must be listed in key picker")
	}
}

func TestMacrosRoundTrip(t *testing.T) {
	kb := newFake()
	f := NewAir96V2(via.New(kb))
	in := keys.Macros{
		{Repeats: 1, Actions: []keys.MacroAction{
			{Key: layout.KeyLMeta, Event: keys.KeyDown},
			{Key: layout.KeySpace, Event: keys.KeyDown, Delay: 30},
			{Key: layout.KeySpace, Event: keys.KeyUp},
			{Key: layout.KeyLMeta, Event: keys.KeyUp},
		}},
		{Repeats: 1, Actions: []keys.MacroAction{}},
		{Repeats: 3, Actions: []keys.MacroAction{
			{Key: layout.KeyVolumeUp, Event: keys.KeyDown},
			{Key: layout.KeyVolumeUp, Event: keys.KeyUp, Delay: 100},
		}},
	}
	if err := f.SetMacros(in); err != nil {
		t.Fatal(err)
	}
	out, err := f.GetMacros()
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 || len(out[1].Actions) != 0 {
		t.Fatalf("expected 3 macros with empty middle, got %+v", out)
	}
	if out[2].Repeats != 3 || len(out[2].Actions) != 2 || out[2].Actions[1].Delay != 100 {
		t.Errorf("repeat macro: %+v", out[2])
	}
	if out[0].Actions[1].Delay != 30 || out[0].Actions[0].Key != layout.KeyLMeta {
		t.Errorf("macro 1: %+v", out[0])
	}
}

func TestDecodeVIATextMacro(t *testing.T) {
	buf := append([]byte("Hi!"), 0)
	m := DecodeMacros(buf, 4)
	if len(m) != 1 || len(m[0].Actions) != 10 {
		t.Fatalf("got %+v", m)
	}
	if m[0].Actions[0].Key != layout.KeyLShift || m[0].Actions[1].Key != layout.KeyH {
		t.Errorf("uppercase should be shift+h: %+v", m[0].Actions[:3])
	}
}

func TestMacroUnsupportedKey(t *testing.T) {
	_, err := EncodeMacros(keys.Macros{{Actions: []keys.MacroAction{{Key: "side_mode_next", Event: keys.KeyDown}}}}, 16)
	if err == nil {
		t.Error("16-bit keycodes can't be in VIA macros")
	}
}

func TestResetAndRestore(t *testing.T) {
	kb := newFake()
	f := NewAir96V2(via.New(kb))
	backup, _ := f.RawKeymap()
	kb.keymap[5] = 0x0004
	f.Invalidate()
	if err := f.RestoreKeymap(backup); err != nil {
		t.Fatal(err)
	}
	if kb.keymap[5] != air96V2DefaultKeymap[5] {
		t.Error("restore failed")
	}
	kb.keymap[7] = 0x0004
	kb.macros[0] = 'x'
	if err := f.ResetToDefaults(); err != nil {
		t.Fatal(err)
	}
	if kb.keymap[7] != air96V2DefaultKeymap[7] || kb.macros[0] != 0 {
		t.Error("reset failed")
	}
}
