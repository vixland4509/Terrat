package terminal

type Cell struct {
	Char      rune
	FG        Color
	BG        Color
	Bold      bool
	Underline bool
	Inverse   bool
}

func EmptyCell() Cell {
	return Cell{
		Char: ' ',
		FG:   ColorDefaultFG,
		BG:   ColorDefaultBG,
	}
}
