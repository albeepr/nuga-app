package qmkkeys

import (
	"fmt"
	"sort"
	"sync"

	"github.com/mishamyrt/nuga-lib/features/keys"
	"github.com/mishamyrt/nuga-lib/features/keys/layout"
	"github.com/mishamyrt/nuga-lib/features/via"
)

// Feature implements key remapping and macros over VIA.
type Feature struct {
	client   *via.Client
	board    Board
	defaults []uint16

	lock   sync.Mutex
	layers int
	keymap []uint16 // cached dynamic keymap, nil when stale
}

// New creates QMK keys feature.
func New(client *via.Client, board Board, defaults []uint16) *Feature {
	return &Feature{client: client, board: board, defaults: defaults}
}

// NewAir96V2 creates keys feature for Air96 V2.
func NewAir96V2(client *via.Client) *Feature {
	return New(client, Air96V2, air96V2DefaultKeymap)
}

func (f *Feature) index(layer byte, p Position) int {
	return (int(layer)*f.board.Rows+int(p.Row))*f.board.Cols + int(p.Col)
}

// load reads keymap from keyboard unless cached.
func (f *Feature) load() ([]uint16, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	if f.keymap != nil {
		return f.keymap, nil
	}
	if f.layers == 0 {
		n, err := f.client.LayerCount()
		if err != nil {
			return nil, err
		}
		f.layers = n
	}
	codes, err := f.client.ReadKeymap(f.layers, f.board.Rows, f.board.Cols)
	if err != nil {
		return nil, err
	}
	f.keymap = codes
	return codes, nil
}

// Invalidate drops cached keymap.
func (f *Feature) Invalidate() {
	f.lock.Lock()
	f.keymap = nil
	f.lock.Unlock()
}

func (f *Feature) layersFor(mac bool) [2]byte {
	if mac {
		return f.board.MacLayers
	}
	return f.board.WinLayers
}

// toKeyMap converts raw codes to a Nuga key map with Fn-layer twins.
func (f *Feature) toKeyMap(codes []uint16, mac bool) *layout.KeyMap {
	layers := f.layersFor(mac)
	fnLayer := layers[1]
	km := make(layout.KeyMap, len(f.board.Keys)*2)
	for name, pos := range f.board.Keys {
		km[name] = ToAction(codes[f.index(layers[0], pos)], fnLayer)
		if name != layout.KeyFn {
			km[layout.KeyName(FnPrefix+string(name))] = ToAction(codes[f.index(fnLayer, pos)], fnLayer)
		}
	}
	return &km
}

func (f *Feature) get(mac bool) (*layout.KeyMap, error) {
	codes, err := f.load()
	if err != nil {
		return nil, err
	}
	return f.toKeyMap(codes, mac), nil
}

func (f *Feature) set(km *layout.KeyMap, mac bool) error {
	codes, err := f.load()
	if err != nil {
		return err
	}
	layers := f.layersFor(mac)
	type write struct {
		layer byte
		pos   Position
		code  uint16
	}
	writes := make([]write, 0)
	for name, action := range *km {
		layer, pos, ok := f.resolve(name, layers)
		if !ok {
			continue
		}
		idx := f.index(layer, pos)
		current := codes[idx]
		// Keep codes the user did not change (including ones Nuga can't name).
		if sameAction(ToAction(current, layers[1]), action) {
			continue
		}
		code, err := FromAction(action, layers[1])
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if code != current {
			writes = append(writes, write{layer, pos, code})
		}
	}
	sort.Slice(writes, func(i, j int) bool {
		return f.index(writes[i].layer, writes[i].pos) < f.index(writes[j].layer, writes[j].pos)
	})
	for _, w := range writes {
		if err := f.client.SetKeycode(w.layer, w.pos.Row, w.pos.Col, w.code); err != nil {
			f.Invalidate()
			return err
		}
		f.lock.Lock()
		if f.keymap != nil {
			f.keymap[f.index(w.layer, w.pos)] = w.code
		}
		f.lock.Unlock()
	}
	return nil
}

// resolve maps a key map name to its layer and matrix position.
func (f *Feature) resolve(name layout.KeyName, layers [2]byte) (byte, Position, bool) {
	if pos, ok := f.board.Keys[name]; ok {
		return layers[0], pos, true
	}
	s := string(name)
	if len(s) > len(FnPrefix) && s[:len(FnPrefix)] == FnPrefix {
		if pos, ok := f.board.Keys[layout.KeyName(s[len(FnPrefix):])]; ok {
			return layers[1], pos, true
		}
	}
	return 0, Position{}, false
}

func sameAction(a, b layout.Key) bool {
	if a.Type != b.Type {
		return false
	}
	switch a.Type {
	case layout.ActionMacro:
		return a.MacroIndex != nil && b.MacroIndex != nil && *a.MacroIndex == *b.MacroIndex
	case layout.ActionKeystroke:
		if a.Keystroke == nil || b.Keystroke == nil || a.Keystroke.Name != b.Keystroke.Name {
			return false
		}
		return modsOf(a.Keystroke.Modifiers) == modsOf(b.Keystroke.Modifiers)
	}
	return true
}

func modsOf(m *layout.Modifiers) layout.Modifiers {
	if m == nil {
		return layout.Modifiers{}
	}
	return *m
}

// GetMac returns Mac layer key map.
func (f *Feature) GetMac() (*layout.KeyMap, error) { return f.get(true) }

// GetWin returns Windows layer key map.
func (f *Feature) GetWin() (*layout.KeyMap, error) { return f.get(false) }

// SetMac writes Mac layer key map.
func (f *Feature) SetMac(km *layout.KeyMap) error { return f.set(km, true) }

