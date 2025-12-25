// Package ansi provides ANSI escape sequences and palette presets.
// The palette values are derived from pkt.systems/pslog/ansi (MIT License).
// Only the data needed by prettyx is included here to avoid an external dep.
package ansi

// Base ANSI escape codes.
const (
	Reset         = "\x1b[0m"
	Bold          = "\x1b[1m"
	Faint         = "\x1b[90m"
	Red           = "\x1b[31m"
	Green         = "\x1b[32m"
	Yellow        = "\x1b[33m"
	Blue          = "\x1b[34m"
	Magenta       = "\x1b[35m"
	Cyan          = "\x1b[36m"
	Gray          = "\x1b[37m"
	BrightRed     = "\x1b[1;31m"
	BrightGreen   = "\x1b[1;32m"
	BrightYellow  = "\x1b[1;33m"
	BrightBlue    = "\x1b[1;34m"
	BrightMagenta = "\x1b[1;35m"
	BrightCyan    = "\x1b[1;36m"
	BrightWhite   = "\x1b[1;37m"
)

// Palette matches the structure used in pslog for semantic colours.
type Palette struct {
	Key         string
	String      string
	Num         string
	Bool        string
	Nil         string
	Brackets    string
	Punctuation string
	Trace       string
	Debug       string
	Info        string
	Warn        string
	Error       string
	Fatal       string
	Panic       string
	NoLevel     string
	Timestamp   string
	MessageKey  string
	Message     string
}

// PaletteJQDefault mirrors jq's default JQ_COLORS:
// 0;90:null, 0;39:false, 0;39:true, 0;39:numbers, 0;32:strings,
// 1;39:arrays, 1;39:objects, 1;34:keys.
var PaletteJQDefault = Palette{
	Key:         "\x1b[1;34m",
	String:      "\x1b[0;32m",
	Num:         "\x1b[0;39m",
	Bool:        "\x1b[0;39m",
	Nil:         "\x1b[0;90m",
	Brackets:    "\x1b[1;39m",
	Punctuation: "\x1b[1;39m",
}

// PaletteDefault is the pslog default (16-colour friendly).
var PaletteDefault = Palette{
	Key:         Cyan,
	String:      BrightBlue,
	Num:         Magenta,
	Bool:        Yellow,
	Nil:         Faint,
	Brackets:    Faint,
	Punctuation: Faint,
	Trace:       Blue,
	Debug:       Green,
	Info:        BrightGreen,
	Warn:        BrightYellow,
	Error:       BrightRed,
	Fatal:       BrightRed,
	Panic:       BrightRed,
	NoLevel:     Faint,
	Timestamp:   Faint,
	MessageKey:  Cyan,
	Message:     Bold,
}

// PaletteOutrunElectric delivers an outrun electric palette with neon pinks and blues.
var PaletteOutrunElectric = Palette{
	Key:         "\x1b[38;5;201m",
	String:      "\x1b[38;5;81m",
	Num:         "\x1b[38;5;99m",
	Bool:        "\x1b[38;5;69m",
	Nil:         "\x1b[38;5;60m",
	Brackets:    "\x1b[38;5;117m",
	Punctuation: "\x1b[38;5;60m",
	Trace:       "\x1b[38;5;33m",
	Debug:       "\x1b[38;5;39m",
	Info:        "\x1b[38;5;45m",
	Warn:        "\x1b[38;5;129m",
	Error:       "\x1b[38;5;205m",
	Fatal:       "\x1b[38;5;206m",
	Panic:       "\x1b[38;5;213m",
	NoLevel:     "\x1b[38;5;59m",
	Timestamp:   "\x1b[38;5;117m",
	MessageKey:  "\x1b[38;5;33m",
	Message:     "\x1b[1;38;5;219m",
}

