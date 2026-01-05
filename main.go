package main

import (
	"groupie-tracker/ui"

	"fyne.io/fyne/v2/app"
)

func main() {
	a := app.New()
	ui.CreateWindow(a)
}
