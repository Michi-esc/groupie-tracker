package main

import (
	"fmt"
	"groupie-tracker/models"
	"groupie-tracker/ui"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func main() {
	log.Println("[START] Loading Groupie Tracker...")

	// Initialize translations cache (pre-load all languages)
	ui.InitTranslations()
	log.Println("[OK] Translations pre-loaded")

	// Initialize geocode cache (load persisted geocoding data)
	if err := models.InitGeocodeCache(); err != nil {
		log.Printf("Warning: Could not initialize geocode cache: %v\n", err)
	}
	log.Println("[OK] Geocode cache initialized")

	// Créer l'application Fyne avec un ID stable pour les préférences
	myApp := app.NewWithID("groupie-tracker")
	log.Println("[OK] Fyne app created")

	// Appliquer le thème personnalisé
	myApp.Settings().SetTheme(&ui.CustomTheme{})

	// Créer la fenêtre principale
	win := ui.NewWindow(myApp)
	log.Println("[OK] Window created")

	// Afficher la liste au démarrage
	showArtistList(win)
	log.Println("[OK] Loading artists list...")

	// Afficher la fenêtre et lancer l'application
	win.Window.ShowAndRun()
}

func showArtistList(win *ui.Window) {
	// Afficher le chargement
	win.ShowLoading(ui.T().Loading)

	// Récupérer les artistes de l'API
	go func() {
		artists, err := models.FetchArtists()
		if err != nil {
			log.Println("Erreur:", err)
			// update UI on main thread with error + retry
			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   ui.T().Error,
				Content: err.Error(),
			})
			fyne.Do(func() {
				dialog.ShowError(err, win.Window)
				retryBtn := widget.NewButton("🔁 Retry", func() {
					showArtistList(win)
				})
				retryBtn.Importance = widget.HighImportance
				msg := widget.NewLabel(fmt.Sprintf("%s: %v", ui.T().Error, err))
				content := container.NewCenter(container.NewVBox(msg, retryBtn))
				win.SetContent(content)
			})
			return
		}

		// Créer et afficher la liste
		list := ui.NewArtistListWithWindow(win, artists, func(artist models.Artist) {
			showArtistDetail(win, artist)
		}, func() {
			showMap(win, artists)
		})

		// Connecter le callback de refresh pour le bouton langue
		win.OnRefresh = func() {
			showArtistList(win)
		}

		fyne.Do(func() {
			win.SetContent(list)
		})
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
	win.ShowLoading(ui.T().Loading)

	// Créer et afficher la page de carte en passant la window
	ui.NewMapPageWithWindow(win, artists, func() {
		// Retourner à la liste
		showArtistList(win)
	})
}