// PaletteDoomIosvkem mirrors doom-emacs' iosvkem theme with dusky oranges and seafoam greens.
var PaletteDoomIosvkem = Palette{
	Key:         "\x1b[38;5;222m",
	String:      "\x1b[38;5;216m",
	Num:         "\x1b[38;5;109m",
	Bool:        "\x1b[38;5;151m",
	Nil:         "\x1b[38;5;244m",
	Brackets:    "\x1b[38;5;114m",
	Punctuation: "\x1b[38;5;244m",
	Trace:       "\x1b[38;5;66m",
	Debug:       "\x1b[38;5;72m",
	Info:        "\x1b[38;5;114m",
	Warn:        "\x1b[38;5;208m",
	Error:       "\x1b[38;5;203m",
	Fatal:       "\x1b[38;5;197m",
	Panic:       "\x1b[38;5;199m",
	NoLevel:     "\x1b[38;5;240m",
	Timestamp:   "\x1b[38;5;242m",
	MessageKey:  "\x1b[38;5;67m",
	Message:     "\x1b[1;38;5;223m",
}

// PaletteDoomGruvbox echoes doom-gruvbox colours with earthy reds and ambers.
var PaletteDoomGruvbox = Palette{
	Key:         "\x1b[38;5;214m",
	String:      "\x1b[38;5;178m",
	Num:         "\x1b[38;5;108m",
	Bool:        "\x1b[38;5;142m",
	Nil:         "\x1b[38;5;101m",
	Brackets:    "\x1b[38;5;172m",
	Punctuation: "\x1b[38;5;101m",
	Trace:       "\x1b[38;5;66m",
	Debug:       "\x1b[38;5;72m",
	Info:        "\x1b[38;5;107m",
	Warn:        "\x1b[38;5;208m",
	Error:       "\x1b[38;5;167m",
	Fatal:       "\x1b[38;5;160m",
	Panic:       "\x1b[38;5;161m",
	NoLevel:     "\x1b[38;5;95m",
	Timestamp:   "\x1b[38;5;137m",
	MessageKey:  "\x1b[38;5;172m",
	Message:     "\x1b[1;38;5;221m",
}

// PaletteDoomDracula mirrors doom-dracula with pink, purple, and cyan accents.
var PaletteDoomDracula = Palette{
	Key:         "\x1b[38;5;219m",
	String:      "\x1b[38;5;141m",
	Num:         "\x1b[38;5;111m",
	Bool:        "\x1b[38;5;81m",
	Nil:         "\x1b[38;5;240m",
	Brackets:    "\x1b[38;5;147m",
	Punctuation: "\x1b[38;5;95m",
	Trace:       "\x1b[38;5;60m",
	Debug:       "\x1b[38;5;98m",
	Info:        "\x1b[38;5;117m",
	Warn:        "\x1b[38;5;219m",
	Error:       "\x1b[38;5;204m",
	Fatal:       "\x1b[38;5;198m",
	Panic:       "\x1b[38;5;199m",
	NoLevel:     "\x1b[38;5;59m",
	Timestamp:   "\x1b[38;5;95m",
	MessageKey:  "\x1b[38;5;147m",
	Message:     "\x1b[1;38;5;225m",
}

// PaletteDoomNord channels doom-nord with cool glacier blues.
var PaletteDoomNord = Palette{
	Key:         "\x1b[38;5;153m",
	String:      "\x1b[38;5;152m",
	Num:         "\x1b[38;5;109m",
	Bool:        "\x1b[38;5;115m",
	Nil:         "\x1b[38;5;245m",
	Brackets:    "\x1b[38;5;110m",
	Punctuation: "\x1b[38;5;245m",
	Trace:       "\x1b[38;5;67m",
	Debug:       "\x1b[38;5;74m",
	Info:        "\x1b[38;5;117m",
	Warn:        "\x1b[38;5;179m",
	Error:       "\x1b[38;5;210m",
	Fatal:       "\x1b[38;5;204m",
	Panic:       "\x1b[38;5;205m",
	NoLevel:     "\x1b[38;5;103m",
	Timestamp:   "\x1b[38;5;109m",
	MessageKey:  "\x1b[38;5;110m",
	Message:     "\x1b[1;38;5;195m",
}