// SetWin writes Windows layer key map.
func (f *Feature) SetWin(km *layout.KeyMap) error { return f.set(km, false) }

// DefaultKeyMap returns factory key map for OS mode.
func (f *Feature) DefaultKeyMap(mac bool) *layout.KeyMap {
	return f.toKeyMap(f.defaults, mac)
}

func (f *Feature) layerCodes(mac bool) ([]uint32, error) {
	codes, err := f.load()
	if err != nil {
		return nil, err
	}
	layers := f.layersFor(mac)
	size := f.board.Rows * f.board.Cols
	out := make([]uint32, 0, size*2)
	for _, l := range layers {
		for _, c := range codes[int(l)*size : int(l+1)*size] {
			out = append(out, uint32(c))
		}
	}
	return out, nil
}

// GetMacCodes returns raw Mac base+Fn layer codes.
func (f *Feature) GetMacCodes() ([]uint32, error) { return f.layerCodes(true) }

// GetWinCodes returns raw Windows base+Fn layer codes.
func (f *Feature) GetWinCodes() ([]uint32, error) { return f.layerCodes(false) }

// SetMacCodes is not used for QMK keyboards (see RestoreState).
func (f *Feature) SetMacCodes(_ []uint32) error { return fmt.Errorf("use RestoreState") }

// SetWinCodes is not used for QMK keyboards (see RestoreState).
func (f *Feature) SetWinCodes(_ []uint32) error { return fmt.Errorf("use RestoreState") }

// Parse converts raw base+Fn layer codes (as returned by GetMacCodes) to key map.
func (f *Feature) Parse(raw []uint32) (*layout.KeyMap, error) {
	size := f.board.Rows * f.board.Cols
	if len(raw) != size*2 {
		return nil, fmt.Errorf("unexpected key code count %d", len(raw))
	}
	codes := make([]uint16, size*4)
	for i, c := range raw {
		codes[i] = uint16(c)
	}
	// Place both layers as layers 0/1 and decode with that numbering.
	tmp := &Feature{board: f.board}
	tmp.board.MacLayers = [2]byte{0, 1}
	return tmp.toKeyMap(codes, true), nil
}

// KeyDescriptions returns key picker entries, including unknown codes
// currently stored on the keyboard so they keep a visible name.
func (f *Feature) KeyDescriptions() []KeyDescription {
	codes, err := f.load()
	unknown := make([]uint16, 0)
	if err == nil {
		seen := map[uint16]bool{}
		for _, c := range codes {
			action := ToAction(c, f.board.MacLayers[1])
			if action.Type == layout.ActionKeystroke {
				if _, ok := parseUnknownName(action.Keystroke.Name); ok && !seen[c] {
					// MO(win fn) shows as unknown on Mac layers and vice versa;
					// both are the Fn key from the user's point of view.
					if c == qkMomentary|uint16(f.board.WinLayers[1]) {
						continue
					}
					seen[c] = true
					unknown = append(unknown, c)
				}
			}
		}
	}
	return Descriptions(unknown)
}

// GetMacros reads macros from keyboard.
func (f *Feature) GetMacros() (keys.Macros, error) {
	count, err := f.client.MacroCount()
	if err != nil {
		return nil, err
	}
	size, err := f.client.MacroBufferSize()
	if err != nil {
		return nil, err
	}
	buf, err := f.client.ReadMacroBuffer(size)
	if err != nil {
		return nil, err
	}
	return DecodeMacros(buf, count), nil
}

// SetMacros writes macros to keyboard.
func (f *Feature) SetMacros(macros keys.Macros) error {
	count, err := f.client.MacroCount()
	if err != nil {
		return err
	}
	size, err := f.client.MacroBufferSize()
	if err != nil {
		return err
	}
	data, err := EncodeMacros(macros, count)
	if err != nil {
		return err
	}
	return f.client.WriteMacroBuffer(data, size)
}

// ResetToDefaults restores factory keymap and clears macros.
func (f *Feature) ResetToDefaults() error {
	defer f.Invalidate()
	if err := f.client.ResetKeymap(); err != nil {
		return err
	}
	return f.client.ResetMacros()
}

// RawKeymap returns all dynamic keymap layers.
func (f *Feature) RawKeymap() ([]uint16, error) {
	codes, err := f.load()
	if err != nil {
		return nil, err
	}
	return append([]uint16(nil), codes...), nil
}

// RestoreKeymap writes all layers, changing only keys that differ.
func (f *Feature) RestoreKeymap(target []uint16) error {
	codes, err := f.load()
	if err != nil {
		return err
	}
	if len(target) != len(codes) {
		return fmt.Errorf("backup has %d keys, keyboard has %d", len(target), len(codes))
	}
	defer f.Invalidate()
	size := f.board.Rows * f.board.Cols
	for i, code := range target {
		if codes[i] == code {
			continue
		}
		layer, rest := i/size, i%size
		if err := f.client.SetKeycode(byte(layer), byte(rest/f.board.Cols), byte(rest%f.board.Cols), code); err != nil {
			return err
		}
	}
	return nil
}

// RawMacros returns raw macro buffer.
func (f *Feature) RawMacros() ([]byte, error) {
	size, err := f.client.MacroBufferSize()
	if err != nil {
		return nil, err
	}
	return f.client.ReadMacroBuffer(size)
}

// RestoreMacros writes raw macro buffer.
func (f *Feature) RestoreMacros(buf []byte) error {
	size, err := f.client.MacroBufferSize()
	if err != nil {
		return err
	}
	if len(buf) > size {
		buf = buf[:size]
	}
	return f.client.WriteMacroBuffer(buf, size)
}
