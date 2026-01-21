package ui

import (
	"fmt"
	"groupie-tracker/models"
	"image/color"
	"log"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// concert info
type ConcertInfo struct {
	Artist string
	Dates  []string
}

// page carte
func NewMapPageWithWindow(win *Window, artists []models.Artist, onBack func()) {
	// Créer une barre de chargement simple
	loadingLabel := widget.NewLabel(T().Loading)
	loadingBar := widget.NewProgressBarInfinite()

	// Créer le bouton retour
	backButton := widget.NewButton(T().Back, onBack)
	backButton.Importance = widget.HighImportance

	// Container temporaire avec chargement
	tempContainer := container.NewVBox(
		backButton,
		widget.NewLabel(T().Map),
		loadingBar,
		loadingLabel,
	)

	// Afficher le container temporaire
	win.SetContent(tempContainer)

	// chargement de la carte en arrière-plan
	go func() {
		// récupère relations et locations via l'api
		relations, err := models.FetchRelations()
		if err != nil {
			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   T().Error,
				Content: fmt.Sprintf("Impossible de charger les relations: %v", err),
			})
			return
		}

		locations, err := models.FetchLocations()
		if err != nil {
			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   T().Error,
				Content: fmt.Sprintf("Impossible de charger les locations: %v", err),
			})
			return
		}

		fyne.Do(func() {
			loadingLabel.SetText(T().Loading)
		})

		// on construit une map des lieux avec leurs coords
		locationsMap := make(map[string]*models.LocationCoords)
		for _, loc := range locations.Index {
			for _, place := range loc.Lieux {
				parts := strings.Split(place, ",")
				if len(parts) == 2 {
					var lat, lon float64
					fmt.Sscanf(strings.TrimSpace(parts[0]), "%f", &lat)
					fmt.Sscanf(strings.TrimSpace(parts[1]), "%f", &lon)

					locationsMap[place] = &models.LocationCoords{
						ID:        loc.ID,
						Lieux:     place,
						Latitude:  lat,
						Longitude: lon,
					}
				}
			}
		}

		// associe chaque lieu aux concerts
		var concertLocations []*models.LocationCoords
		concertsByLocation := make(map[string][]ConcertInfo)

		for _, artist := range artists {
			for _, rel := range relations.Index {
				if rel.ID == artist.ID {
					for location, dates := range rel.DatesLocations {
						if _, ok := locationsMap[location]; ok {
							concertsByLocation[location] = append(concertsByLocation[location], ConcertInfo{
								Artist: artist.Name,
								Dates:  dates,
							})
						}
					}
					break
				}
			}
		}

		// on garde les lieux uniques
		seen := make(map[string]bool)
		for locName, loc := range locationsMap {
			if !seen[locName] {
				seen[locName] = true
				concertLocations = append(concertLocations, loc)
			}
		}

		if len(concertLocations) == 0 {
			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   T().Error,
				Content: "Aucun lieu de concert n'a pu être chargé",
			})
			return
		}

		// on peut générer la carte visuelle
		fyne.Do(func() {
			loadingLabel.SetText(fmt.Sprintf("Rendering %d locations...", len(concertLocations)))
		})

		// création de la carte canvas
		var mapCanvas fyne.CanvasObject
		defer func() {
			if r := recover(); r != nil {
				log.Println("Erreur dans createMapCanvas:", r)
				mapCanvas = canvas.NewText("Erreur lors de la création de la carte", TextLight)
			}
		}()

		mapCanvas = createMapCanvasFromAPI(concertLocations)
		scrollMap := container.NewScroll(mapCanvas)
		scrollMap.SetMinSize(fyne.NewSize(1000, 600))

		// on prépare la liste des lieux
		fyne.Do(func() {
			loadingLabel.SetText("Loading locations list...")
		})
		time.Sleep(300 * time.Millisecond)

		// on génère la liste des lieux
		var locationsList fyne.CanvasObject
		defer func() {
			if r := recover(); r != nil {
				log.Println("Erreur dans createLocationsList:", r)
				locationsList = canvas.NewText("Erreur lors de la création de la liste", TextLight)
			}
		}()
		scrollLocations := container.NewScroll(locationsList)
		scrollLocations.SetMinSize(fyne.NewSize(600, 600))

		// petit résumé du nombre de lieux
		infoLabel := widget.NewLabel(fmt.Sprintf("%d "+T().Location, len(concertLocations)))
		infoLabel.Alignment = fyne.TextAlignCenter
		infoLabel.TextStyle = fyne.TextStyle{Bold: true}

		// titre de la page
		title := widget.NewLabel(T().ConcertLocations)
		title.TextStyle = fyne.TextStyle{Bold: true}
		title.Alignment = fyne.TextAlignCenter

		// carte + liste côte à côte
		contentDisplay := container.NewHSplit(scrollMap, scrollLocations)

		// border final
		finalContent := container.NewBorder(
			container.NewVBox(backButton, title, infoLabel),
			nil, nil, nil,
			contentDisplay,
		)

		// Mettre à jour le contenu de la window depuis le thread UI
		log.Println("Affichage de la carte avec", len(concertLocations), "lieux")
		fyne.Do(func() {
			win.SetContent(finalContent)
		})
	}()
}

