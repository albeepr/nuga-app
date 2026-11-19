// Package qmkkeys implements key remapping, macros and state backup for
// NuPhy keyboards running QMK/VIA firmware (Air96 V2).
package qmkkeys

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mishamyrt/nuga-lib/features/keys/layout"
)

// QMK keycode ranges (quantum/keycodes.h in NuPhy's QMK fork).
const (
	kcNo          = 0x0000
	kcTransparent = 0x0001
	qkMods        = 0x0100
	qkModsMax     = 0x1FFF
	qkMomentary   = 0x5220
	qkMacro       = 0x7700
	qkMacroMax    = 0x771F
	qkKB          = 0x7E00
)

// Extra key groups for QMK keyboards.
const (
	GroupNuPhy     = "NuPhy"
	GroupSideLight = "Side light"
	GroupLayers    = "Layers"
	GroupOther     = "Other"
)

// Extra key names, valid only for QMK keyboards.
const (
	KeyTransparent  layout.KeyName = "qmk_transparent"
	KeySideLayer    layout.KeyName = "qmk_side_layer"
	KeyMissionCtl   layout.KeyName = "mission_control"
	KeyLaunchpad    layout.KeyName = "launchpad"
	KeyMediaStop    layout.KeyName = "media_stop"
	KeyBacklightTog layout.KeyName = "backlight_toggle"
	KeyBacklightSpI layout.KeyName = "backlight_speed_up"
	KeyBacklightSpD layout.KeyName = "backlight_speed_down"
	unknownPrefix                  = "qmk_0x"
)

type extraKey struct {
	name  layout.KeyName
	title string
	group string
	code  uint16
}

// nuphyKey returns NuPhy custom keycode (enum custom_keycodes in ansi.h).
func nuphyKey(index uint16) uint16 { return qkKB + index }

// extraKeys are QMK/NuPhy-only actions offered in the key picker.
var extraKeys = []extraKey{
	{KeyTransparent, "Transparent (use lower layer)", GroupLayers, kcTransparent},
	{KeySideLayer, "Side light layer (hold)", GroupLayers, qkMomentary | 4},
	{KeyMissionCtl, "Mission Control", GroupMultimedia(), 0xC1},
	{KeyLaunchpad, "Launchpad", GroupMultimedia(), 0xC2},
	{KeyMediaStop, "Stop", GroupMultimedia(), 0xAD},
	{KeyBacklightTog, "Backlight Toggle", layout.GroupBacklight, 0x7820},
	{KeyBacklightSpI, "Backlight Speed Up", layout.GroupBacklight, 0x7829},
	{KeyBacklightSpD, "Backlight Speed Down", layout.GroupBacklight, 0x782A},
	{"nuphy_link_usb", "Connect: USB", GroupNuPhy, nuphyKey(1)},
	{"nuphy_link_rf", "Connect: 2.4G", GroupNuPhy, nuphyKey(2)},
	{"nuphy_link_ble1", "Connect: Bluetooth 1", GroupNuPhy, nuphyKey(3)},
	{"nuphy_link_ble2", "Connect: Bluetooth 2", GroupNuPhy, nuphyKey(4)},
	{"nuphy_link_ble3", "Connect: Bluetooth 3", GroupNuPhy, nuphyKey(5)},
	{"nuphy_mac_task", "Mac Mission Control", GroupNuPhy, nuphyKey(6)},
	{"nuphy_mac_console", "Mac Console", GroupNuPhy, nuphyKey(9)},
	{"nuphy_mac_dnd", "Mac Do Not Disturb", GroupNuPhy, nuphyKey(10)},
	{"nuphy_mac_print_whole", "Mac Screenshot (whole screen)", GroupNuPhy, nuphyKey(11)},
	{"nuphy_dev_reset", "Reset Wireless Pairings (hold)", GroupNuPhy, nuphyKey(19)},
	{"nuphy_sleep_mode", "Toggle Auto Sleep", GroupNuPhy, nuphyKey(20)},
	{"nuphy_bat_show", "Show Battery Level", GroupNuPhy, nuphyKey(21)},
	{"nuphy_rgb_test", "RGB Test (hold 3s)", GroupNuPhy, nuphyKey(22)},
	{"side_brightness_up", "Side Light Brightness Up", GroupSideLight, nuphyKey(13)},
	{"side_brightness_down", "Side Light Brightness Down", GroupSideLight, nuphyKey(14)},
	{"side_mode_next", "Side Light Next Mode", GroupSideLight, nuphyKey(15)},
	{"side_color_next", "Side Light Next Color", GroupSideLight, nuphyKey(16)},
	{"side_speed_up", "Side Light Faster", GroupSideLight, nuphyKey(17)},
	{"side_speed_down", "Side Light Slower", GroupSideLight, nuphyKey(18)},
}

