package ui

import (
	"fmt"
	"groupie-tracker/models"
	"strings"
	"time"

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
	artists        []models.Artist
	onSelect       func(models.Artist)
	onShowMap      func()
	searchText     string
	grid           *fyne.Container
	searchDebounce *time.Timer
}

// NewArtistList crée une nouvelle liste d'artistes
func NewArtistList(artists []models.Artist, onSelect func(models.Artist), onShowMap func()) *fyne.Container {
	list := &ArtistList{
		artists:   artists,
		onSelect:  onSelect,
		onShowMap: onShowMap,
	}

	// === HEADER ===
	titleText := canvas.NewText("🎵 Groupie Tracker", TextWhite)
	titleText.TextStyle = fyne.TextStyle{Bold: true}
	titleText.TextSize = 36
	titleText.Alignment = fyne.TextAlignCenter

	subtitleText := canvas.NewText("Découvrez vos artistes musicaux préférés", TextLight)
	subtitleText.TextSize = 14
	subtitleText.Alignment = fyne.TextAlignCenter

	// Bouton pour voir la carte
	mapButton := widget.NewButton("🗺️ Voir la carte des concerts", list.onShowMap)
	mapButton.Importance = widget.HighImportance

	headerBg := canvas.NewRectangle(BgDarker)
	header := container.NewMax(
		headerBg,
		container.NewVBox(
			widget.NewLabel(""), // Spacer
			container.NewCenter(titleText),
			container.NewCenter(subtitleText),
			container.NewCenter(mapButton),
			widget.NewLabel(""), // Spacer
		),
	)

	// === BARRE DE RECHERCHE ===
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("🔍 Rechercher un artiste ou un membre...")
	searchEntry.OnChanged = func(text string) {
		list.searchText = strings.ToLower(text)
		if list.searchDebounce != nil {
			list.searchDebounce.Stop()
		}
		list.searchDebounce = time.AfterFunc(200*time.Millisecond, func() {
			list.rebuildGrid()
		})
	}

	// Grille d'artistes - 4 colonnes
	grid := container.New(
		layout.NewGridLayout(4),
	)
	list.grid = grid
	list.rebuildGrid()

	// Conteneur avec scroll
	scroll := container.NewScroll(grid)

	// Créer un container pour la recherche avec fond
	searchContainer := container.NewVBox(searchEntry)

	// Layout avec BorderLayout - header en haut, search juste après, scroll au centre
	return container.New(
		layout.NewBorderLayout(header, nil, nil, nil),
		header,
		container.New(
			layout.NewBorderLayout(searchContainer, nil, nil, nil),
			searchContainer,
			scroll,
		),
	)
}

// rebuildGrid met à jour la grille selon le texte de recherche
func (l *ArtistList) rebuildGrid() {
	if l.grid == nil {
		return
	}
	l.grid.Objects = []fyne.CanvasObject{}
	filteredList := l.filteredArtists()

	if len(filteredList) == 0 {
		// Message "aucun résultat"
		msgText := canvas.NewText("Aucun résultat trouvé", TextLight)
		msgText.TextSize = 16
		msgText.Alignment = fyne.TextAlignCenter
		l.grid.Add(container.NewCenter(msgText))
	} else {
		for _, artist := range filteredList {
			card := createArtistCard(artist, l.onSelect)
			l.grid.Add(card)
		}
	}
	l.grid.Refresh()
}

// filteredArtists retourne les artistes filtrés par nom
func (l *ArtistList) filteredArtists() []models.Artist {
	if l.searchText == "" {
		return l.artists
	}
	st := l.searchText
	res := make([]models.Artist, 0, len(l.artists))
	for _, a := range l.artists {
		if strings.Contains(strings.ToLower(a.Name), st) {
			res = append(res, a)
			continue
		}
		// Optionnel: filtrer aussi sur les membres
		for _, m := range a.Members {
			if strings.Contains(strings.ToLower(m), st) {
				res = append(res, a)
				break
			}
		}
	}
	return res
}

func createArtistCard(artist models.Artist, onSelect func(models.Artist)) *fyne.Container {
	// Image de l'artiste (chargement simple)
	uri, _ := storage.ParseURI(artist.Image)
	img := canvas.NewImageFromURI(uri)
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(220, 200))

	// Nom de l'artiste
	nameText := canvas.NewText(artist.Name, TextWhite)
	nameText.TextStyle = fyne.TextStyle{Bold: true}
	nameText.TextSize = 16
	nameText.Alignment = fyne.TextAlignCenter

	// Informations
	infoMembers := canvas.NewText(fmt.Sprintf("👥 %d membres", len(artist.Members)), TextLight)
	infoMembers.TextSize = 12
	infoMembers.Alignment = fyne.TextAlignCenter

	infoCreated := canvas.NewText(fmt.Sprintf("🎸 Créé en %d", artist.CreationDate), TextLight)
	infoCreated.TextSize = 12
	infoCreated.Alignment = fyne.TextAlignCenter

	// Bouton "Voir les détails"
	btn := widget.NewButton("→ Voir les détails", func() {
		onSelect(artist)
	})
	btn.Importance = widget.HighImportance

	// Layout de la carte
	cardContent := container.NewVBox(
		img,
		widget.NewLabel(""), // Spacer
		container.NewCenter(nameText),
		container.NewCenter(infoMembers),
		container.NewCenter(infoCreated),
		widget.NewLabel(""), // Spacer
		btn,
	)

	// Fond de la carte avec bordure
	cardBg := canvas.NewRectangle(CardBg)
	cardBgBorder := canvas.NewRectangle(AccentCyan)
	cardBgBorder.StrokeWidth = 2

	// Card container avec padding
	return container.New(
		layout.NewMaxLayout(),
		cardBgBorder,
		cardBg,
		container.NewPadded(cardContent),
	)
}
