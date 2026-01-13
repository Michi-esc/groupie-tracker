package ui

import (
	"fmt"
	"groupie-tracker/models"
	"image/color"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// ArtistList affiche la liste des artistes
type ArtistList struct {
	widget.BaseWidget
	artists        []models.Artist
	onSelect       func(models.Artist)
	searchText     string
	grid           *fyne.Container
	searchDebounce *time.Timer
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
		if list.searchDebounce != nil {
			list.searchDebounce.Stop()
		}
		list.searchDebounce = time.AfterFunc(200*time.Millisecond, func() {
			// Mise à jour après un court délai pour limiter les rafraîchissements
			list.rebuildGrid()
		})
	}

	// Grille d'artistes - 4 par ligne avec alignement correct
	grid := container.New(
		layout.NewGridLayout(4), // Force 4 colonnes
	)
	list.grid = grid

	// Remplir la grille initialement
	list.rebuildGrid()

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

// rebuildGrid met à jour la grille selon le texte de recherche
func (l *ArtistList) rebuildGrid() {
	if l.grid == nil {
		return
	}
	// Réinitialiser les objets
	l.grid.Objects = []fyne.CanvasObject{}
	for _, artist := range l.filteredArtists() {
		card := createArtistCard(artist, l.onSelect)
		l.grid.Add(card) // Pas de padding supplémentaire pour un alignement correct
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
	// Image de l'artiste (chargement asynchrone avec cache)
	img := canvas.NewImageFromResource(nil)
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(200, 200))
	loadImageAsync(img, artist.Image)

	// Nom de l'artiste en noir et centré sous l'image
	nameText := canvas.NewText(artist.Name, color.Black)
	nameText.TextStyle = fyne.TextStyle{Bold: true}
	nameText.Alignment = fyne.TextAlignCenter
	// Afficher le nom juste sous l'image avec un fond discret
	captionBg := canvas.NewRectangle(color.RGBA{R: 235, G: 235, B: 235, A: 255})
	caption := container.NewMax(
		captionBg,
		container.NewPadded(container.NewCenter(nameText)),
	)

	// Informations
	members := widget.NewLabel(fmt.Sprintf("%d membres", len(artist.Members)))
	members.Alignment = fyne.TextAlignCenter

	created := widget.NewLabel(fmt.Sprintf("Créé en %d", artist.CreationDate))
	created.Alignment = fyne.TextAlignCenter

	// Bouton pour voir les détails (plus grand et mis en avant)
	btn := widget.NewButton("Voir les détails", func() {
		onSelect(artist)
	})
	btn.Importance = widget.HighImportance
	// Adapter la taille du bouton pour correspondre aux pixels de la carte
	btnBox := container.NewGridWrap(fyne.NewSize(220, 44), btn)

	// Card container
	card := container.NewVBox(
		img,
		caption,
		members,
		created,
		container.NewCenter(btnBox),
	)

	// Fond avec bordure
	bg := canvas.NewRectangle(color.RGBA{R: 240, G: 240, B: 240, A: 255})

	return container.New(
		layout.NewMaxLayout(),
		bg,
		container.NewPadded(card),
	)
}

// --- Chargement d'images optimisé (cache + async) ---
var imgCacheMu sync.Mutex
var imageCache = map[string]fyne.Resource{}

func loadImageAsync(img *canvas.Image, url string) {
	if res := getCachedResource(url); res != nil {
		img.Resource = res
		img.Refresh()
		return
	}
	go func() {
		res, err := fetchImageResource(url)
		if err != nil || res == nil {
			return
		}
		// Mettre à jour directement l'image après chargement
		img.Resource = res
		img.Refresh()
	}()
}

func getCachedResource(url string) fyne.Resource {
	imgCacheMu.Lock()
	defer imgCacheMu.Unlock()
	return imageCache[url]
}

func setCachedResource(url string, res fyne.Resource) {
	imgCacheMu.Lock()
	imageCache[url] = res
	imgCacheMu.Unlock()
}

func fetchImageResource(url string) (fyne.Resource, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("image HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	res := fyne.NewStaticResource(url, data)
	setCachedResource(url, res)
	return res, nil
}
