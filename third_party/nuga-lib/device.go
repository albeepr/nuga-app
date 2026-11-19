// Package nuga is the entrypoint for NuPhy keyboard control library
package nuga

import (
	"errors"

	"github.com/mishamyrt/nuga-lib/device"
	"github.com/mishamyrt/nuga-lib/features"
	"github.com/mishamyrt/nuga-lib/features/qmkkeys"
	"github.com/mishamyrt/nuga-lib/features/qmklight"
	"github.com/mishamyrt/nuga-lib/features/qmkstate"
	"github.com/mishamyrt/nuga-lib/hid"
)

// ErrNotSupported is returned when you try to open a keyboard that is not supported by the application
var ErrNotSupported = errors.New("device is not supported")

// Device represents keyboard with its controls
type Device struct {
	Name         device.Model
	Path         string
	Firmware     string
	Capabilities *Capability
	Features     *features.Features
	Handle       hid.Handler
}

// Close connection with hid device
func (d *Device) Close() error {
	if d.Handle == nil {
		return nil
	}
	return d.Handle.Close()
}

// IsVIA returns true when the device is a QMK/VIA keyboard
func (d *Device) IsVIA() bool {
	return device.IsVIA(d.Name)
}

// Open real keyboard USB handle.
// NuPhy proprietary-protocol keyboards (Halo series) are tried first,
// then QMK/VIA keyboards (Air V2 series).
func Open() (*Device, error) {
	handle, err := hid.Open()
	if err != nil {
		viaDev, viaErr := openVIA()
		if viaErr == nil {
			return viaDev, nil
		}
		return nil, err
	}
	if !device.IsSupported(handle.Info.Model) {
		return nil, ErrNotSupported
	}
	model := device.Model(handle.Info.Model)
	capabilities := GetCapabilities(model)
	featuresRepo := features.New(handle, model)
	return &Device{
		Name:         model,
		Path:         handle.Info.Path,
		Firmware:     handle.Info.Firmware,
		Features:     featuresRepo,
		Capabilities: capabilities,
		Handle:       handle,
	}, nil
}

func openVIA() (*Device, error) {
	handle, err := hid.OpenVIA()
	if err != nil {
		return nil, err
	}
	model := device.Model(handle.Info.Model)
	feats := features.NewVIA(handle, model)
	caps := *GetCapabilities(model)
	if l, ok := feats.Light.(*qmklight.Feature); ok && l.HasSidelight() {
		caps |= SidelightCapability
	}
	return &Device{
		Name:         model,
		Path:         handle.Info.Path,
		Firmware:     handle.Info.Firmware,
		Features:     feats,
		Capabilities: &caps,
		Handle:       handle,
	}, nil
}

func (d *Device) viaFeatures() (*qmklight.Feature, *qmkkeys.Feature, error) {
	l, lok := d.Features.Light.(*qmklight.Feature)
	k, kok := d.Features.Keys.(*qmkkeys.Feature)
	if !lok || !kok {
		return nil, nil, ErrNotSupported
	}
	return l, k, nil
}

// SaveVIAState collects full state of a QMK/VIA keyboard.
func (d *Device) SaveVIAState() (*qmkstate.State, error) {
	l, k, err := d.viaFeatures()
	if err != nil {
		return nil, err
	}
	return qmkstate.Collect(l, k)
}

// RestoreVIAState writes full state to a QMK/VIA keyboard.
func (d *Device) RestoreVIAState(s *qmkstate.State) error {
	l, k, err := d.viaFeatures()
	if err != nil {
		return err
	}
	return qmkstate.Restore(l, k, s)
}

// ResetVIADefaults restores factory keymap and clears macros.
func (d *Device) ResetVIADefaults() error {
	_, k, err := d.viaFeatures()
	if err != nil {
		return err
	}
	return k.ResetToDefaults()
}

// VIAStatus returns battery/link/OS switch status (patched firmware only).
func (d *Device) VIAStatus() (*qmklight.Status, error) {
	l, _, err := d.viaFeatures()
	if err != nil {
		return nil, err
	}
	if !l.HasSidelight() {
		return nil, ErrNotSupported
	}
	return l.GetStatus()
}
