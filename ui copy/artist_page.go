package ui

import (
	"fmt"
	"groupie-tracker/models"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// NewArtistPage crée la page de détail d'un artiste
// Affiche : Image, Membres, Dates, Lieux, Bouton retour
func NewArtistPage(a models.Artist, onBack func()) *fyne.Container {
	// Titre principal
	title := widget.NewLabelWithStyle(
		a.Name,
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	// Informations de base
	info := widget.NewLabel(fmt.Sprintf(
		"Créé en %d\nPremier album: %s",
		a.CreationDate,
		a.FirstAlbum,
	))

	// Liste des membres
	membersText := "Membres:\n" + strings.Join(a.Members, "\n")
	members := widget.NewLabel(membersText)

	// Lieux de concert (si disponibles)
	var locations *widget.Label
	if len(a.Locations) > 0 {
		locationsText := "Lieux de concert:\n" + strings.Join(a.Locations, "\n")
		locations = widget.NewLabel(locationsText)
	} else {
		locations = widget.NewLabel("Aucun lieu de concert disponible")
	}

	// Dates de concert (si disponibles)
	var dates *widget.Label
	if len(a.ConcertDates) > 0 {
		datesText := "Dates de concert:\n" + strings.Join(a.ConcertDates, "\n")
		dates = widget.NewLabel(datesText)
	} else {
		dates = widget.NewLabel("Aucune date de concert disponible")
	}

	// Bouton retour
	back := widget.NewButton("← Retour", func() {
		onBack()
	})

	// TODO: Ajouter l'image de l'artiste (a.Image)

	// Construction de la page avec scroll
	content := container.NewVBox(
		title,
		widget.NewSeparator(),
		info,
		widget.NewSeparator(),
		members,
		widget.NewSeparator(),
		locations,
		widget.NewSeparator(),
		dates,
	)

	scrollContent := container.NewScroll(content)

	// Layout final avec bouton retour en haut
	return container.NewBorder(
		back,
		nil,
		nil,
		nil,
		scrollContent,
	)
}
