package main

import (
	"groupie-tracker/models"
	"groupie-tracker/ui"
	"log"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
)

func main() {
	log.Println("🚀 Démarrage de l'application Groupie Tracker...")

	// Créer l'application Fyne
	myApp := app.New()
	log.Println("✓ Application Fyne créée")

	// Créer la fenêtre principale
	win := ui.NewWindow(myApp)
	log.Println("✓ Fenêtre créée")

	// Afficher la liste au démarrage
	showArtistList(win)
	log.Println("✓ Liste d'artistes en cours de chargement...")

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