// GroupMultimedia returns multimedia group name.
func GroupMultimedia() string { return layout.GroupMultimedia }

// nugaToQMK maps Nuga key names that are not plain HID usages.
var nugaToQMK = map[layout.KeyName]uint16{
	layout.KeyMute:                    0xA8,
	layout.KeyVolumeUp:                0xA9,
	layout.KeyVolumeDown:              0xAA,
	layout.KeyForward:                 0xAB,
	layout.KeyRewind:                  0xAC,
	layout.KeyPlayPause:               0xAE,
	layout.KeyBrightnessUp:            0xBD,
	layout.KeyBrightnessDown:          0xBE,
	layout.KeySpotlight:               nuphyKey(7),
	layout.KeyVoiceCommand:            nuphyKey(8),
	layout.KeyScreenshot:              nuphyKey(12),
	layout.KeyBacklightEffectNext:     0x7821,
	layout.KeyBacklightEffectPrevious: 0x7822,
	layout.KeyBacklightColorNext:      0x7823,
	layout.KeyBacklightColorPrevious:  0x7824,
	layout.KeyBacklightBrightnessUp:   0x7827,
	layout.KeyBacklightBrightnessDown: 0x7828,
}

var (
	nameToCode = map[layout.KeyName]uint16{}
	codeToName = map[uint16]layout.KeyName{}
)

func init() {
	for name, desc := range layout.Keys {
		if name == layout.KeyNone || name == layout.KeyFn || name == layout.KeyLMetaAlias {
			continue
		}
		kind := desc.Code & 0xFF
		if kind == 0x00 || kind == 0x06 {
			// Regular keys and modifiers: Nuga stores the HID usage in the top byte.
			register(name, uint16(desc.Code>>24))
		}
	}
	for name, code := range nugaToQMK {
		register(name, code)
	}
	for _, k := range extraKeys {
		register(k.name, k.code)
	}
}

func register(name layout.KeyName, code uint16) {
	nameToCode[name] = code
	if _, exists := codeToName[code]; !exists {
		codeToName[code] = name
	}
}

// isBasic returns true for 8-bit QMK keycodes that can carry modifiers and
// can be used inside macros.
func isBasic(code uint16) bool {
	return code >= 0x04 && code <= 0xFF
}

// UnknownName returns key name used to preserve a QMK code Nuga has no name for.
func UnknownName(code uint16) layout.KeyName {
	return layout.KeyName(fmt.Sprintf("%s%04X", unknownPrefix, code))
}

func parseUnknownName(name layout.KeyName) (uint16, bool) {
	s := string(name)
	if !strings.HasPrefix(s, unknownPrefix) {
		return 0, false
	}
	v, err := strconv.ParseUint(s[len(unknownPrefix):], 16, 16)
	if err != nil {
		return 0, false
	}
	return uint16(v), true
}

// ToAction converts a QMK keycode into a Nuga key action.
// fnLayer is the layer that the Fn key activates for the current OS mode.
func ToAction(code uint16, fnLayer byte) layout.Key {
	switch {
	case code == kcNo:
		return layout.Key{Type: layout.ActionNone}
	case code >= qkMacro && code <= qkMacroMax:
		idx := uint8(code - qkMacro)
		return layout.Key{Type: layout.ActionMacro, MacroIndex: &idx}
	case code == qkMomentary|uint16(fnLayer):
		return keystroke(layout.KeyFn, nil)
	case code >= qkMods && code <= qkModsMax:
		mods := byte(code>>8) & 0x1F
		base := code & 0xFF
		name, ok := codeToName[base]
		// Right-hand modifier variants (bit 0x10) have no Nuga equivalent.
		if ok && mods&0x10 == 0 {
			return keystroke(name, &layout.Modifiers{
				Ctrl:  mods&0x01 != 0,
				Shift: mods&0x02 != 0,
				Alt:   mods&0x04 != 0,
				Meta:  mods&0x08 != 0,
			})
		}
	default:
		if name, ok := codeToName[code]; ok {
			return keystroke(name, nil)
		}
	}
	return keystroke(UnknownName(code), nil)
}