// compat
func geocodeLocationFast(location string) *models.LocationCoords {
	// compat
	return nil
}

// dessine carte
func createMapCanvasFromAPI(locations []*models.LocationCoords) fyne.CanvasObject {
	if len(locations) == 0 {
		return canvas.NewText(T().NoLocations, ContrastColor(BgDarker))
	}

	// calcul des bornes lat/lon
	minLat := locations[0].Latitude
	maxLat := locations[0].Latitude
	minLon := locations[0].Longitude
	maxLon := locations[0].Longitude

	for _, loc := range locations {
		if loc.Latitude < minLat {
			minLat = loc.Latitude
		}
		if loc.Latitude > maxLat {
			maxLat = loc.Latitude
		}
		if loc.Longitude < minLon {
			minLon = loc.Longitude
		}
		if loc.Longitude > maxLon {
			maxLon = loc.Longitude
		}
	}

	// on ajoute un petit padding
	latRange := maxLat - minLat
	lonRange := maxLon - minLon

	// évite les ranges nuls
	if latRange < 0.1 {
		latRange = 0.1
	}
	if lonRange < 0.1 {
		lonRange = 0.1
	}

	latPadding := latRange * 0.1
	lonPadding := lonRange * 0.1
	minLat -= latPadding
	maxLat += latPadding
	minLon -= lonPadding
	maxLon += lonPadding

	// recalcul après padding
	latRange = maxLat - minLat
	lonRange = maxLon - minLon

	// dimensions du canvas
	mapWidth := float32(1000)
	mapHeight := float32(600)

	// fond de carte
	mapBg := canvas.NewRectangle(BgDarker)
	mapBg.SetMinSize(fyne.NewSize(mapWidth, mapHeight))

	// conteneur pour tout empiler
	mapContainer := container.New(layout.NewMaxLayout())
	mapContainer.Add(mapBg)

	// convertit lat/lon en pixels
	toLat := func(lat float64) float32 {
		if latRange == 0 {
			return mapHeight / 2
		}
		return mapHeight - float32((lat-minLat)/latRange)*mapHeight
	}
	toLon := func(lon float64) float32 {
		if lonRange == 0 {
			return mapWidth / 2
		}
		return float32((lon-minLon)/lonRange) * mapWidth
	}

	// on place les marqueurs
	for _, loc := range locations {
		x := toLon(loc.Longitude)
		y := toLat(loc.Latitude)

		// petit cercle rose pour le point
		marker := canvas.NewCircle(AccentPink)
		marker.StrokeWidth = 1
		marker.StrokeColor = AccentCyan
		marker.Move(fyne.NewPos(x-5, y-5))
		marker.Resize(fyne.NewSize(10, 10))

		// on ajoute le point
		mapContainer.Add(marker)

		// texte du lieu, pas trop fréquent pour éviter le fouillis
		if len(locations) <= 50 || (len(locations) > 50 && indexOfLocationFromAPI(locations, loc)%3 == 0) {
			locationName := strings.ReplaceAll(loc.Lieux, "_", " ")
			locationName = strings.ReplaceAll(locationName, "-", ", ")
			parts := strings.Split(locationName, ",")
			displayName := parts[0]
			if len(parts) > 1 {
				displayName = parts[len(parts)-1]
			}
			locationText := canvas.NewText(strings.TrimSpace(displayName), ContrastColor(BgDarker))
			locationText.TextSize = 9
			locationText.Move(fyne.NewPos(x+10, y-5))
			mapContainer.Add(locationText)
		}
	}

	// petite grille légère
	gridColor := color.RGBA{R: 100, G: 100, B: 100, A: 30}

	// lignes verticales
	for i := 0; i <= 3; i++ {
		x := float32(i) * mapWidth / 3
		line := canvas.NewLine(gridColor)
		line.StrokeWidth = 0.5
		line.Position1 = fyne.NewPos(x, 0)
		line.Position2 = fyne.NewPos(x, mapHeight)
		mapContainer.Add(line)
	}

	// lignes horizontales
	for i := 0; i <= 3; i++ {
		y := float32(i) * mapHeight / 3
		line := canvas.NewLine(gridColor)
		line.StrokeWidth = 0.5
		line.Position1 = fyne.NewPos(0, y)
		line.Position2 = fyne.NewPos(mapWidth, y)
		mapContainer.Add(line)
	}

	// coordonnées affichées dans les coins
	topLeft := canvas.NewText(fmt.Sprintf("%.1f°N, %.1f°W", maxLat, minLon), ContrastColor(BgDarker))
	topLeft.TextSize = 8
	topLeft.Move(fyne.NewPos(5, 5))
	mapContainer.Add(topLeft)

	bottomRight := canvas.NewText(fmt.Sprintf("%.1f°S, %.1f°E", minLat, maxLon), ContrastColor(BgDarker))
	bottomRight.TextSize = 8
	bottomRight.Move(fyne.NewPos(mapWidth-100, mapHeight-20))
	mapContainer.Add(bottomRight)

	mapContainer.Resize(fyne.NewSize(mapWidth, mapHeight))
	return mapContainer
}

