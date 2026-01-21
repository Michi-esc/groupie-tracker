package ui

import (
	"fmt"
	"groupie-tracker/models"
	"log"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

// widget liste artistes
type ArtistList struct {
	widget.BaseWidget
	artists        []models.Artist
	allLocations   []string
	onSelect       func(models.Artist)
	onShowMap      func()
	searchText     string
	grid           *fyne.Container
	searchDebounce *time.Timer

	// filtres mémorisés
	creationMin  int
	creationMax  int
	albumMin     int
	albumMax     int
	memberCounts map[int]bool
	selectedLocs map[string]bool
}

// build liste artistes
func NewArtistList(artists []models.Artist, onSelect func(models.Artist), onShowMap func()) *fyne.Container {
	return NewArtistListWithWindow(nil, artists, onSelect, onShowMap)
}

// build liste artistes avec window pour bouton langue
func NewArtistListWithWindow(win *Window, artists []models.Artist, onSelect func(models.Artist), onShowMap func()) *fyne.Container {
	list := &ArtistList{
		artists: artists, onShowMap: onShowMap, onSelect: onSelect,
		memberCounts: make(map[int]bool),
		selectedLocs: make(map[string]bool),
	}

	// bornes min/max pour filtres
	list.creationMin, list.creationMax = getCreationYearRange(artists)
	list.albumMin, list.albumMax = getFirstAlbumYearRange(artists)

	// récupère les lieux uniques
	list.allLocations = extractAllLocations(artists)

	// par défaut on coche tout
	for i := 1; i <= 8; i++ {
		list.memberCounts[i] = true
	}
	// on ajoute les lieux initiaux
	for _, loc := range list.allLocations {
		list.selectedLocs[loc] = true
	}
	// on coche tout pour ne rien filtrer au début, on ajustera après enrichissement

	// on garde le callback pour rafraîchir les cases
	var updateLocationChecks func(string)

	// charge les relations en async
	go func() {
		relations, err := models.FetchRelations()
		if err != nil {
			// si ça rate on continue quand même
			log.Println("Erreur lors du chargement des relations:", err)
		} else if relations != nil {
			// enrichit chaque artiste avec ses lieux
			for i := range list.artists {
				for _, rel := range relations.Index {
					if rel.ID == list.artists[i].ID {
						// extrait les lieux uniques depuis datesLocations
						locSet := make(map[string]bool)
						for loc := range rel.DatesLocations {
							locSet[loc] = true
						}
						for loc := range locSet {
							list.artists[i].LocationsList = append(list.artists[i].LocationsList, loc)
						}
						break
					}
				}
			}

			// on met à jour la liste des lieux
			list.allLocations = extractAllLocations(list.artists)

			// on coche tous les lieux
			for _, loc := range list.allLocations {
				list.selectedLocs[loc] = true
			}

			// rafraîchit les cases si déjà affichées
			if updateLocationChecks != nil {
				updateLocationChecks("")
			}
		}

		// on rafraîchit la grille une fois
		fyne.Do(func() {
			list.rebuildGrid()
		})
	}()

	// barre de recherche
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder(T().SearchPlaceholder)
	searchEntry.OnChanged = func(text string) {
		list.searchText = strings.ToLower(text)
		if list.searchDebounce != nil {
			list.searchDebounce.Stop()
		}
		list.searchDebounce = time.AfterFunc(200*time.Millisecond, func() {
			fyne.Do(func() {
				list.rebuildGrid()
			})
		})
	}

	// panneau de filtres
	filterPanel, _, updateLocationChecksFunc := list.createFilterPanel()
	updateLocationChecks = updateLocationChecksFunc

	// grille d'artistes (4 colonnes)
	grid := container.New(layout.NewGridLayout(4))
	list.grid = grid

	// on construit la grille dès le départ
	list.rebuildGrid()

	// conteneur scroll
	scroll := container.NewScroll(grid)

	// bouton pour ouvrir la carte
	mapButton := widget.NewButton(T().ShowMap, list.onShowMap)
	mapButton.Importance = widget.HighImportance

	// barre de boutons
	topButtons := container.NewHBox()
	if win != nil && win.LangButton != nil {
		topButtons.Add(win.LangButton)
	}
	topButtons.Add(mapButton)

	return container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle(T().WindowTitle, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			container.NewCenter(topButtons),
			searchEntry,
			filterPanel,
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		scroll,
	)
}

// panneau filtres
func (l *ArtistList) createFilterPanel() (*fyne.Container, *fyne.Container, func(string)) {
	// filtre date de création
	creationLabel := widget.NewLabel(fmt.Sprintf(T().CreationYear+": %d - %d", l.creationMin, l.creationMax))
	creationMinEntry := widget.NewEntry()
	creationMinEntry.SetText(strconv.Itoa(l.creationMin))
	creationMaxEntry := widget.NewEntry()
	creationMaxEntry.SetText(strconv.Itoa(l.creationMax))

	creationApply := widget.NewButton(T().Filters, func() {
		min, _ := strconv.Atoi(creationMinEntry.Text)
		max, _ := strconv.Atoi(creationMaxEntry.Text)
		if min > 0 && max > 0 && min <= max {
			l.creationMin = min
			l.creationMax = max
			creationLabel.SetText(fmt.Sprintf(T().CreationYear+": %d - %d", min, max))
			l.rebuildGrid()
		}
	})

	creationFilter := container.NewVBox(
		creationLabel,
		container.NewGridWithColumns(3,
			widget.NewLabel(T().Min+":"),
			creationMinEntry,
			widget.NewLabel(""),
		),
		container.NewGridWithColumns(3,
			widget.NewLabel(T().Max+":"),
			creationMaxEntry,
			creationApply,
		),
	)

	// filtre sur l'année du premier album
	albumLabel := widget.NewLabel(fmt.Sprintf(T().FirstAlbum+": %d - %d", l.albumMin, l.albumMax))
	albumMinEntry := widget.NewEntry()
	albumMinEntry.SetText(strconv.Itoa(l.albumMin))
	albumMaxEntry := widget.NewEntry()
	albumMaxEntry.SetText(strconv.Itoa(l.albumMax))

	albumApply := widget.NewButton(T().Filters, func() {
		min, _ := strconv.Atoi(albumMinEntry.Text)
		max, _ := strconv.Atoi(albumMaxEntry.Text)
		if min > 0 && max > 0 && min <= max {
			l.albumMin = min
			l.albumMax = max
			albumLabel.SetText(fmt.Sprintf(T().FirstAlbum+": %d - %d", min, max))
			l.rebuildGrid()
		}
	})

	albumFilter := container.NewVBox(
		albumLabel,
		container.NewGridWithColumns(3,
			widget.NewLabel(T().Min+":"),
			albumMinEntry,
			widget.NewLabel(""),
		),
		container.NewGridWithColumns(3,
			widget.NewLabel(T().Max+":"),
			albumMaxEntry,
			albumApply,
		),
	)

	// filtre par nombre de membres
	memberChecks := make([]*widget.Check, 0)
	memberContainer := container.NewVBox()

	for i := 1; i <= 8; i++ {
		count := i
		check := widget.NewCheck(fmt.Sprintf("%d "+T().Members, count), func(checked bool) {
			l.memberCounts[count] = checked
			l.rebuildGrid()
		})
		check.Checked = true
		memberChecks = append(memberChecks, check)
		memberContainer.Add(check)
	}

	// filtre des lieux (checkbox + recherche)
	locationSearch := widget.NewEntry()
	locationSearch.SetPlaceHolder(T().Search + " " + T().Location + "...")

	locationChecks := container.NewVBox()
	updateLocationChecks := func(filter string) {
		locationChecks.Objects = nil
		filter = strings.ToLower(filter)

		// parcourt les lieux et ajoute les cases
		for _, loc := range l.allLocations {
			if filter != "" && !strings.Contains(strings.ToLower(loc), filter) {
				continue
			}

			locCopy := loc
			// on garde l'état déjà coché
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

	// bouton pour tout réinitialiser
	resetBtn := widget.NewButton("🔄 "+T().ResetFilters, func() {
		// reset des dates
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
		creationLabel.SetText(fmt.Sprintf(T().CreationYear+": %d - %d", minC, maxC))
		albumLabel.SetText(fmt.Sprintf(T().FirstAlbum+": %d - %d", minA, maxA))

		// reset des membres
		for i := 1; i <= 8; i++ {
			l.memberCounts[i] = true
		}
		for _, check := range memberChecks {
			check.Checked = true
			check.Refresh()
		}

		// reset des lieux
		for _, loc := range l.allLocations {
			l.selectedLocs[loc] = true
		}
		locationSearch.SetText("")
		updateLocationChecks("")

		l.rebuildGrid()
	})
	resetBtn.Importance = widget.HighImportance

	// accordion pour ranger les filtres
	accordion := widget.NewAccordion(
		widget.NewAccordionItem(T().CreationYear, creationFilter),
		widget.NewAccordionItem("💿 "+T().FirstAlbum, albumFilter),
		widget.NewAccordionItem("👥 "+T().Members, memberContainer),
		widget.NewAccordionItem(T().Location, locationFilter),
	)

	return container.NewVBox(
		accordion,
		resetBtn,
	), locationChecks, updateLocationChecks
}

// rebuild grille
func (l *ArtistList) rebuildGrid() {
	if l.grid == nil {
		return
	}
	l.grid.Objects = []fyne.CanvasObject{}
	filteredList := l.filteredArtists()

	if len(filteredList) == 0 {
		// message quand rien ne correspond
		msgText := widget.NewLabel(T().NoResults)
		msgText.Alignment = fyne.TextAlignCenter
		l.grid.Add(msgText)
	} else {
		for _, artist := range filteredList {
			card := createArtistCard(artist, l.onSelect)
			l.grid.Add(card)
		}
	}
	l.grid.Refresh()
}

// artistes filtrés
func (l *ArtistList) filteredArtists() []models.Artist {
	res := make([]models.Artist, 0, len(l.artists))
	st := l.searchText
	seen := make(map[int]bool) // évite les doublons

	for _, a := range l.artists {
		// évite d'ajouter deux fois le même
		if seen[a.ID] {
			continue
		}

		// filtre sur le texte recherché
		matchesSearch := st == "" || strings.Contains(strings.ToLower(a.Name), st)
		if !matchesSearch {
			// on cherche aussi dans les membres
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

		// filtre sur l'année de création
		if a.CreationDate < l.creationMin || a.CreationDate > l.creationMax {
			continue
		}

		// filtre sur l'année du premier album
		albumYear := extractAlbumYear(a.FirstAlbum)
		if albumYear > 0 && (albumYear < l.albumMin || albumYear > l.albumMax) {
			continue
		}

		// filtre sur le nombre de membres
		memberCount := len(a.Members)
		if !l.memberCounts[memberCount] {
			continue
		}

		// filtre sur les lieux
		matchesLocation := false
		if len(a.LocationsList) == 0 {
			matchesLocation = true // pas de lieu -> on affiche
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

		seen[a.ID] = true
		res = append(res, a)
	}
	return res
}

// utils filtres
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
	// format attendu "DD-MM-YYYY"
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
	// convertit "city-state-country" en "City, State, Country"
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
	// image de l'artiste
	uri, _ := storage.ParseURI(artist.Image)
	img := canvas.NewImageFromURI(uri)
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(200, 200))

	// nom centré sous l'image (color choisi selon contraste)
	captionBg := canvas.NewRectangle(CardBgLight)
	nameText := canvas.NewText(artist.Name, ContrastColor(CardBgLight))
	nameText.TextStyle = fyne.TextStyle{Bold: true}
	nameText.Alignment = fyne.TextAlignCenter

	// petit fond sous le nom
	caption := container.NewMax(
		captionBg,
		container.NewPadded(container.NewCenter(nameText)),
	)

	// infos rapides (texte rendu selon contraste du fond de la carte)
	members := canvas.NewText(fmt.Sprintf("%d "+T().Members, len(artist.Members)), ContrastColor(CardBg))
	members.Alignment = fyne.TextAlignCenter

	created := canvas.NewText(fmt.Sprintf(T().Created, artist.CreationDate), ContrastColor(CardBg))
	created.Alignment = fyne.TextAlignCenter

	// bouton pour ouvrir la fiche
	btn := widget.NewButton(T().ShowDetails, func() {
		onSelect(artist)
	})
	btn.Importance = widget.HighImportance

	// container de la carte
	card := container.NewVBox(
		img,
		caption,
		members,
		created,
		container.NewCenter(btn),
	)

	// fond avec bordure (utilise couleurs sombres pour lisibilité)
	bg := canvas.NewRectangle(CardBg)

	return container.New(
		layout.NewMaxLayout(),
		bg,
		container.NewPadded(card),
	)
}
