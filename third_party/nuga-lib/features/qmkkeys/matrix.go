package qmkkeys

import "github.com/mishamyrt/nuga-lib/features/keys/layout"

// Position is a key position in the QMK switch matrix.
type Position struct {
	Row, Col byte
}

// Board describes a QMK keyboard's matrix and layer usage.
type Board struct {
	Rows, Cols int
	// Layers used for each OS mode: [base, fn].
	MacLayers [2]byte
	WinLayers [2]byte
	// Keys maps physical key names (as used by Nuga UI layout) to matrix positions.
	Keys map[layout.KeyName]Position
}

// FnPrefix marks the Fn-layer twin of a physical key in key maps.
const FnPrefix = "fn_"

// Air96V2 is NuPhy Air96 V2 ANSI. Matrix taken from NuPhy's VIA definition,
// layers from keyboards/nuphy/air96_v2/ansi/keymaps/via/keymap.c:
// 0 Mac, 1 Mac Fn, 2 Win, 3 Win Fn, 4 side-light function layer.
var Air96V2 = Board{
	Rows:      6,
	Cols:      21,
	MacLayers: [2]byte{0, 1},
	WinLayers: [2]byte{2, 3},
	Keys: map[layout.KeyName]Position{
		// Row 0
		layout.KeyEsc: {0, 0}, layout.KeyF1: {0, 2}, layout.KeyF2: {0, 3}, layout.KeyF3: {0, 4},
		layout.KeyF4: {0, 5}, layout.KeyF5: {0, 6}, layout.KeyF6: {0, 7}, layout.KeyF7: {0, 8},
		layout.KeyF8: {0, 9}, layout.KeyF9: {0, 10}, layout.KeyF10: {0, 11}, layout.KeyF11: {0, 12},
		layout.KeyF12: {0, 13}, layout.KeyScreenshot: {0, 14}, layout.KeyDel: {2, 14},
		layout.KeyHome: {1, 15}, layout.KeyEnd: {2, 15}, layout.KeyPgUp: {1, 16}, layout.KeyPgDn: {2, 16},
		// Row 1
		layout.KeyGrave: {1, 0}, layout.KeyNum1: {1, 1}, layout.KeyNum2: {1, 2}, layout.KeyNum3: {1, 3},
		layout.KeyNum4: {1, 4}, layout.KeyNum5: {1, 5}, layout.KeyNum6: {1, 6}, layout.KeyNum7: {1, 7},
		layout.KeyNum8: {1, 8}, layout.KeyNum9: {1, 9}, layout.KeyNum0: {1, 10}, layout.KeyMinus: {1, 11},
		layout.KeyEqual: {1, 12}, layout.KeyBackspace: {1, 13}, layout.NumLock: {1, 17},
		layout.NumPadDivide: {1, 18}, layout.NumPadMultiply: {1, 19}, layout.NumPadSubtract: {1, 20},
		// Row 2
		layout.KeyTab: {2, 0}, layout.KeyQ: {2, 1}, layout.KeyW: {2, 2}, layout.KeyE: {2, 3},
		layout.KeyR: {2, 4}, layout.KeyT: {2, 5}, layout.KeyY: {2, 6}, layout.KeyU: {2, 7},
		layout.KeyI: {2, 8}, layout.KeyO: {2, 9}, layout.KeyP: {2, 10}, layout.KeyLBracket: {2, 11},
		layout.KeyRBracket: {2, 12}, layout.KeyBackslash: {2, 13}, layout.NumPad7: {2, 17},
		layout.NumPad8: {2, 18}, layout.NumPad9: {2, 19}, layout.NumPadAdd: {2, 20},
		// Row 3
		layout.KeyCapsLock: {3, 0}, layout.KeyA: {3, 1}, layout.KeyS: {3, 2}, layout.KeyD: {3, 3},
		layout.KeyF: {3, 4}, layout.KeyG: {3, 5}, layout.KeyH: {3, 6}, layout.KeyJ: {3, 7},
		layout.KeyK: {3, 8}, layout.KeyL: {3, 9}, layout.KeySemicolon: {3, 10}, layout.KeyQuote: {3, 11},
		layout.KeyEnter: {3, 13}, layout.NumPad4: {3, 17}, layout.NumPad5: {3, 18}, layout.NumPad6: {3, 19},
		// Row 4
		layout.KeyLShift: {4, 0}, layout.KeyZ: {4, 2}, layout.KeyX: {4, 3}, layout.KeyC: {4, 4},
		layout.KeyV: {4, 5}, layout.KeyB: {4, 6}, layout.KeyN: {4, 7}, layout.KeyM: {4, 8},
		layout.KeyComma: {4, 9}, layout.KeyPeriod: {4, 10}, layout.KeyFwdslash: {4, 11},
		layout.KeyRShift: {4, 13}, layout.KeyUp: {4, 15}, layout.NumPad1: {4, 17},
		layout.NumPad2: {4, 18}, layout.NumPad3: {4, 19}, layout.NumPadEnter: {4, 20},
		// Row 5
		layout.KeyLCtrl: {5, 0}, layout.KeyLAlt: {5, 1}, layout.KeyLMeta: {5, 2}, layout.KeySpace: {5, 6},
		layout.KeyRMeta: {5, 9}, layout.KeyFn: {5, 10}, layout.KeyRCtrl: {5, 13}, layout.KeyLeft: {5, 14},
		layout.KeyDown: {5, 15}, layout.KeyRight: {5, 16}, layout.NumPad0: {5, 17}, layout.NumPadDot: {5, 19},
	},
}
