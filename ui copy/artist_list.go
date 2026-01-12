package ui

import (
	"groupie-tracker/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

// NewArtistList crée une grille scrollable d'artistes
// onSelect est appelé quand l'utilisateur clique sur une carte
func NewArtistList(artists []models.Artist, onSelect func(models.Artist)) *container.Scroll {
	grid := container.NewGridWrap(
		fyne.NewSize(220, 280),
	)

	for _, artist := range artists {
		card := NewArtistCard(artist, onSelect)
		grid.Add(card)
	}

	return container.NewScroll(grid)
}
