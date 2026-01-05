package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

func CreateWindow(a fyne.App) {
	w := a.NewWindow("Groupie Tracker")

	content := container.NewVBox(
	// contenu plus tard
	)

	w.SetContent(content)
	w.Resize(fyne.NewSize(1200, 800))
	w.ShowAndRun()
}
