package ui

import (
	"fmt"
	"groupie-tracker/models"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

// ArtistList affiche la liste des artistes
type ArtistList struct {
	widget.BaseWidget
	artists    []models.Artist
	onSelect   func(models.Artist)
	searchText string
}

// NewArtistList crée une nouvelle liste d'artistes
func NewArtistList(artists []models.Artist, onSelect func(models.Artist)) *fyne.Container {
	list := &ArtistList{
		artists:  artists,
		onSelect: onSelect,
	}

	// Barre de recherche
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("🔍 Rechercher un artiste...")
	searchEntry.OnChanged = func(text string) {
		list.searchText = strings.ToLower(text)
		// Rafraîchir l'affichage
	}

	// Grille d'artistes
	grid := container.NewGridWrap(
		fyne.NewSize(250, 350),
	)

	for _, artist := range artists {
		card := createArtistCard(artist, onSelect)
		grid.Add(card)
	}

	// Conteneur avec scroll
	scroll := container.NewScroll(grid)

	return container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("🎵 Groupie Tracker", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			searchEntry,
			widget.NewSeparator(),
		),
		nil, nil, nil,
		scroll,
	)
}

func createArtistCard(artist models.Artist, onSelect func(models.Artist)) *fyne.Container {
	// Image de l'artiste
	uri, _ := storage.ParseURI(artist.Image)
	img := canvas.NewImageFromURI(uri)
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(200, 200))

	// Nom de l'artiste
	name := widget.NewLabelWithStyle(artist.Name, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	// Informations
	members := widget.NewLabel(fmt.Sprintf("%d membres", len(artist.Members)))
	members.Alignment = fyne.TextAlignCenter

	created := widget.NewLabel(fmt.Sprintf("Créé en %d", artist.CreationDate))
	created.Alignment = fyne.TextAlignCenter

	// Bouton pour voir les détails
	btn := widget.NewButton("Voir les détails", func() {
		onSelect(artist)
	})

	// Card container
	card := container.NewVBox(
		img,
		name,
		members,
		created,
		btn,
	)

	// Fond avec bordure
	bg := canvas.NewRectangle(color.RGBA{R: 240, G: 240, B: 240, A: 255})

	return container.New(
		layout.NewMaxLayout(),
		bg,
		container.NewPadded(card),
	)
}
