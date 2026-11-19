package dto

import "github.com/mishamyrt/nuga-lib"

// Supports represents keyboard model capabilities
type Supports struct {
	Backlight bool `json:"backlight"`
	Sidelight bool `json:"sidelight"`
	Halolight bool `json:"halolight"`
	Keys      bool `json:"keys"`
	// State is true when the keyboard supports saving and restoring its state
	State bool `json:"state"`
}

// SupportedCapabilities returns keyboard model capabilities
func SupportedCapabilities(c *nuga.Capability) *Supports {
	return &Supports{
		Backlight: c.Has(nuga.BacklightCapability),
		Sidelight: c.Has(nuga.SidelightCapability),
		Halolight: c.Has(nuga.HalolightCapability),
		Keys:      c.Has(nuga.KeysCapability),
		State:     true,
	}
}

// KeyboardStatus represents live keyboard status (QMK keyboards with patched firmware)
type KeyboardStatus struct {
	Battery    uint8  `json:"battery"`
	Charging   bool   `json:"charging"`
	Connection string `json:"connection"`
	OS         string `json:"os"`
}