// PaletteTokyoNight draws on Tokyo Night's neon blues, violets, and warm highlights.
var PaletteTokyoNight = Palette{
	Key:         "\x1b[38;5;69m",
	String:      "\x1b[38;5;110m",
	Num:         "\x1b[38;5;176m",
	Bool:        "\x1b[38;5;117m",
	Nil:         "\x1b[38;5;244m",
	Brackets:    "\x1b[38;5;74m",
	Punctuation: "\x1b[38;5;244m",
	Trace:       "\x1b[38;5;63m",
	Debug:       "\x1b[38;5;67m",
	Info:        "\x1b[38;5;111m",
	Warn:        "\x1b[38;5;173m",
	Error:       "\x1b[38;5;210m",
	Fatal:       "\x1b[38;5;205m",
	Panic:       "\x1b[38;5;219m",
	NoLevel:     "\x1b[38;5;239m",
	Timestamp:   "\x1b[38;5;109m",
	MessageKey:  "\x1b[38;5;74m",
	Message:     "\x1b[1;38;5;218m",
}

// PaletteSolarizedNightfall adapts Solarized Night with teal highlights and amber warnings.
var PaletteSolarizedNightfall = Palette{
	Key:         "\x1b[38;5;37m",
	String:      "\x1b[38;5;86m",
	Num:         "\x1b[38;5;61m",
	Bool:        "\x1b[38;5;65m",
	Nil:         "\x1b[38;5;239m",
	Brackets:    "\x1b[38;5;33m",
	Punctuation: "\x1b[38;5;239m",
	Trace:       "\x1b[38;5;24m",
	Debug:       "\x1b[38;5;30m",
	Info:        "\x1b[38;5;36m",
	Warn:        "\x1b[38;5;136m",
	Error:       "\x1b[38;5;160m",
	Fatal:       "\x1b[38;5;166m",
	Panic:       "\x1b[38;5;161m",
	NoLevel:     "\x1b[38;5;238m",
	Timestamp:   "\x1b[38;5;244m",
	MessageKey:  "\x1b[38;5;33m",
	Message:     "\x1b[1;38;5;230m",
}

// PaletteCatppuccinMocha recreates Catppuccin Mocha with soft pastels and rosewater highlights.
var PaletteCatppuccinMocha = Palette{
	Key:         "\x1b[38;5;217m",
	String:      "\x1b[38;5;183m",
	Num:         "\x1b[38;5;147m",
	Bool:        "\x1b[38;5;152m",
	Nil:         "\x1b[38;5;244m",
	Brackets:    "\x1b[38;5;182m",
	Punctuation: "\x1b[38;5;244m",
	Trace:       "\x1b[38;5;104m",
	Debug:       "\x1b[38;5;109m",
	Info:        "\x1b[38;5;150m",
	Warn:        "\x1b[38;5;216m",
	Error:       "\x1b[38;5;211m",
	Fatal:       "\x1b[38;5;205m",
	Panic:       "\x1b[38;5;204m",
	NoLevel:     "\x1b[38;5;240m",
	Timestamp:   "\x1b[38;5;110m",
	MessageKey:  "\x1b[38;5;182m",
	Message:     "\x1b[1;38;5;223m",
}

