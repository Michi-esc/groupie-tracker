package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// window wrapper
type Window struct {
	Window     fyne.Window
	Content    *fyne.Container
	LangButton *widget.Button
	OnRefresh  func()
}

// create window
func NewWindow(app fyne.App) *Window {
	w := app.NewWindow(T().WindowTitle)
	w.Resize(fyne.NewSize(1200, 800))
	w.CenterOnScreen()

	content := container.NewMax()

	// bouton langue
	langButton := widget.NewButton("🌐 FR/EN", nil)
	langButton.Importance = widget.LowImportance

	win := &Window{
		Window:     w,
		Content:    content,
		LangButton: langButton,
	}

	// action bouton langue
	langButton.OnTapped = func() {
		ToggleLang()
		w.SetTitle(T().WindowTitle)
		if win.OnRefresh != nil {
			win.OnRefresh()
		}
	}

	w.SetContent(content)

	return win
}

// change content
func (w *Window) SetContent(content fyne.CanvasObject) {
	// force ui thread
	fyne.Do(func() {
		w.Content.Objects = []fyne.CanvasObject{content}
		w.Content.Refresh()
	})
}

// show loader
func (w *Window) ShowLoading(message string) {
	progress := widget.NewProgressBarInfinite()
	label := widget.NewLabel(message)

	content := container.NewCenter(
		container.NewVBox(
			progress,
			label,
		),
	)

	w.SetContent(content)
}
