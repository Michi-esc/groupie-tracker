package ui

import (
	"fmt"
	"groupie-tracker/models"
<<<<<<< HEAD
	"image/color"
	"io"
	"net/http"
	"strconv"
=======
>>>>>>> daba17f1232b7d87c4e05e37aeb4df8d950f091a
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
	allLocations   []string
	onSelect       func(models.Artist)
	onShowMap      func()
	searchText     string
	grid           *fyne.Container
	searchDebounce *time.Timer

	// Filtres
	creationMin  int
	creationMax  int
	albumMin     int
	albumMax     int
	memberCounts map[int]bool // Pour stocker quels nombres de membres sont sélectionnés
	selectedLocs map[string]bool
}

// NewArtistList crée une nouvelle liste d'artistes
func NewArtistList(artists []models.Artist, onSelect func(models.Artist), onShowMap func()) *fyne.Container {
	list := &ArtistList{
<<<<<<< HEAD
		artists:      artists,
		onSelect:     onSelect,
		memberCounts: make(map[int]bool),
		selectedLocs: make(map[string]bool),
	}

	// Initialiser les valeurs min/max pour les filtres
	list.creationMin, list.creationMax = getCreationYearRange(artists)
	list.albumMin, list.albumMax = getFirstAlbumYearRange(artists)

	// Extraire toutes les locations uniques
	list.allLocations = extractAllLocations(artists)

	// Par défaut, tout est sélectionné
	for i := 1; i <= 8; i++ {
		list.memberCounts[i] = true
	}
	for _, loc := range list.allLocations {
		list.selectedLocs[loc] = true
	}

	// Variables pour stocker les widgets de filtres à mettre à jour
	var updateLocationChecks func(string)

	// Charger les locations de manière asynchrone
	go func() {
		locations, err := models.FetchLocations()
		if err == nil && locations != nil {
			// Enrichir les artistes avec leurs locations
			for i := range list.artists {
				for _, loc := range locations.Index {
					if loc.ID == list.artists[i].ID {
						list.artists[i].LocationsList = loc.Locations
						break
					}
				}
			}

			// Mettre à jour les locations disponibles
			list.allLocations = extractAllLocations(list.artists)

			// Initialiser tous les lieux comme sélectionnés
			for _, loc := range list.allLocations {
				list.selectedLocs[loc] = true
			}

			// Rafraîchir les checkboxes de lieux si elles existent
			if updateLocationChecks != nil {
				updateLocationChecks("")
			}

			// Rafraîchir la grille
			list.rebuildGrid()
		}
	}()

	// Barre de recherche
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("🔍 Rechercher un artiste ou membre...")
=======
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
>>>>>>> daba17f1232b7d87c4e05e37aeb4df8d950f091a
	searchEntry.OnChanged = func(text string) {
		list.searchText = strings.ToLower(text)
		if list.searchDebounce != nil {
			list.searchDebounce.Stop()
		}
		list.searchDebounce = time.AfterFunc(200*time.Millisecond, func() {
			list.rebuildGrid()
		})
	}

<<<<<<< HEAD
	// Panneau de filtres
	filterPanel, _, updateLocationChecksFunc := list.createFilterPanel()
	updateLocationChecks = updateLocationChecksFunc

	// Grille d'artistes - 4 par ligne
	grid := container.New(layout.NewGridLayout(4))
=======
	// Grille d'artistes - 4 colonnes
	grid := container.New(
		layout.NewGridLayout(4),
	)
>>>>>>> daba17f1232b7d87c4e05e37aeb4df8d950f091a
	list.grid = grid
	list.rebuildGrid()

	// Conteneur avec scroll
	scroll := container.NewScroll(grid)

<<<<<<< HEAD
	return container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("🎵 Groupie Tracker", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			searchEntry,
			filterPanel,
			widget.NewSeparator(),
=======
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
>>>>>>> daba17f1232b7d87c4e05e37aeb4df8d950f091a
		),
	)
}