// index lieu
func indexOfLocationFromAPI(locations []*models.LocationCoords, target *models.LocationCoords) int {
	for i, loc := range locations {
		if loc.Lieux == target.Lieux {
			return i
		}
	}
	return -1
}

// liste lieux
func createLocationsListFromAPI(locations []*models.LocationCoords, concertsByLocation map[string][]ConcertInfo) *fyne.Container {
	var items []fyne.CanvasObject

	// titre de la liste
	titleLabel := widget.NewLabel(T().LocationsListTitle)
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	items = append(items, titleLabel)
	items = append(items, widget.NewSeparator())

	for _, loc := range locations {
		// conteneur pour chaque lieu
		locationName := strings.ReplaceAll(loc.Lieux, "_", " ")
		locationName = strings.ReplaceAll(locationName, "-", ", ")

		// affiche les coords fournies par l'api
		coordsText := fmt.Sprintf("(%.4f, %.4f)", loc.Latitude, loc.Longitude)
		locLabel := widget.NewLabel("• " + locationName + " " + coordsText)
		locLabel.TextStyle = fyne.TextStyle{Bold: true}

		items = append(items, locLabel)

		// liste les concerts associés
		if concerts, ok := concertsByLocation[loc.Lieux]; ok {
			for _, concert := range concerts {
				artistLabel := widget.NewLabel("  ♫ " + concert.Artist)
				items = append(items, artistLabel)

				// on affiche quelques dates
				maxDates := 3
				for i, date := range concert.Dates {
					if i >= maxDates {
						remaining := len(concert.Dates) - maxDates
						items = append(items, widget.NewLabel(fmt.Sprintf(T().MoreDatesFmt, remaining)))
						break
					}
					dateLabel := widget.NewLabel("      • " + date)
					items = append(items, dateLabel)
				}
			}
		}

		items = append(items, widget.NewSeparator())
	}

	return container.NewVBox(items...)
}
