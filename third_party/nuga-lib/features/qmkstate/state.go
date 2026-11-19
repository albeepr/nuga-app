// Package qmkstate saves and restores the full state of QMK/VIA NuPhy keyboards.
package qmkstate

import (
	"github.com/mishamyrt/nuga-lib/features/qmkkeys"
	"github.com/mishamyrt/nuga-lib/features/qmklight"
)

// FormatVersion is the backup format version.
const FormatVersion = 1

// State is a full keyboard backup.
type State struct {
	Format int                `json:"format"`
	Keymap []uint16           `json:"keymap"`
	Macros []byte             `json:"macros"`
	Lights *qmklight.RawState `json:"lights"`
}

// Collect reads keyboard state.
func Collect(l *qmklight.Feature, k *qmkkeys.Feature) (*State, error) {
	keymap, err := k.RawKeymap()
	if err != nil {
		return nil, err
	}
	macros, err := k.RawMacros()
	if err != nil {
		return nil, err
	}
	lights, err := l.GetRawState()
	if err != nil {
		return nil, err
	}
	return &State{Format: FormatVersion, Keymap: keymap, Macros: macros, Lights: lights}, nil
}

// Restore writes keyboard state.
func Restore(l *qmklight.Feature, k *qmkkeys.Feature, s *State) error {
	if err := k.RestoreKeymap(s.Keymap); err != nil {
		return err
	}
	if err := k.RestoreMacros(s.Macros); err != nil {
		return err
	}
	if s.Lights != nil {
		return l.SetRawState(s.Lights)
	}
	return nil
}
