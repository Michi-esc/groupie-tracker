package main

import (
	"groupie-tracker/models"
	"groupie-tracker/ui"
	"log"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
)

func main() {
	log.Println("[START] Loading Groupie Tracker...")

	myApp := app.New()
	log.Println("[OK] Fyne app created")

	myApp.Settings().SetTheme(&ui.CustomTheme{})

	win := ui.NewWindow(myApp)
	log.Println("[OK] Window created")

	showArtistList(win)
	log.Println("[OK] Loading artists list...")

	win.Window.ShowAndRun()
}

func showArtistList(win *ui.Window) {
	win.ShowLoading("Chargement des artistes...")

	go func() {
		artists, err := models.FetchArtists()
		if err != nil {
			log.Println("Erreur:", err)
			dialog.ShowError(err, win.Window)
			return
		}

		list := ui.NewArtistList(artists, func(artist models.Artist) {
			showArtistDetail(win, artist)
		}, func() {
			showMap(win, artists)
		})

		win.SetContent(list)
	}()
}

func showArtistDetail(win *ui.Window, artist models.Artist) {
	detailPage := ui.NewArtistPage(artist, func() {
		showArtistList(win)
	})

	win.SetContent(detailPage)
}

func showMap(win *ui.Window, artists []models.Artist) {
	win.ShowLoading("Chargement de la carte des concerts...")

	ui.NewMapPageWithWindow(win, artists, func() {
		showArtistList(win)
	})
}