// createFilterPanel crée le panneau de filtres
func (l *ArtistList) createFilterPanel() (*fyne.Container, *fyne.Container, func(string)) {
	// Filtre date de création
	creationLabel := widget.NewLabel(fmt.Sprintf("Date de création: %d - %d", l.creationMin, l.creationMax))
	creationMinEntry := widget.NewEntry()
	creationMinEntry.SetText(strconv.Itoa(l.creationMin))
	creationMaxEntry := widget.NewEntry()
	creationMaxEntry.SetText(strconv.Itoa(l.creationMax))

	creationApply := widget.NewButton("Appliquer", func() {
		min, _ := strconv.Atoi(creationMinEntry.Text)
		max, _ := strconv.Atoi(creationMaxEntry.Text)
		if min > 0 && max > 0 && min <= max {
			l.creationMin = min
			l.creationMax = max
			creationLabel.SetText(fmt.Sprintf("Date de création: %d - %d", min, max))
			l.rebuildGrid()
		}
	})

	creationFilter := container.NewVBox(
		creationLabel,
		container.NewGridWithColumns(3,
			widget.NewLabel("Min:"),
			creationMinEntry,
			widget.NewLabel(""),
		),
		container.NewGridWithColumns(3,
			widget.NewLabel("Max:"),
			creationMaxEntry,
			creationApply,
		),
	)

	// Filtre premier album
	albumLabel := widget.NewLabel(fmt.Sprintf("Premier album: %d - %d", l.albumMin, l.albumMax))
	albumMinEntry := widget.NewEntry()
	albumMinEntry.SetText(strconv.Itoa(l.albumMin))
	albumMaxEntry := widget.NewEntry()
	albumMaxEntry.SetText(strconv.Itoa(l.albumMax))

	albumApply := widget.NewButton("Appliquer", func() {
		min, _ := strconv.Atoi(albumMinEntry.Text)
		max, _ := strconv.Atoi(albumMaxEntry.Text)
		if min > 0 && max > 0 && min <= max {
			l.albumMin = min
			l.albumMax = max
			albumLabel.SetText(fmt.Sprintf("Premier album: %d - %d", min, max))
			l.rebuildGrid()
		}
	})

	albumFilter := container.NewVBox(
		albumLabel,
		container.NewGridWithColumns(3,
			widget.NewLabel("Min:"),
			albumMinEntry,
			widget.NewLabel(""),
		),
		container.NewGridWithColumns(3,
			widget.NewLabel("Max:"),
			albumMaxEntry,
			albumApply,
		),
	)

	// Filtre nombre de membres (checkboxes)
	memberChecks := make([]*widget.Check, 0)
	memberContainer := container.NewVBox()

	for i := 1; i <= 8; i++ {
		count := i
		check := widget.NewCheck(fmt.Sprintf("%d membre(s)", count), func(checked bool) {
			l.memberCounts[count] = checked
			l.rebuildGrid()
		})
		check.Checked = true
		memberChecks = append(memberChecks, check)
		memberContainer.Add(check)
	}

	// Filtre lieux (checkboxes avec recherche)
	locationSearch := widget.NewEntry()
	locationSearch.SetPlaceHolder("Rechercher un lieu...")

	locationChecks := container.NewVBox()
	updateLocationChecks := func(filter string) {
		locationChecks.Objects = nil
		filter = strings.ToLower(filter)

		// Parcourir tous les lieux et créer les checkboxes
		for _, loc := range l.allLocations {
			if filter != "" && !strings.Contains(strings.ToLower(loc), filter) {
				continue
			}

			locCopy := loc
			// Vérifier l'état actuel dans selectedLocs
			isChecked := l.selectedLocs[locCopy]

			check := widget.NewCheck(loc, func(checked bool) {
				l.selectedLocs[locCopy] = checked
				l.rebuildGrid()
			})
			check.Checked = isChecked
			locationChecks.Add(check)
		}
		locationChecks.Refresh()
	}

	locationSearch.OnChanged = func(text string) {
		updateLocationChecks(text)
	}
	updateLocationChecks("")

	locationScroll := container.NewScroll(locationChecks)
	locationScroll.SetMinSize(fyne.NewSize(300, 150))

	locationFilter := container.NewVBox(
		locationSearch,
		locationScroll,
	)

	// Bouton réinitialiser
	resetBtn := widget.NewButton("🔄 Réinitialiser tous les filtres", func() {
		// Réinitialiser dates
		minC, maxC := getCreationYearRange(l.artists)
		minA, maxA := getFirstAlbumYearRange(l.artists)
		l.creationMin = minC
		l.creationMax = maxC
		l.albumMin = minA
		l.albumMax = maxA

		creationMinEntry.SetText(strconv.Itoa(minC))
		creationMaxEntry.SetText(strconv.Itoa(maxC))
		albumMinEntry.SetText(strconv.Itoa(minA))
		albumMaxEntry.SetText(strconv.Itoa(maxA))
		creationLabel.SetText(fmt.Sprintf("Date de création: %d - %d", minC, maxC))
		albumLabel.SetText(fmt.Sprintf("Premier album: %d - %d", minA, maxA))

		// Réinitialiser membres
		for i := 1; i <= 8; i++ {
			l.memberCounts[i] = true
		}
		for _, check := range memberChecks {
			check.Checked = true
			check.Refresh()
		}

		// Réinitialiser lieux
		for _, loc := range l.allLocations {
			l.selectedLocs[loc] = true
		}
		locationSearch.SetText("")
		updateLocationChecks("")

		l.rebuildGrid()
	})
	resetBtn.Importance = widget.HighImportance

	// Accordion pour organiser les filtres
	accordion := widget.NewAccordion(
		widget.NewAccordionItem("📅 Date de création", creationFilter),
		widget.NewAccordionItem("💿 Année premier album", albumFilter),
		widget.NewAccordionItem("👥 Nombre de membres", memberContainer),
		widget.NewAccordionItem("📍 Lieux de concerts", locationFilter),
	)

	return container.NewVBox(
		accordion,
		resetBtn,
	), locationChecks, updateLocationChecks
}

