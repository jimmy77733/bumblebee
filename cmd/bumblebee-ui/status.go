//go:build windows

package main

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type lineBoxLayout struct {
	lines float32
}

func (l *lineBoxLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(1, linesHeight(l.lines))
}

func (l *lineBoxLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objects {
		if o == nil || !o.Visible() {
			continue
		}
		o.Resize(size)
		o.Move(fyne.NewPos(0, 0))
	}
}

func linesHeight(lines float32) float32 {
	if lines < 1 {
		lines = 1
	}
	th := fyne.CurrentApp().Settings().Theme()
	textH := fyne.MeasureText("Hg", th.Size(theme.SizeNameText), fyne.TextStyle{}).Height
	sp := th.Size(theme.SizeNameLineSpacing)
	pad := th.Size(theme.SizeNameInnerPadding)
	return textH*lines + sp*(lines-1) + pad
}

func newScanStatus() (*widget.Label, *widget.Label, *fyne.Container) {
	head := widget.NewLabel("就緒")
	head.Wrapping = fyne.TextWrapOff
	head.Truncation = fyne.TextTruncateEllipsis
	path := widget.NewLabel(" ")
	path.Wrapping = fyne.TextWrapBreak
	path.Truncation = fyne.TextTruncateEllipsis
	box := container.NewVBox(
		container.New(&lineBoxLayout{lines: 1}, head),
		container.New(&lineBoxLayout{lines: 3}, path),
	)
	return head, path, box
}

func splitStatusText(text string) (head, path string) {
	text = strings.TrimRight(text, "\n")
	head, path, ok := strings.Cut(text, "\n")
	if !ok {
		return text, " "
	}
	path = strings.TrimSpace(path)
	if path == "" {
		path = " "
	}
	return head, path
}
