package qmkkeys

import (
	"fmt"
	"strconv"

	"github.com/mishamyrt/nuga-lib/features/keys"
	"github.com/mishamyrt/nuga-lib/features/keys/layout"
)

// VIA macro encoding (quantum/send_string/send_string_keycodes.h).
const (
	ssPrefix  = 1
	ssTap     = 1
	ssDown    = 2
	ssUp      = 3
	ssDelay   = 4
	maxDelay  = 9999
	maxRepeat = 100
)

// EncodeMacros converts Nuga macros into VIA macro buffer content
// with exactly `slots` null-terminated entries.
func EncodeMacros(macros keys.Macros, slots int) ([]byte, error) {
	if len(macros) > slots {
		return nil, fmt.Errorf("keyboard supports %d macros, got %d", slots, len(macros))
	}
	buf := make([]byte, 0, 256)
	for i := 0; i < slots; i++ {
		if i < len(macros) {
			data, err := encodeMacro(macros[i])
			if err != nil {
				return nil, fmt.Errorf("macro %d: %w", i+1, err)
			}
			buf = append(buf, data...)
		}
		buf = append(buf, 0)
	}
	return buf, nil
}

func encodeMacro(m keys.Macro) ([]byte, error) {
	repeats := int(m.Repeats)
	if repeats < 1 {
		repeats = 1
	}
	if repeats > maxRepeat {
		repeats = maxRepeat
	}
	one := make([]byte, 0, len(m.Actions)*4)
	for _, a := range m.Actions {
		code, ok := MacroKeyCode(a.Key)
		if !ok {
			return nil, fmt.Errorf("key %q can't be used in a macro on this keyboard", a.Key)
		}
		event := byte(ssDown)
		if a.Event == keys.KeyUp {
			event = ssUp
		}
		one = append(one, ssPrefix, event, code)
		if a.Delay > 0 {
			d := a.Delay
			if d > maxDelay {
				d = maxDelay
			}
			one = append(one, ssPrefix, ssDelay)
			one = append(one, []byte(strconv.Itoa(int(d)))...)
			one = append(one, '|')
		}
	}
	out := make([]byte, 0, len(one)*repeats)
	for i := 0; i < repeats; i++ {
		out = append(out, one...)
	}
	return out, nil
}

// DecodeMacros parses VIA macro buffer. Trailing empty slots are dropped,
// empty slots before a used one are kept so macro indexes stay stable.
func DecodeMacros(buf []byte, slots int) keys.Macros {
	macros := make(keys.Macros, 0, slots)
	start := 0
	for i := 0; i < slots && start <= len(buf); i++ {
		end := start
		for end < len(buf) && buf[end] != 0 {
			end++
		}
		macros = append(macros, decodeMacro(buf[start:end]))
		start = end + 1
	}
	last := -1
	for i, m := range macros {
		if len(m.Actions) > 0 {
			last = i
		}
	}
	return macros[:last+1]
}

func decodeMacro(data []byte) keys.Macro {
	actions := make([]keys.MacroAction, 0)
	add := func(name layout.KeyName, ev keys.MacroEvent) {
		actions = append(actions, keys.MacroAction{Key: name, Event: ev})
	}
	for i := 0; i < len(data); {
		if data[i] == ssPrefix && i+1 < len(data) {
			switch data[i+1] {
			case ssTap, ssDown, ssUp:
				if i+2 >= len(data) {
					i = len(data)
					continue
				}
				name := MacroKeyName(data[i+2])
				switch data[i+1] {
				case ssTap:
					add(name, keys.KeyDown)
					add(name, keys.KeyUp)
				case ssDown:
					add(name, keys.KeyDown)
				case ssUp:
					add(name, keys.KeyUp)
				}
				i += 3
				continue
			case ssDelay:
				j := i + 2
				for j < len(data) && data[j] != '|' {
					j++
				}
				d, _ := strconv.Atoi(string(data[i+2 : j]))
				if len(actions) > 0 {
					actions[len(actions)-1].Delay += uint16(d)
				}
				i = j + 1
				continue
			}
		}
		// Plain text typed by VIA's macro editor.
		if names, shift := asciiKeys(data[i]); len(names) > 0 {
			if shift {
				add(layout.KeyLShift, keys.KeyDown)
			}
			add(names[0], keys.KeyDown)
			add(names[0], keys.KeyUp)
			if shift {
				add(layout.KeyLShift, keys.KeyUp)
			}
		}
		i++
	}
	actions, repeats := foldRepeats(actions)
	return keys.Macro{Actions: actions, Repeats: repeats}
}

// foldRepeats detects a macro written as N copies of the same sequence.
func foldRepeats(a []keys.MacroAction) ([]keys.MacroAction, uint16) {
	n := len(a)
	for period := 1; period <= n/2; period++ {
		if n%period != 0 {
			continue
		}
		match := true
		for i := period; i < n && match; i++ {
			match = a[i] == a[i-period]
		}
		if match {
			return a[:period], uint16(n / period)
		}
	}
	return a, 1
}

var asciiUnshifted = map[byte]layout.KeyName{
	' ': layout.KeySpace, '\n': layout.KeyEnter, '\t': layout.KeyTab,
	'-': layout.KeyMinus, '=': layout.KeyEqual, '[': layout.KeyLBracket, ']': layout.KeyRBracket,
	'\\': layout.KeyBackslash, ';': layout.KeySemicolon, '\'': layout.KeyQuote, '`': layout.KeyGrave,
	',': layout.KeyComma, '.': layout.KeyPeriod, '/': layout.KeyFwdslash,
}

var asciiShifted = map[byte]layout.KeyName{
	'!': layout.KeyNum1, '@': layout.KeyNum2, '#': layout.KeyNum3, '$': layout.KeyNum4,
	'%': layout.KeyNum5, '^': layout.KeyNum6, '&': layout.KeyNum7, '*': layout.KeyNum8,
	'(': layout.KeyNum9, ')': layout.KeyNum0, '_': layout.KeyMinus, '+': layout.KeyEqual,
	'{': layout.KeyLBracket, '}': layout.KeyRBracket, '|': layout.KeyBackslash, ':': layout.KeySemicolon,
	'"': layout.KeyQuote, '~': layout.KeyGrave, '<': layout.KeyComma, '>': layout.KeyPeriod,
	'?': layout.KeyFwdslash,
}

func asciiKeys(c byte) ([]layout.KeyName, bool) {
	switch {
	case c >= 'a' && c <= 'z':
		return []layout.KeyName{layout.KeyName(string(c))}, false
	case c >= 'A' && c <= 'Z':
		return []layout.KeyName{layout.KeyName(string(c + 32))}, true
	case c >= '1' && c <= '9':
		return []layout.KeyName{layout.KeyName("num" + string(c))}, false
	case c == '0':
		return []layout.KeyName{layout.KeyNum0}, false
	}
	if n, ok := asciiUnshifted[c]; ok {
		return []layout.KeyName{n}, false
	}
	if n, ok := asciiShifted[c]; ok {
		return []layout.KeyName{n}, true
	}
	return nil, false
}