// rebuildGrid met à jour la grille selon les filtres
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

// filteredArtists retourne les artistes filtrés
func (l *ArtistList) filteredArtists() []models.Artist {
	res := make([]models.Artist, 0, len(l.artists))
	st := l.searchText

	for _, a := range l.artists {
		// Filtre de recherche textuelle
		matchesSearch := st == "" || strings.Contains(strings.ToLower(a.Name), st)
		if !matchesSearch {
			// Chercher dans les membres
			for _, m := range a.Members {
				if strings.Contains(strings.ToLower(m), st) {
					matchesSearch = true
					break
				}
			}
		}
		if !matchesSearch {
			continue
		}

		// Filtre date de création
		if a.CreationDate < l.creationMin || a.CreationDate > l.creationMax {
			continue
		}

		// Filtre année premier album
		albumYear := extractAlbumYear(a.FirstAlbum)
		if albumYear > 0 && (albumYear < l.albumMin || albumYear > l.albumMax) {
			continue
		}

		// Filtre nombre de membres
		memberCount := len(a.Members)
		if !l.memberCounts[memberCount] {
			continue
		}

		// Filtre lieux
		matchesLocation := false
		if len(a.LocationsList) == 0 {
			matchesLocation = true // Pas de lieux = on affiche
		} else {
			for _, loc := range a.LocationsList {
				normalized := normalizeLocation(loc)
				if l.selectedLocs[normalized] {
					matchesLocation = true
					break
				}
			}
		}
		if !matchesLocation {
			continue
		}

		res = append(res, a)
	}
	return res
}

// Fonctions utilitaires pour les filtres
func getCreationYearRange(artists []models.Artist) (int, int) {
	if len(artists) == 0 {
		return 1950, 2024
	}
	min, max := artists[0].CreationDate, artists[0].CreationDate
	for _, a := range artists {
		if a.CreationDate < min {
			min = a.CreationDate
		}
		if a.CreationDate > max {
			max = a.CreationDate
		}
	}
	return min, max
}

func getFirstAlbumYearRange(artists []models.Artist) (int, int) {
	min, max := 3000, 0
	for _, a := range artists {
		year := extractAlbumYear(a.FirstAlbum)
		if year > 0 {
			if year < min {
				min = year
			}
			if year > max {
				max = year
			}
		}
	}
	if min == 3000 {
		min = 1950
	}
	if max == 0 {
		max = 2024
	}
	return min, max
}

func extractAlbumYear(dateStr string) int {
	// Format attendu: "DD-MM-YYYY"
	parts := strings.Split(dateStr, "-")
	if len(parts) == 3 {
		year, err := strconv.Atoi(parts[2])
		if err == nil {
			return year
		}
	}
	return 0
}

func extractAllLocations(artists []models.Artist) []string {
	locationSet := make(map[string]bool)
	for _, a := range artists {
		for _, loc := range a.LocationsList {
			normalized := normalizeLocation(loc)
			locationSet[normalized] = true
		}
	}

	locations := make([]string, 0, len(locationSet))
	for loc := range locationSet {
		locations = append(locations, loc)
	}
	return locations
}

func normalizeLocation(loc string) string {
	// Convertir "city-state-country" en "City, State, Country"
	parts := strings.Split(loc, "-")
	normalized := make([]string, len(parts))
	for i, p := range parts {
		p = strings.TrimSpace(p)
		if len(p) > 0 {
			normalized[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
		}
	}
	return strings.Join(normalized, ", ")
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
