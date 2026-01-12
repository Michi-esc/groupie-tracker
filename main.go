package main

import (
	"groupie-tracker/models"
	"groupie-tracker/ui"
	"log"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
)

func main() {
	// Créer l'application
	myApp := app.New()

	// Créer la fenêtre principale
	win := ui.NewWindow(myApp)

	// Variable pour stocker l'artiste actuellement affiché
	var currentArtist *models.Artist

	// Fonction pour revenir à la liste
	onBack := func() {
		currentArtist = nil
		showArtistList(win)
	}

	// Fonction pour sélectionner un artiste
	onSelectArtist := func(artist models.Artist) {
		currentArtist = &artist
		showArtistDetail(win, artist)
	}

	// Afficher la liste au démarrage
	showArtistList(win)

	// Afficher la fenêtre et lancer l'application
	win.Window.ShowAndRun()
}

func showArtistList(win *ui.Window) {
	// Afficher le chargement
	win.ShowLoading("Chargement des artistes...")

	// Récupérer les artistes de l'API
	go func() {
		artists, err := models.FetchArtists()
		if err != nil {
			log.Println("Erreur:", err)
			dialog.ShowError(err, win.Window)
			return
		}

		// Créer et afficher la liste
		list := ui.NewArtistList(artists, func(artist models.Artist) {
			showArtistDetail(win, artist)
		})

		win.SetContent(list)
	}()
}

func showArtistDetail(win *ui.Window, artist models.Artist) {
	// Créer et afficher la page de détail
	detailPage := ui.NewArtistPage(artist, func() {
		showArtistList(win)
	})

	win.SetContent(detailPage)
}
