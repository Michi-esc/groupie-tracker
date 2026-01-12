package ui

import (
	"fmt"
	"groupie-tracker/models"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

// ArtistPage affiche les détails d'un artiste
func NewArtistPage(artist models.Artist, onBack func()) fyne.CanvasObject {
	// Bouton retour
	backBtn := widget.NewButton("← Retour", onBack)

	// Image de l'artiste
	uri, _ := storage.ParseURI(artist.Image)
	img := canvas.NewImageFromURI(uri)
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(300, 300))

	// Nom de l'artiste
	title := widget.NewLabelWithStyle(artist.Name, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	// Informations de base
	infoText := fmt.Sprintf(`
📅 Année de création: %d
🎤 Nombre de membres: %d
💿 Premier album: %s

👥 Membres:
%s
`,
		artist.CreationDate,
		len(artist.Members),
		artist.FirstAlbum,
		strings.Join(artist.Members, "\n"),
	)

	info := widget.NewLabel(infoText)
	info.Wrapping = fyne.TextWrapWord

	// Section concerts (peut être complétée avec les données de l'API)
	concertsLabel := widget.NewLabelWithStyle("🎸 Concerts", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	concertsInfo := widget.NewLabel("Chargement des informations de concerts...")

	// Contenu principal
	content := container.NewVBox(
		backBtn,
		widget.NewSeparator(),
		container.NewCenter(img),
		title,
		widget.NewSeparator(),
		info,
		widget.NewSeparator(),
		concertsLabel,
		concertsInfo,
	)

	scroll := container.NewScroll(content)
	return scroll
}
