package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

// AppUI représente l'interface principale de l'application
type AppUI struct {
	App    fyne.App
	Window fyne.Window
	Body   *fyne.Container
}

// NewAppUI crée et initialise la fenêtre principale
func NewAppUI(app fyne.App) *AppUI {
	w := app.NewWindow("Groupie Tracker")

	body := container.NewMax()
	w.SetContent(body)
	w.Resize(fyne.NewSize(1200, 800))

	return &AppUI{
		App:    app,
		Window: w,
		Body:   body,
	}
}

// SetContent remplace le contenu de la fenêtre
func (ui *AppUI) SetContent(obj fyne.CanvasObject) {
	ui.Body.Objects = []fyne.CanvasObject{obj}
	ui.Body.Refresh()
}
