package qmkstate

import (
	"encoding/json"
	"testing"

	"github.com/mishamyrt/nuga-lib/features/qmkkeys"
	"github.com/mishamyrt/nuga-lib/features/qmklight"
	"github.com/mishamyrt/nuga-lib/features/via"
)

// fakeKeyboard emulates a patched Air96 V2 well enough for a backup round trip.
type fakeKeyboard struct {
	keymap [5 * 6 * 21]uint16
	macros [1024]byte
	custom map[[2]byte][]byte
}

func newFake() *fakeKeyboard {
	k := &fakeKeyboard{custom: map[[2]byte][]byte{}}
	for i := range k.keymap {
		k.keymap[i] = uint16(i % 200)
	}
	return k
}

func (k *fakeKeyboard) Command(p []byte, _ int) ([]byte, error) {
	r := make([]byte, 32)
	copy(r, p)
	switch p[0] {
	case via.CmdGetLayerCount:
		r[1] = 5
	case via.CmdKeymapGetBuffer:
		off, n := int(p[1])<<8|int(p[2]), int(p[3])
		for i := 0; i < n; i++ {
			c := k.keymap[(off+i)/2]
			r[4+i] = byte(c >> (8 * (1 - (off+i)%2)))
		}
	case via.CmdSetKeycode:
		k.keymap[(int(p[1])*6+int(p[2]))*21+int(p[3])] = uint16(p[4])<<8 | uint16(p[5])
	case via.CmdMacroGetCount:
		r[1] = 16
	case via.CmdMacroGetBufferSize:
		r[1], r[2] = 4, 0
	case via.CmdMacroGetBuffer:
		off, n := int(p[1])<<8|int(p[2]), int(p[3])
		copy(r[4:4+n], k.macros[off:off+n])
	case via.CmdMacroSetBuffer:
		off, n := int(p[1])<<8|int(p[2]), int(p[3])
		copy(k.macros[off:off+n], p[4:4+n])
	case via.CmdCustomGetValue:
		copy(r[3:], k.custom[[2]byte{p[1], p[2]}])
	case via.CmdCustomSetValue:
		k.custom[[2]byte{p[1], p[2]}] = append([]byte(nil), p[3:]...)
	}
	return r, nil
}

func TestBackupRestoreRoundTrip(t *testing.T) {
	src := newFake()
	src.macros[0], src.macros[1], src.macros[2] = 1, 1, 0x04 // tap A
	src.custom[[2]byte{3, 2}] = []byte{7}                   // backlight effect
	src.custom[[2]byte{3, 4}] = []byte{100, 200}            // hue/sat
	src.custom[[2]byte{0, 2}] = []byte{3}                   // side mode breath
	l := qmklight.New(src, "")
	k := qmkkeys.NewAir96V2(via.New(src))
	state, err := Collect(l, k)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	var loaded State
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}

	dst := newFake()
	for i := range dst.keymap {
		dst.keymap[i] = 0
	}
	l2 := qmklight.New(dst, "")
	k2 := qmkkeys.NewAir96V2(via.New(dst))
	if err := Restore(l2, k2, &loaded); err != nil {
		t.Fatal(err)
	}
	if dst.keymap != src.keymap {
		t.Error("keymap differs after restore")
	}
	if dst.macros != src.macros {
		t.Error("macros differ after restore")
	}
	if dst.custom[[2]byte{3, 2}][0] != 7 || dst.custom[[2]byte{3, 4}][1] != 200 || dst.custom[[2]byte{0, 2}][0] != 3 {
		t.Errorf("lights differ after restore: %v", dst.custom)
	}
}
