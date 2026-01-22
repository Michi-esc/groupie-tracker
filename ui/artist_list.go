package ui

import (
	"fmt"
	"groupie-tracker/models"
	"image/color"
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

type ArtistList struct {
	widget.BaseWidget
	artists        []models.Artist
	allLocations   []string
	onSelect       func(models.Artist)
	onShowMap      func()
	searchText     string
	grid           *fyne.Container
	searchDebounce *time.Timer

	creationMin  int
	creationMax  int
	albumMin     int
	albumMax     int
	memberCounts map[int]bool
	selectedLocs map[string]bool
}

func NewArtistList(artists []models.Artist, onSelect func(models.Artist), onShowMap func()) *fyne.Container {
	list := &ArtistList{
		artists: artists, onShowMap: onShowMap, onSelect: onSelect,
		memberCounts: make(map[int]bool),
		selectedLocs: make(map[string]bool),
	}

	list.creationMin, list.creationMax = getCreationYearRange(artists)
	list.albumMin, list.albumMax = getFirstAlbumYearRange(artists)

	list.allLocations = extractAllLocations(artists)

	for i := 1; i <= 8; i++ {
		list.memberCounts[i] = true
	}
	for _, loc := range list.allLocations {
		list.selectedLocs[loc] = true
	}

	var updateLocationChecks func(string)

	go func() {
		relations, err := models.FetchRelations()
		if err != nil {
			log.Println("Erreur lors du chargement des relations:", err)
		} else if relations != nil {
			for i := range list.artists {
				for _, rel := range relations.Index {
					if rel.ID == list.artists[i].ID {
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

			list.allLocations = extractAllLocations(list.artists)

			for _, loc := range list.allLocations {
				list.selectedLocs[loc] = true
			}

			if updateLocationChecks != nil {
				updateLocationChecks("")
			}
		}

		fyne.Do(func() {
			list.rebuildGrid()
		})
	}()

	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Search artists or members...")
	searchEntry.OnChanged = func(text string) {
		list.searchText = strings.ToLower(text)
		if list.searchDebounce != nil {
			list.searchDebounce.Stop()
		}
		list.searchDebounce = time.AfterFunc(200*time.Millisecond, func() {
			list.rebuildGrid()
		})
	}

	filterPanel, _, updateLocationChecksFunc := list.createFilterPanel()
	updateLocationChecks = updateLocationChecksFunc

	grid := container.New(layout.NewGridLayout(4))
	list.grid = grid

	list.rebuildGrid()

	scroll := container.NewScroll(grid)

	mapButton := widget.NewButton("Map", list.onShowMap)
	mapButton.Importance = widget.HighImportance

	return container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("GROUPIE TRACKER", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			container.NewCenter(mapButton),
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

func (l *ArtistList) createFilterPanel() (*fyne.Container, *fyne.Container, func(string)) {
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

	locationSearch := widget.NewEntry()
	locationSearch.SetPlaceHolder("Rechercher un lieu...")

	locationChecks := container.NewVBox()
	updateLocationChecks := func(filter string) {
		locationChecks.Objects = nil
		filter = strings.ToLower(filter)

		for _, loc := range l.allLocations {
			if filter != "" && !strings.Contains(strings.ToLower(loc), filter) {
				continue
			}

			locCopy := loc
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

	resetBtn := widget.NewButton("Reinitialiser tous les filtres", func() {
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

		for i := 1; i <= 8; i++ {
			l.memberCounts[i] = true
		}
		for _, check := range memberChecks {
			check.Checked = true
			check.Refresh()
		}

		for _, loc := range l.allLocations {
			l.selectedLocs[loc] = true
		}
		locationSearch.SetText("")
		updateLocationChecks("")

		l.rebuildGrid()
	})
	resetBtn.Importance = widget.HighImportance

	accordion := widget.NewAccordion(
		widget.NewAccordionItem("Creation Date", creationFilter),
		widget.NewAccordionItem("First Album Year", albumFilter),
		widget.NewAccordionItem("Number of Members", memberContainer),
		widget.NewAccordionItem("Locations", locationFilter),
	)

	return container.NewVBox(
		accordion,
		resetBtn,
	), locationChecks, updateLocationChecks
}

func (l *ArtistList) rebuildGrid() {
	if l.grid == nil {
		return
	}
	l.grid.Objects = []fyne.CanvasObject{}
	filteredList := l.filteredArtists()

	if len(filteredList) == 0 {
		msgText := widget.NewLabel("Aucun résultat trouvé")
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

func (l *ArtistList) filteredArtists() []models.Artist {
	res := make([]models.Artist, 0, len(l.artists))
	st := l.searchText
	seen := make(map[int]bool)

	for _, a := range l.artists {
		if seen[a.ID] {
			continue
		}

		matchesSearch := st == "" || strings.Contains(strings.ToLower(a.Name), st)
		if !matchesSearch {
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

		if a.CreationDate < l.creationMin || a.CreationDate > l.creationMax {
			continue
		}

		albumYear := extractAlbumYear(a.FirstAlbum)
		if albumYear > 0 && (albumYear < l.albumMin || albumYear > l.albumMax) {
			continue
		}

		memberCount := len(a.Members)
		if !l.memberCounts[memberCount] {
			continue
		}

		matchesLocation := false
		if len(a.LocationsList) == 0 {
			matchesLocation = true
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
	uri, _ := storage.ParseURI(artist.Image)
	img := canvas.NewImageFromURI(uri)
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(200, 200))

	nameText := canvas.NewText(artist.Name, color.Black)
	nameText.TextStyle = fyne.TextStyle{Bold: true}
	nameText.Alignment = fyne.TextAlignCenter

	captionBg := canvas.NewRectangle(color.RGBA{R: 235, G: 235, B: 235, A: 255})
	caption := container.NewMax(
		captionBg,
		container.NewPadded(container.NewCenter(nameText)),
	)

	members := widget.NewLabel(fmt.Sprintf("%d membres", len(artist.Members)))
	members.Alignment = fyne.TextAlignCenter

	created := widget.NewLabel(fmt.Sprintf("Créé en %d", artist.CreationDate))
	created.Alignment = fyne.TextAlignCenter

	btn := widget.NewButton("Voir les détails", func() {
		onSelect(artist)
	})
	btn.Importance = widget.HighImportance

	card := container.NewVBox(
		img,
		caption,
		members,
		created,
		container.NewCenter(btn),
	)

	bg := canvas.NewRectangle(color.RGBA{R: 240, G: 240, B: 240, A: 255})

	return container.New(
		layout.NewMaxLayout(),
		bg,
		container.NewPadded(card),
	)
}
