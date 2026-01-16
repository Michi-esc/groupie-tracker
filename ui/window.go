package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Window gère l'interface principale
type Window struct {
	Window  fyne.Window
	Content *fyne.Container
}

// NewWindow crée une nouvelle fenêtre
func NewWindow(app fyne.App) *Window {
	w := app.NewWindow("Groupie Tracker")
	w.Resize(fyne.NewSize(1200, 800))
	w.CenterOnScreen()

	content := container.NewMax()
	w.SetContent(content)

	return &Window{
		Window:  w,
		Content: content,
	}
}

// SetContent change le contenu de la fenêtre (thread-safe)
func (w *Window) SetContent(content fyne.CanvasObject) {
	// S'assurer que la modification se fait dans le thread UI de Fyne
	fyne.Do(func() {
		w.Content.Objects = []fyne.CanvasObject{content}
		w.Content.Refresh()
	})
}

// ShowLoading affiche un indicateur de chargement
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