// PaletteGruvboxLight is a Gruvbox light variant with warm browns and turquoise hints.
var PaletteGruvboxLight = Palette{
	Key:         "\x1b[38;5;130m",
	String:      "\x1b[38;5;108m",
	Num:         "\x1b[38;5;66m",
	Bool:        "\x1b[38;5;142m",
	Nil:         "\x1b[38;5;180m",
	Brackets:    "\x1b[38;5;136m",
	Punctuation: "\x1b[38;5;180m",
	Trace:       "\x1b[38;5;109m",
	Debug:       "\x1b[38;5;114m",
	Info:        "\x1b[38;5;73m",
	Warn:        "\x1b[38;5;173m",
	Error:       "\x1b[38;5;167m",
	Fatal:       "\x1b[38;5;161m",
	Panic:       "\x1b[38;5;125m",
	NoLevel:     "\x1b[38;5;181m",
	Timestamp:   "\x1b[38;5;180m",
	MessageKey:  "\x1b[38;5;136m",
	Message:     "\x1b[1;38;5;223m",
}

// PaletteMonokaiVibrant supplies a Monokai-inspired mix of neon yellows and minty greens.
var PaletteMonokaiVibrant = Palette{
	Key:         "\x1b[38;5;229m",
	String:      "\x1b[38;5;121m",
	Num:         "\x1b[38;5;198m",
	Bool:        "\x1b[38;5;118m",
	Nil:         "\x1b[38;5;59m",
	Brackets:    "\x1b[38;5;141m",
	Punctuation: "\x1b[38;5;59m",
	Trace:       "\x1b[38;5;104m",
	Debug:       "\x1b[38;5;114m",
	Info:        "\x1b[38;5;121m",
	Warn:        "\x1b[38;5;215m",
	Error:       "\x1b[38;5;197m",
	Fatal:       "\x1b[38;5;161m",
	Panic:       "\x1b[38;5;201m",
	NoLevel:     "\x1b[38;5;240m",
	Timestamp:   "\x1b[38;5;103m",
	MessageKey:  "\x1b[38;5;141m",
	Message:     "\x1b[1;38;5;229m",
}

// PaletteOneDarkAurora reflects the One Dark Aurora theme with cyan, violet, and crimson tones.
var PaletteOneDarkAurora = Palette{
	Key:         "\x1b[38;5;110m",
	String:      "\x1b[38;5;147m",
	Num:         "\x1b[38;5;141m",
	Bool:        "\x1b[38;5;115m",
	Nil:         "\x1b[38;5;59m",
	Brackets:    "\x1b[38;5;75m",
	Punctuation: "\x1b[38;5;59m",
	Trace:       "\x1b[38;5;24m",
	Debug:       "\x1b[38;5;31m",
	Info:        "\x1b[38;5;38m",
	Warn:        "\x1b[38;5;178m",
	Error:       "\x1b[38;5;203m",
	Fatal:       "\x1b[38;5;197m",
	Panic:       "\x1b[38;5;199m",
	NoLevel:     "\x1b[38;5;240m",
	Timestamp:   "\x1b[38;5;109m",
	MessageKey:  "\x1b[38;5;75m",
	Message:     "\x1b[1;38;5;189m",
}

// PaletteSynthwave84 channels synthwave aesthetics with glowing magentas, cyans, and gold accents.
var PaletteSynthwave84 = Palette{
	Key:         "\x1b[38;5;198m",
	String:      "\x1b[38;5;51m",
	Num:         "\x1b[38;5;207m",
	Bool:        "\x1b[38;5;219m",
	Nil:         "\x1b[38;5;102m",
	Brackets:    "\x1b[38;5;45m",
	Punctuation: "\x1b[38;5;102m",
	Trace:       "\x1b[38;5;63m",
	Debug:       "\x1b[38;5;69m",
	Info:        "\x1b[38;5;81m",
	Warn:        "\x1b[38;5;220m",
	Error:       "\x1b[38;5;205m",
	Fatal:       "\x1b[38;5;200m",
	Panic:       "\x1b[38;5;201m",
	NoLevel:     "\x1b[38;5;60m",
	Timestamp:   "\x1b[38;5;69m",
	MessageKey:  "\x1b[38;5;45m",
	Message:     "\x1b[1;38;5;219m",
}
