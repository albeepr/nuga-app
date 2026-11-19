// Package device provides supported device models
package device

// Model represents supported keyboard model
type Model string

const (
	// Halo75 represents NuPhy Halo 75 keyboard name
	Halo75 Model = "Halo75"
	// Halo65 represents NuPhy Halo 65 keyboard name
	Halo65 Model = "Halo65"
	// Halo96 represents NuPhy Halo 96 keyboard name
	Halo96 Model = "Halo96"
	// Air96V2 represents NuPhy Air96 V2 keyboard name (QMK/VIA firmware)
	Air96V2 Model = "Air96V2"
)

// IsSupported returns true if model is supported by Nuga library
func IsSupported(model string) bool {
	m := Model(model)
	return m == Halo75 || m == Halo65 || m == Halo96 || m == Air96V2
}

// IsVIA returns true if model is driven by QMK/VIA firmware instead of NuPhy protocol
func IsVIA(model Model) bool {
	return model == Air96V2
}
