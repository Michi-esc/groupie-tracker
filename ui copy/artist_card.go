package ui

import (
	"fmt"
	"groupie-tracker/models"

	"fyne.io/fyne/v2/widget"
)

// NewArtistCard crée une carte cliquable pour un artiste
// Affiche : Image, Nom, Année de création
func NewArtistCard(a models.Artist, onSelect func(models.Artist)) *widget.Card {
	title := a.Name
	subtitle := fmt.Sprintf("Créé en %d", a.CreationDate)

	// TODO: Ajouter l'image de l'artiste (a.Image)
	// Pour l'instant, carte simple avec titre et sous-titre

	card := widget.NewCard(title, subtitle, nil)

	// Rendre la carte cliquable
	card.SetContent(widget.NewButton("Voir détails", func() {
		onSelect(a)
	}))

	return card
}