// FromAction converts a Nuga key action into a QMK keycode.
func FromAction(k layout.Key, fnLayer byte) (uint16, error) {
	switch k.Type {
	case layout.ActionNone, "":
		return kcNo, nil
	case layout.ActionMacro:
		if k.MacroIndex == nil || *k.MacroIndex > qkMacroMax-qkMacro {
			return 0, fmt.Errorf("invalid macro index")
		}
		return qkMacro + uint16(*k.MacroIndex), nil
	}
	if k.Keystroke == nil {
		return kcNo, nil
	}
	name := k.Keystroke.Name
	var code uint16
	if name == layout.KeyFn {
		code = qkMomentary | uint16(fnLayer)
	} else if c, ok := parseUnknownName(name); ok {
		code = c
	} else if c, ok := nameToCode[name]; ok {
		code = c
	} else if name == layout.KeyNone {
		return kcNo, nil
	} else {
		return 0, fmt.Errorf("key %q is not available on this keyboard", name)
	}
	if m := k.Keystroke.Modifiers; m != nil && isBasic(code) {
		var mods uint16
		if m.Ctrl {
			mods |= 0x01
		}
		if m.Shift {
			mods |= 0x02
		}
		if m.Alt {
			mods |= 0x04
		}
		if m.Meta {
			mods |= 0x08
		}
		code |= mods << 8
	}
	return code, nil
}

// MacroKeyCode returns 8-bit keycode usable in a VIA macro.
func MacroKeyCode(name layout.KeyName) (byte, bool) {
	code, ok := nameToCode[name]
	if !ok || !isBasic(code) {
		return 0, false
	}
	return byte(code), true
}

// MacroKeyName returns Nuga key name for an 8-bit macro keycode.
func MacroKeyName(code byte) layout.KeyName {
	if name, ok := codeToName[uint16(code)]; ok {
		return name
	}
	return UnknownName(uint16(code))
}

func keystroke(name layout.KeyName, mods *layout.Modifiers) layout.Key {
	if mods != nil && !mods.Ctrl && !mods.Shift && !mods.Alt && !mods.Meta {
		mods = nil
	}
	return layout.Key{
		Type:      layout.ActionKeystroke,
		Keystroke: &layout.KeystrokeParams{Name: name, Modifiers: mods},
	}
}

// KeyDescription is an entry for the key picker.
type KeyDescription struct {
	Name  layout.KeyName
	Title string
	Group string
}

// Descriptions returns all key actions supported on QMK keyboards,
// plus entries for unknown codes found on the keyboard.
func Descriptions(unknown []uint16) []KeyDescription {
	out := make([]KeyDescription, 0, len(nameToCode)+len(unknown))
	seen := map[layout.KeyName]bool{}
	for name := range nameToCode {
		if desc, ok := layout.Keys[name]; ok {
			out = append(out, KeyDescription{name, desc.Title, desc.Group})
			seen[name] = true
		}
	}
	for _, k := range extraKeys {
		if !seen[k.name] {
			out = append(out, KeyDescription{k.name, k.title, k.group})
			seen[k.name] = true
		}
	}
	out = append(out, KeyDescription{layout.KeyNone, "None", layout.GroupSpecial})
	out = append(out, KeyDescription{layout.KeyFn, "Function", layout.GroupSystem})
	for _, c := range unknown {
		name := UnknownName(c)
		if !seen[name] {
			out = append(out, KeyDescription{name, fmt.Sprintf("QMK code 0x%04X", c), GroupOther})
			seen[name] = true
		}
	}
	return out
}
