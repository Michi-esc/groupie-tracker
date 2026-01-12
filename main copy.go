package main

import (
	"groupie-tracker/models"
	"groupie-tracker/ui"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func main() {
	// Initialisation de l'application Fyne
	myApp := app.New()
	appUI := ui.NewAppUI(myApp)

	// Variable pour stocker la vue actuelle
	var currentArtist *models.Artist

	// Fonction de retour à la liste
	onBack := func() {
		currentArtist = nil
		showArtistList(appUI)
	}

	// Fonction de recherche (à implémenter)
	onSearch := func() {
		dialog.ShowInformation("Recherche", "Fonction de recherche à implémenter", appUI.Window)
	}

	// Enregistrer les raccourcis clavier
	ui.RegisterShortcuts(appUI.Window, onBack, onSearch)

	// Afficher la liste au démarrage
	showArtistList(appUI)

	// Lancer l'application
	appUI.Window.ShowAndRun()
}

// showArtistList affiche la liste des artistes avec un loader
func showArtistList(appUI *ui.AppUI) {
	// Afficher un loader pendant le chargement
	spinner := widget.NewProgressBarInfinite()
	appUI.SetContent(container.NewCenter(spinner))

	// Simuler un appel API (à remplacer par: artists, err := api.GetArtists())
	go func() {
		// TODO: Remplacer par le vrai appel API
		// artists, err := api.GetArtists()
		// if err != nil {
		//     dialog.ShowError(err, appUI.Window)
		//     return
		// }

		// Données de test
		artists := getDummyArtists()

		// Fonction appelée lors du clic sur une carte
		onSelectArtist := func(artist models.Artist) {
			showArtistDetail(appUI, artist)
		}

		// Créer et afficher la liste
		artistList := ui.NewArtistList(artists, onSelectArtist)
		appUI.SetContent(artistList)
	}()
}

// showArtistDetail affiche la page de détail d'un artiste
func showArtistDetail(appUI *ui.AppUI, artist models.Artist) {
	onBack := func() {
		showArtistList(appUI)
	}

	detailPage := ui.NewArtistPage(artist, onBack)
	appUI.SetContent(detailPage)
}

// getDummyArtists retourne des données de test
// À SUPPRIMER une fois l'API backend connectée
func getDummyArtists() []models.Artist {
	return []models.Artist{
		{
			ID:           1,
			Name:         "Queen",
			Members:      []string{"Freddie Mercury", "Brian May", "Roger Taylor", "John Deacon"},
			CreationDate: 1970,
			FirstAlbum:   "14-12-1973",
			Image:        "https://example.com/queen.jpg",
			Locations:    []string{"London", "Paris", "New York"},
			ConcertDates: []string{"01-01-2024", "15-02-2024", "30-03-2024"},
		},
		{
			ID:           2,
			Name:         "The Beatles",
			Members:      []string{"John Lennon", "Paul McCartney", "George Harrison", "Ringo Starr"},
			CreationDate: 1960,
			FirstAlbum:   "22-03-1963",
			Image:        "https://example.com/beatles.jpg",
			Locations:    []string{"Liverpool", "Hamburg", "New York"},
			ConcertDates: []string{"10-05-2024", "20-06-2024"},
		},
		{
			ID:           3,
			Name:         "Pink Floyd",
			Members:      []string{"Roger Waters", "David Gilmour", "Nick Mason", "Richard Wright"},
			CreationDate: 1965,
			FirstAlbum:   "05-08-1967",
			Image:        "https://example.com/pinkfloyd.jpg",
			Locations:    []string{"London", "Los Angeles"},
			ConcertDates: []string{"01-07-2024"},
		},
	}
}
