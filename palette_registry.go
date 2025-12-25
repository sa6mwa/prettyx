package prettyx

import (
	"fmt"
	"sort"
	"strings"

	"pkt.systems/prettyx/internal/ansi"
)

const (
	paletteDefaultName = "default"
	paletteNoneName    = "none"
)

var paletteRegistry = map[string]ansi.Palette{
	paletteDefaultName:    ansi.PaletteJQDefault,
	"jq":                  ansi.PaletteJQDefault,
	"catppuccin-mocha":    ansi.PaletteCatppuccinMocha,
	"doom-dracula":        ansi.PaletteDoomDracula,
	"doom-gruvbox":        ansi.PaletteDoomGruvbox,
	"doom-iosvkem":        ansi.PaletteDoomIosvkem,
	"doom-nord":           ansi.PaletteDoomNord,
	"gruvbox-light":       ansi.PaletteGruvboxLight,
	"monokai-vibrant":     ansi.PaletteMonokaiVibrant,
	"one-dark-aurora":     ansi.PaletteOneDarkAurora,
	"outrun-electric":     ansi.PaletteOutrunElectric,
	"solarized-nightfall": ansi.PaletteSolarizedNightfall,
	"synthwave84":         ansi.PaletteSynthwave84,
	"tokyo-night":         ansi.PaletteTokyoNight,
	"default-16":          ansi.PaletteDefault, // pslog classic
	"classic":             ansi.PaletteDefault, // pslog classic
	"pslog":               ansi.PaletteDefault,
}

// PaletteNames returns the sorted list of palette names, including "none".
func PaletteNames() []string {
	names := make([]string, 0, len(paletteRegistry)+1)
	for name := range paletteRegistry {
		names = append(names, name)
	}
	names = append(names, paletteNoneName)
	sort.Strings(names)
	return names
}

// resolvePalette returns the ColorPalette for the given options, defaulting to
// paletteDefaultName when opts.Palette is empty. The special palette name
// "none" disables colouring. If enableColor is false we return a no-color
// palette regardless of the selection (still validating the name).
func resolvePalette(opts *Options, enableColor bool) (ColorPalette, error) {
	name := paletteDefaultName
	if opts != nil && strings.TrimSpace(opts.Palette) != "" {
		name = strings.ToLower(strings.TrimSpace(opts.Palette))
	}

	if name == paletteNoneName {
		return NoColorPalette(), nil
	}

	ap, ok := paletteRegistry[name]
	if !ok {
		return ColorPalette{}, fmt.Errorf("unknown palette %q (use one of: %s)", name, strings.Join(PaletteNames(), ", "))
	}

	if !enableColor {
		return NoColorPalette(), nil
	}
	return colorPaletteFromAnsi(ap), nil
}

func colorPaletteFromAnsi(ap ansi.Palette) ColorPalette {
	brackets := ap.Brackets
	if brackets == "" {
		brackets = ap.Nil
	}
	punct := ap.Punctuation
	if punct == "" {
		punct = brackets
	}

	return ColorPalette{
		Key:         ap.Key,
		String:      ap.String,
		Number:      ap.Num,
		True:        ap.Bool,
		False:       ap.Bool,
		Null:        ap.Nil,
		Brackets:    brackets,
		Punctuation: punct,
	}
}

// NoColorPalette disables all styling while keeping the formatter path shared.
func NoColorPalette() ColorPalette {
	return ColorPalette{}
}
