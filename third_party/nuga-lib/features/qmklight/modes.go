package qmklight

import "github.com/mishamyrt/nuga-lib/features/light"

const (
	c  = light.SpecificColor
	s  = light.Speed
	cs = light.SpecificColor | light.Speed
)

// Domain lists RGB matrix effects in the order of the NuPhy Air96 V2 VIA
// definition. Codes are QMK `rgb_matrix_get_mode()` values; 0 means off.
var Domain = light.Domain{
	Name: "Backlight",
	Modes: []light.Mode{
		{Name: "Lights off", Code: 0, Features: 0},
		{Name: "Solid color", Code: 1, Features: c},
		{Name: "Gradient up/down", Code: 2, Features: cs},
		{Name: "Gradient left/right", Code: 3, Features: cs},
		{Name: "Breathing", Code: 4, Features: cs},
		{Name: "Band saturation", Code: 5, Features: cs},
		{Name: "Band value", Code: 6, Features: cs},
		{Name: "Pinwheel saturation", Code: 7, Features: cs},
		{Name: "Pinwheel value", Code: 8, Features: cs},
		{Name: "Spiral saturation", Code: 9, Features: cs},
		{Name: "Spiral value", Code: 10, Features: cs},
		{Name: "Cycle all", Code: 11, Features: s},
		{Name: "Cycle left/right", Code: 12, Features: s},
		{Name: "Cycle up/down", Code: 13, Features: s},
		{Name: "Rainbow moving chevron", Code: 14, Features: s},
		{Name: "Cycle out/in", Code: 15, Features: s},
		{Name: "Cycle out/in dual", Code: 16, Features: s},
		{Name: "Cycle pinwheel", Code: 17, Features: s},
		{Name: "Cycle spiral", Code: 18, Features: s},
		{Name: "Dual beacon", Code: 19, Features: cs},
		{Name: "Rainbow beacon", Code: 20, Features: s},
		{Name: "Rainbow pinwheels", Code: 21, Features: s},
		{Name: "Raindrops", Code: 22, Features: c},
		{Name: "Jellybean raindrops", Code: 23, Features: 0},
		{Name: "Hue breathing", Code: 24, Features: cs},
		{Name: "Hue pendulum", Code: 25, Features: cs},
		{Name: "Hue wave", Code: 26, Features: cs},
		{Name: "Typing heatmap", Code: 27, Features: 0},
		{Name: "Digital rain", Code: 28, Features: 0},
		{Name: "Reactive simple", Code: 29, Features: cs},
		{Name: "Reactive", Code: 30, Features: cs},
		{Name: "Reactive wide", Code: 31, Features: cs},
		{Name: "Reactive multiwide", Code: 32, Features: cs},
		{Name: "Reactive cross", Code: 33, Features: cs},
		{Name: "Reactive multicross", Code: 34, Features: cs},
		{Name: "Reactive nexus", Code: 35, Features: cs},
		{Name: "Reactive multinexus", Code: 36, Features: cs},
		{Name: "Splash", Code: 37, Features: s},
		{Name: "Multisplash", Code: 38, Features: s},
		{Name: "Solid splash", Code: 39, Features: cs},
		{Name: "Solid multisplash", Code: 40, Features: cs},
	},
}

// ModesCount is the number of non-off modes.
const ModesCount = 40

var defaultColors = [light.ModeColorsCount]light.RGB{
	light.ColorRed,
	light.ColorGreen,
	light.ColorBlue,
	light.ColorYellow,
	light.ColorPurple,
	light.ColorCyan,
	light.ColorWhite,
}
