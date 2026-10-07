package editor

// Viewport is the visible window over the document: Top is the first
// visible line, Left the first visible screen column (tabs expanded), and
// Width/Height its size in cells.
type Viewport struct {
	Top, Left     int
	Width, Height int
}

// scrollMargin keeps a few lines of context above and below the cursor.
const scrollMargin = 3

// follow scrolls the viewport just enough to keep (line, col) visible, col
// being a screen column.
func (v *Viewport) follow(line, col int) {
	if v.Height > 0 {
		margin := min(scrollMargin, (v.Height-1)/2)
		if line < v.Top+margin {
			v.Top = max(line-margin, 0)
		}
		if line >= v.Top+v.Height-margin {
			v.Top = line - v.Height + margin + 1
		}
	}
	if v.Width > 0 {
		if col < v.Left {
			v.Left = col
		}
		if col >= v.Left+v.Width {
			v.Left = col - v.Width + 1
		}
	}
}
