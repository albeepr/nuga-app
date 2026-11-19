// Package features provides keyboard features repository
package features

import (
	"github.com/mishamyrt/nuga-lib/device"
	"github.com/mishamyrt/nuga-lib/features/keys"
	"github.com/mishamyrt/nuga-lib/features/keys/layout"
	"github.com/mishamyrt/nuga-lib/features/light"
	"github.com/mishamyrt/nuga-lib/features/qmkkeys"
	"github.com/mishamyrt/nuga-lib/features/qmklight"
	"github.com/mishamyrt/nuga-lib/features/via"
	"github.com/mishamyrt/nuga-lib/hid"
)

// Features represents keyboard features repository.
type Features struct {
	Light LightFeature
	Keys  KeysFeature
}

// New creates Features instance with handle
func New(handle *hid.Device, model device.Model) *Features {
	return &Features{
		Light: light.New(handle),
		Keys:  keys.New(handle, &model),
	}
}

// NewVIA creates Features instance for QMK/VIA keyboards
func NewVIA(handle *hid.VIADevice, model device.Model) *Features {
	return &Features{
		Light: qmklight.New(handle, qmklight.DefaultPaletteFile(string(model))),
		Keys:  qmkkeys.NewAir96V2(via.New(handle)),
	}
}

// BacklightDomainProvider is implemented by light features whose
// backlight modes differ from the Halo series.
type BacklightDomainProvider interface {
	BacklightDomain() light.Domain
}

// SidelightDomainProvider is implemented by light features whose
// side light modes differ from the Halo series.
type SidelightDomainProvider interface {
	SidelightDomain() light.Domain
}

// KeyDescriptionsProvider is implemented by keys features that offer
// their own set of key actions.
type KeyDescriptionsProvider interface {
	KeyDescriptions() []qmkkeys.KeyDescription
}

// DefaultKeyMapProvider is implemented by keys features that know
// their factory key map.
type DefaultKeyMapProvider interface {
	DefaultKeyMap(mac bool) *layout.KeyMap
}

// BacklightPalette is implemented by light features that keep the
// per-mode color palette on the host instead of the keyboard.
type BacklightPalette interface {
	GetPalette() [][]light.RGB
	SetPaletteColor(mode, index uint8, color light.RGB) error
}

// LightFeature represents keyboard light feature
type LightFeature interface {
	GetEffects() (*light.Effects, error)
	SetEffects(p *light.Effects) error
	SetBacklightColors(colors *light.BacklightColors) error
	GetBacklightColors() (*light.BacklightColors, error)
}

// KeysFeature represents keyboard keys feature
type KeysFeature interface {
	GetWinCodes() ([]uint32, error)
	GetMacCodes() ([]uint32, error)
	SetWinCodes(keys []uint32) error
	SetMacCodes(keys []uint32) error
	GetWin() (*layout.KeyMap, error)
	GetMac() (*layout.KeyMap, error)
	SetWin(keyMap *layout.KeyMap) error
	SetMac(keyMap *layout.KeyMap) error
	Parse(keys []uint32) (*layout.KeyMap, error)
	GetMacros() (keys.Macros, error)
	SetMacros(macros keys.Macros) error
}
