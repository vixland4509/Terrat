package terminal

type Theme struct {
	ID          string
	Name        string
	Category    string
	IsDark      bool
	BG          Color
	FG          Color
	HeaderBG    Color
	HeaderLine  Color
	Border      Color
	Cursor      Color
	SelectionBG Color
	SelectionFG Color
	BadgeText   Color
	MutedText   Color
	CloseDot    Color
	MinDot      Color
	MaxDot      Color
	ANSI        [16]Color
}

var ThemeTokyoNight = &Theme{
	ID:          "tokyo-night",
	Name:        "Tokyo Night",
	Category:    "Dark",
	IsDark:      true,
	BG:          Color(0x1a1b26),
	FG:          Color(0xc0caf5),
	HeaderBG:    Color(0x13141c),
	HeaderLine:  Color(0x292e42),
	Border:      Color(0x3b4261),
	Cursor:      Color(0x7aa2f7),
	SelectionBG: Color(0x364a82),
	SelectionFG: Color(0xffffff),
	BadgeText:   Color(0x7dcfff),
	MutedText:   Color(0x565f89),
	CloseDot:    Color(0xf7768e),
	MinDot:      Color(0xe0af68),
	MaxDot:      Color(0x9ece6a),
	ANSI: [16]Color{
		0x15161e, 0xf7768e, 0x9ece6a, 0xe0af68, 0x7aa2f7, 0xbb9af7, 0x7dcfff, 0xa9b1d6,
		0x414868, 0xf7768e, 0x73daca, 0xe0af68, 0x7aa2f7, 0xbb9af7, 0x2ac3de, 0xc0caf5,
	},
}

var ThemeCatppuccinMocha = &Theme{
	ID:          "catppuccin-mocha",
	Name:        "Catppuccin Mocha",
	Category:    "Dark",
	IsDark:      true,
	BG:          Color(0x1e1e2e),
	FG:          Color(0xcdd6f4),
	HeaderBG:    Color(0x181825),
	HeaderLine:  Color(0x313244),
	Border:      Color(0x45475a),
	Cursor:      Color(0xf5e0dc),
	SelectionBG: Color(0x585b70),
	SelectionFG: Color(0xffffff),
	BadgeText:   Color(0x89dceb),
	MutedText:   Color(0x6c7086),
	CloseDot:    Color(0xf38ba8),
	MinDot:      Color(0xf9e2af),
	MaxDot:      Color(0xa6e3a1),
	ANSI: [16]Color{
		0x45475a, 0xf38ba8, 0xa6e3a1, 0xf9e2af, 0x89b4fa, 0xf5c2e7, 0x94e2d5, 0xbac2de,
		0x585b70, 0xf38ba8, 0xa6e3a1, 0xf9e2af, 0x89b4fa, 0xf5c2e7, 0x94e2d5, 0xa6adc8,
	},
}

var ThemeTokyoDay = &Theme{
	ID:          "tokyo-day",
	Name:        "Tokyo Day",
	Category:    "Light",
	IsDark:      false,
	BG:          Color(0xf2f4f8),
	FG:          Color(0x343b58),
	HeaderBG:    Color(0xe1e3e8),
	HeaderLine:  Color(0xc4c7d0),
	Border:      Color(0xb8b9c1),
	Cursor:      Color(0x2e7de9),
	SelectionBG: Color(0xb4d5fe),
	SelectionFG: Color(0x1a1b26),
	BadgeText:   Color(0x1667c2),
	MutedText:   Color(0x68707e),
	CloseDot:    Color(0xf52a65),
	MinDot:      Color(0x8c6c3e),
	MaxDot:      Color(0x587539),
	ANSI: [16]Color{
		0xe9e9ed, 0xf52a65, 0x587539, 0x8c6c3e, 0x2e7de9, 0x9854f1, 0x007197, 0x343b58,
		0x9699a3, 0xf52a65, 0x587539, 0x8c6c3e, 0x2e7de9, 0x9854f1, 0x007197, 0x1a1b26,
	},
}

var ThemeSolarizedLight = &Theme{
	ID:          "solarized-light",
	Name:        "Solarized Light",
	Category:    "Light",
	IsDark:      false,
	BG:          Color(0xfdf6e3),
	FG:          Color(0x657b83),
	HeaderBG:    Color(0xeee8d5),
	HeaderLine:  Color(0xd3cbb7),
	Border:      Color(0x93a1a1),
	Cursor:      Color(0x268bd2),
	SelectionBG: Color(0xd6cbb0),
	SelectionFG: Color(0x073642),
	BadgeText:   Color(0x2aa198),
	MutedText:   Color(0x839496),
	CloseDot:    Color(0xdc322f),
	MinDot:      Color(0xb58900),
	MaxDot:      Color(0x859900),
	ANSI: [16]Color{
		0xeee8d5, 0xdc322f, 0x859900, 0xb58900, 0x268bd2, 0xd33682, 0x2aa198, 0x073642,
		0x93a1a1, 0xcb4b16, 0x586e75, 0x657b83, 0x839496, 0x6c71c4, 0x93a1a1, 0x002b36,
	},
}

var AllThemes = []*Theme{
	ThemeTokyoNight,
	ThemeCatppuccinMocha,
	ThemeTokyoDay,
	ThemeSolarizedLight,
}

func GetTheme(id string) *Theme {
	for _, t := range AllThemes {
		if t.ID == id {
			return t
		}
	}
	return nil
}

func ResolveTheme(pref string, isSystemDark bool) *Theme {
	switch pref {
	case "tokyo-night":
		return ThemeTokyoNight
	case "catppuccin-mocha":
		return ThemeCatppuccinMocha
	case "tokyo-day":
		return ThemeTokyoDay
	case "solarized-light":
		return ThemeSolarizedLight
	case "auto", "":
		if isSystemDark {
			return ThemeTokyoNight
		}
		return ThemeTokyoDay
	default:
		if t := GetTheme(pref); t != nil {
			return t
		}
		if isSystemDark {
			return ThemeTokyoNight
		}
		return ThemeTokyoDay
	}
}
