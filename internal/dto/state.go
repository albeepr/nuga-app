package dto

import (
	"github.com/mishamyrt/nuga-lib/dump"
	"github.com/mishamyrt/nuga-lib/features/qmkstate"
)

// StateFile represents keyboard state file
type StateFile struct {
	Version     string   `json:"version"`
	MacroTitles []string `json:"macro_titles"`
	// VIA holds state of QMK/VIA keyboards (Air96 V2)
	VIA *qmkstate.State `json:"via,omitempty"`
	*dump.State
}
