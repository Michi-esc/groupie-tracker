package main

import (
	"groupie-tracker/models"
	"groupie-tracker/ui"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
)

func main() {
	log.Println("🚀 Démarrage de l'application Groupie Tracker...")

	// Créer l'application Fyne
	myApp := app.New()
	log.Println("✓ Application Fyne créée")

	// Appliquer le thème personnalisé
	myApp.Settings().SetTheme(&ui.CustomTheme{})

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

		// Créer et afficher la liste depuis le thread Fyne
		fyne.CurrentApp().Driver().CanvasForObject(win.Content)
		list := ui.NewArtistList(artists, func(artist models.Artist) {
			showArtistDetail(win, artist)
		}, func() {
			showMap(win, artists)
		})

		win.SetContent(list)
	}()
}

func showArtistDetail(win *ui.Window, artist models.Artist) {
	// Créer et afficher la page de détail
	detailPage := ui.NewArtistPage(artist, func() {
		// Retourner à la liste
		showArtistList(win)
	})

	win.SetContent(detailPage)
}

func showMap(win *ui.Window, artists []models.Artist) {
	// Afficher le chargement
	win.ShowLoading("Chargement de la carte des concerts...")

	// Créer et afficher la page de carte en passant la window
	ui.NewMapPageWithWindow(win, artists, func() {
		// Retourner à la liste
		showArtistList(win)
	})
}
