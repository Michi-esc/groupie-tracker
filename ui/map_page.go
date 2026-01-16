package ui

import (
	"encoding/json"
	"fmt"
	"groupie-tracker/models"
	"image/color"
	"log"
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

// Cache global pour le géocodage
var geocodeCache = make(map[string]*LocationCoords)
var cacheMutex = &sync.Mutex{}

// LocationCoords représente les coordonnées d'un lieu
type LocationCoords struct {
	Lat      float64
	Lon      float64
	Location string
	Concerts []ConcertInfo
}

// ConcertInfo représente les infos d'un concert
type ConcertInfo struct {
	Artist string
	Dates  []string
}

// GeocodingResponse représente la réponse de l'API de géocodage
type GeocodingResponse struct {
	Lat string `json:"lat"`
	Lon string `json:"lon"`
}

// NewMapPageWithWindow crée une nouvelle page de carte et l'affiche dans la window
func NewMapPageWithWindow(win *Window, artists []models.Artist, onBack func()) {
	// Créer une barre de chargement simple
	loadingLabel := widget.NewLabel("Loading map...")
	loadingBar := widget.NewProgressBarInfinite()

	// Créer le bouton retour
	backButton := widget.NewButton("← Back", onBack)
	backButton.Importance = widget.HighImportance

	// Container temporaire avec chargement
	tempContainer := container.NewVBox(
		backButton,
		widget.NewLabel("Concerts Map"),
		loadingBar,
		loadingLabel,
	)

	// Afficher le container temporaire
	win.SetContent(tempContainer)

	// Charger la carte en arrière-plan
	go func() {
		// Récupérer les données de relations
		relations, err := models.FetchRelations()
		if err != nil {
			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   "Erreur",
				Content: fmt.Sprintf("Impossible de charger les données: %v", err),
			})
			return
		}

		fyne.Do(func() {
			loadingLabel.SetText("Geocoding locations...")
		})

		// Collecte tous les lieux uniques
		uniqueLocations := make(map[string]bool)
		var concertsByLocation map[string][]ConcertInfo = make(map[string][]ConcertInfo)

		for _, artist := range artists {
			for _, rel := range relations.Index {
				if rel.ID == artist.ID {
					for location, dates := range rel.DatesLocations {
						uniqueLocations[location] = true
						concertsByLocation[location] = append(concertsByLocation[location], ConcertInfo{
							Artist: artist.Name,
							Dates:  dates,
						})
					}
					break
				}
			}
		}

		totalLocations := len(uniqueLocations)
		fyne.Do(func() {
			loadingLabel.SetText(fmt.Sprintf("Geocoding: 0/%d", totalLocations))
		})

		// Géocoder en parallèle avec des goroutines - augmenter les workers pour plus de vitesse
		numWorkers := 16
		locationChan := make(chan string, len(uniqueLocations))
		resultChan := make(chan *LocationCoords, len(uniqueLocations))
		var wg sync.WaitGroup

		// Lancer les workers
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for location := range locationChan {
					coords := geocodeLocationFast(location)
					// Envoyer TOUJOURS un résultat, même s'il y a un fallback
					if coords != nil {
						coords.Concerts = concertsByLocation[location]
						resultChan <- coords
					}
				}
			}()
		}

		// Envoyer tous les lieux à géocoder
		go func() {
			for location := range uniqueLocations {
				locationChan <- location
			}
			close(locationChan)
		}()

		// Récupérer les résultats au fur et à mesure
		successCount := 0
		var locations []*LocationCoords
		var locMutex sync.Mutex

		// Goroutine pour compter les résultats
		wg.Add(1)
		go func() {
			defer wg.Done()
			for coords := range resultChan {
				if coords != nil {
					locMutex.Lock()
					locations = append(locations, coords)
					successCount++
					locMutex.Unlock()

					fyne.Do(func() {
						loadingLabel.SetText(fmt.Sprintf("Geocoding: %d/%d", successCount, totalLocations))
					})
				}
			}
		}()

		// Attendre que tous les workers finissent
		wg.Wait()
		close(resultChan)

		// Attendre que le comptage finisse
		time.Sleep(200 * time.Millisecond)

		if len(locations) == 0 {
			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   "Erreur",
				Content: "Aucun lieu n'a pu être géocodé",
			})
			return
		}

		// Générer la carte visuelle directement
		fyne.Do(func() {
			loadingLabel.SetText(fmt.Sprintf("Rendering %d locations...", len(locations)))
		})

		// Créer la carte visuelle en Canvas
		var mapCanvas fyne.CanvasObject
		defer func() {
			if r := recover(); r != nil {
				log.Println("Erreur dans createMapCanvas:", r)
				mapCanvas = canvas.NewText("Erreur lors de la création de la carte", TextLight)
			}
		}()

		mapCanvas = createMapCanvas(locations)
		scrollMap := container.NewScroll(mapCanvas)
		scrollMap.SetMinSize(fyne.NewSize(1000, 600))

		// Afficher la liste des lieux
		fyne.Do(func() {
			loadingLabel.SetText("Loading locations list...")
		})
		time.Sleep(300 * time.Millisecond)

		// Créer la liste des lieux
		var locationsList fyne.CanvasObject
		defer func() {
			if r := recover(); r != nil {
				log.Println("Erreur dans createLocationsList:", r)
				locationsList = canvas.NewText("Erreur lors de la création de la liste", TextLight)
			}
		}()

		locationsList = createLocationsList(locations)
		scrollLocations := container.NewScroll(locationsList)
		scrollLocations.SetMinSize(fyne.NewSize(600, 600))

		// Info
		infoLabel := widget.NewLabel(fmt.Sprintf("%d locations found", len(locations)))
		infoLabel.Alignment = fyne.TextAlignCenter
		infoLabel.TextStyle = fyne.TextStyle{Bold: true}

		// Titre
		title := widget.NewLabel("Concerts Map")
		title.TextStyle = fyne.TextStyle{Bold: true}
		title.Alignment = fyne.TextAlignCenter

		// Afficher la carte et la liste côte à côte
		contentDisplay := container.NewHSplit(scrollMap, scrollLocations)

		// Créer le container final avec header
		finalContent := container.NewBorder(
			container.NewVBox(backButton, title, infoLabel),
			nil, nil, nil,
			contentDisplay,
		)

		// Mettre à jour le contenu de la window depuis le thread UI
		log.Println("Affichage de la carte avec", len(locations), "lieux")
		fyne.Do(func() {
			win.SetContent(finalContent)
		})
	}()
}

// geocodeLocationFast utilise plusieurs services de géocodage avec fallback
func geocodeLocationFast(location string) *LocationCoords {
	// Vérifier le cache
	cacheMutex.Lock()
	if cached, exists := geocodeCache[location]; exists {
		cacheMutex.Unlock()
		return cached
	}
	cacheMutex.Unlock()

	// Nettoyer le nom de la location
	cleanLocation := strings.ReplaceAll(location, "_", " ")
	cleanLocation = strings.ReplaceAll(cleanLocation, "-", ", ")

	// Essayer d'abord le fallback rapide (par défaut par pays)
	// Cela évite d'attendre les APIs qui peuvent être lentes
	fallbackCoords := getApproxCoordinates(cleanLocation)
	if fallbackCoords != nil {
		cacheMutex.Lock()
		geocodeCache[location] = fallbackCoords
		cacheMutex.Unlock()
		return fallbackCoords
	}

	// Si le fallback n'a pas trouvé, essayer les APIs en parallèle avec timeout court
	apis := []struct {
		name   string
		urlFmt string
	}{
		{
			"osm",
			"https://nominatim.openstreetmap.org/search?format=json&q=%s&limit=1&accept-language=en",
		},
		{
			"locationiq",
			"https://us1.locationiq.com/v1/search?key=pk.0b2779c3718c5e80c5d6fa03e25a4ee0&format=json&q=%s&limit=1",
		},
	}

	// Essayer les APIs en parallèle pour plus de vitesse
	resultChan := make(chan *LocationCoords, len(apis))
	for _, api := range apis {
		go func(a struct {
			name   string
			urlFmt string
		}) {
			result := tryGeocodeAPI(cleanLocation, a.urlFmt, a.name)
			if result != nil {
				resultChan <- result
			}
		}(api)
	}

	// Attendre le premier résultat avec timeout
	select {
	case result := <-resultChan:
		if result != nil {
			cacheMutex.Lock()
			geocodeCache[location] = result
			cacheMutex.Unlock()
			return result
		}
	case <-time.After(3 * time.Second):
		// Timeout - utiliser le fallback
	}

	// Fallback : extraire le pays du lieu et utiliser des coordonnées par défaut
	coords := getApproxCoordinates(cleanLocation)
	if coords != nil {
		cacheMutex.Lock()
		geocodeCache[location] = coords
		cacheMutex.Unlock()
		return coords
	}

	// Dernier fallback : coordonnées mondiales au centre
	defaultCoords := &LocationCoords{
		Lat:      20.0,
		Lon:      0.0,
		Location: location,
	}
	cacheMutex.Lock()
	geocodeCache[location] = defaultCoords
	cacheMutex.Unlock()
	return defaultCoords
}

// tryGeocodeAPI essaie de géocoder avec une API spécifique
func tryGeocodeAPI(cleanLocation, urlFmt, apiName string) *LocationCoords {
	url := fmt.Sprintf(urlFmt, strings.ReplaceAll(cleanLocation, " ", "+"))

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil
	}

	var results []GeocodingResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil
	}

	if len(results) == 0 {
		return nil
	}

	var lat, lon float64
	fmt.Sscanf(results[0].Lat, "%f", &lat)
	fmt.Sscanf(results[0].Lon, "%f", &lon)

	if lat == 0 && lon == 0 {
		return nil
	}

	result := &LocationCoords{
		Lat:      lat,
		Lon:      lon,
		Location: cleanLocation,
	}

	return result
}

// getApproxCoordinates retourne des coordonnées approximatives selon le pays
func getApproxCoordinates(location string) *LocationCoords {
	location = strings.ToLower(strings.TrimSpace(location))

	// Extraire le dernier mot (le pays)
	parts := strings.Split(location, ",")
	if len(parts) > 0 {
		location = strings.TrimSpace(parts[len(parts)-1])
	}

	// Coordonnées approximatives des pays majeurs
	countryCoords := map[string][2]float64{
		"usa":            {39.8283, -98.5795},
		"united states":  {39.8283, -98.5795},
		"uk":             {55.3781, -3.4360},
		"united kingdom": {55.3781, -3.4360},
		"france":         {46.2276, 2.2137},
		"germany":        {51.1657, 10.4515},
		"spain":          {40.4637, -3.7492},
		"italy":          {41.8719, 12.5674},
		"japan":          {36.2048, 138.2529},
		"canada":         {56.1304, -106.3468},
		"australia":      {-25.2744, 133.7751},
		"brazil":         {-14.2350, -51.9253},
		"mexico":         {23.6345, -102.5528},
		"netherlands":    {52.1326, 5.2913},
		"belgium":        {50.5039, 4.4699},
		"switzerland":    {46.8182, 8.2275},
		"sweden":         {60.1282, 18.6435},
		"norway":         {60.4720, 8.4689},
		"denmark":        {56.2639, 9.5018},
		"finland":        {61.9241, 25.7482},
		"portugal":       {39.3999, -8.2245},
		"ireland":        {53.4129, -8.2439},
		"poland":         {51.9194, 19.1451},
		"austria":        {47.5162, 14.5501},
		"czech":          {49.8175, 15.4730},
		"greece":         {39.0742, 21.8243},
		"russia":         {61.5240, 105.3188},
		"china":          {35.8617, 104.1954},
		"korea":          {35.9078, 127.7669},
		"india":          {20.5937, 78.9629},
		"argentina":      {-38.4161, -63.6167},
		"chile":          {-35.6751, -71.5430},
		"colombia":       {4.5709, -74.2973},
		"peru":           {-9.1900, -75.0152},
		"new zealand":    {-40.9006, 174.8860},
		"south africa":   {-30.5595, 22.9375},
		"israel":         {31.0461, 34.8516},
		"turkey":         {38.9637, 35.2433},
	}

	if coords, ok := countryCoords[location]; ok {
		return &LocationCoords{
			Lat:      coords[0],
			Lon:      coords[1],
			Location: location,
		}
	}

	return nil
}

// createMapCanvas crée une représentation visuelle de la carte avec les coordonnées
func createMapCanvas(locations []*LocationCoords) fyne.CanvasObject {
	if len(locations) == 0 {
		return canvas.NewText("Aucun lieu de concert", TextLight)
	}

	// Calculer les limites
	minLat := locations[0].Lat
	maxLat := locations[0].Lat
	minLon := locations[0].Lon
	maxLon := locations[0].Lon

	for _, loc := range locations {
		if loc.Lat < minLat {
			minLat = loc.Lat
		}
		if loc.Lat > maxLat {
			maxLat = loc.Lat
		}
		if loc.Lon < minLon {
			minLon = loc.Lon
		}
		if loc.Lon > maxLon {
			maxLon = loc.Lon
		}
	}

	// Ajouter du padding
	latRange := maxLat - minLat
	lonRange := maxLon - minLon

	// S'assurer qu'il y a au moins un petit range pour éviter les divisions par zéro
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

	// Recalculer après padding
	latRange = maxLat - minLat
	lonRange = maxLon - minLon

	// Dimensions de la carte
	mapWidth := float32(1000)
	mapHeight := float32(600)

	// Créer le fond de la carte
	mapBg := canvas.NewRectangle(BgDarker)
	mapBg.SetMinSize(fyne.NewSize(mapWidth, mapHeight))

	// Créer un conteneur pour les éléments de la carte
	mapContainer := container.New(layout.NewMaxLayout())
	mapContainer.Add(mapBg)

	// Fonction pour convertir les coordonnées lat/lon en pixels
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

	// Ajouter les marqueurs
	for _, loc := range locations {
		x := toLon(loc.Lon)
		y := toLat(loc.Lat)

		// Créer un marqueur (cercle rouge) - simplifié pour plus de vitesse
		marker := canvas.NewCircle(AccentPink)
		marker.StrokeWidth = 1
		marker.StrokeColor = AccentCyan
		marker.Move(fyne.NewPos(x-5, y-5))
		marker.Resize(fyne.NewSize(10, 10))

		// Ajouter le marqueur à la carte
		mapContainer.Add(marker)

		// Ajouter le texte du lieu (petit, tous les 10 marqueurs pour éviter la surcharge)
		if len(locations) <= 50 || (len(locations) > 50 && indexOfLocation(locations, loc)%3 == 0) {
			locationText := canvas.NewText(strings.Split(loc.Location, ",")[0], TextLight)
			locationText.TextSize = 9
			locationText.Move(fyne.NewPos(x+10, y-5))
			mapContainer.Add(locationText)
		}
	}

	// Ajouter une grille légère (moins de lignes pour plus de vitesse)
	gridColor := color.RGBA{R: 100, G: 100, B: 100, A: 30}

	// Lignes verticales - réduites de 5 à 3
	for i := 0; i <= 3; i++ {
		x := float32(i) * mapWidth / 3
		line := canvas.NewLine(gridColor)
		line.StrokeWidth = 0.5
		line.Position1 = fyne.NewPos(x, 0)
		line.Position2 = fyne.NewPos(x, mapHeight)
		mapContainer.Add(line)
	}

	// Lignes horizontales - réduites de 5 à 3
	for i := 0; i <= 3; i++ {
		y := float32(i) * mapHeight / 3
		line := canvas.NewLine(gridColor)
		line.StrokeWidth = 0.5
		line.Position1 = fyne.NewPos(0, y)
		line.Position2 = fyne.NewPos(mapWidth, y)
		mapContainer.Add(line)
	}

	// Ajouter les coordonnées aux coins
	topLeft := canvas.NewText(fmt.Sprintf("%.1f°N, %.1f°W", maxLat, minLon), TextLight)
	topLeft.TextSize = 8
	topLeft.Move(fyne.NewPos(5, 5))
	mapContainer.Add(topLeft)

	bottomRight := canvas.NewText(fmt.Sprintf("%.1f°S, %.1f°E", minLat, maxLon), TextLight)
	bottomRight.TextSize = 8
	bottomRight.Move(fyne.NewPos(mapWidth-100, mapHeight-20))
	mapContainer.Add(bottomRight)

	mapContainer.Resize(fyne.NewSize(mapWidth, mapHeight))
	return mapContainer
}

// indexOfLocation trouve l'index d'une LocationCoords dans une slice
func indexOfLocation(locations []*LocationCoords, target *LocationCoords) int {
	for i, loc := range locations {
		if loc.Location == target.Location {
			return i
		}
	}
	return -1
}

// generateStaticMapURL génère une URL pour une image de carte statique avec marqueurs
func generateStaticMapURL(locations []*LocationCoords) string {
	if len(locations) == 0 {
		return ""
	}

	// Calculer le centre et le zoom optimal
	minLat := locations[0].Lat
	maxLat := locations[0].Lat
	minLon := locations[0].Lon
	maxLon := locations[0].Lon

	for _, loc := range locations {
		if loc.Lat < minLat {
			minLat = loc.Lat
		}
		if loc.Lat > maxLat {
			maxLat = loc.Lat
		}
		if loc.Lon < minLon {
			minLon = loc.Lon
		}
		if loc.Lon > maxLon {
			maxLon = loc.Lon
		}
	}

	centerLat := (minLat + maxLat) / 2
	centerLon := (minLon + maxLon) / 2

	// Utiliser plusieurs services en fallback
	// 1. Essayer avec Google Static Maps
	// Note: Sans clé API, ce service retournera un watermark mais fonctionnera
	baseURL := "https://maps.googleapis.com/maps/api/staticmap?"
	params := fmt.Sprintf("center=%.6f,%.6f&zoom=3&size=1200x700&style=feature:all|element:labels|visibility:off", centerLat, centerLon)

	// Ajouter les marqueurs (limiter à 50)
	for i, loc := range locations {
		if i >= 50 {
			break
		}
		params += fmt.Sprintf("&markers=color:red|%.6f,%.6f", loc.Lat, loc.Lon)
	}

	// URL sans clé API - montrera une démonstration
	return baseURL + params + "&key=AIzaSyB41DRUbKWJHPxagoK4fLi1aZjqsqOlEdE"
}

// createLocationsList crée une liste widget des lieux avec leurs concerts
func createLocationsList(locations []*LocationCoords) *fyne.Container {
	var items []fyne.CanvasObject

	// Titre de la liste
	titleLabel := widget.NewLabel("Liste des lieux de concerts")
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	items = append(items, titleLabel)
	items = append(items, widget.NewSeparator())

	for _, loc := range locations {
		// Créer un conteneur pour chaque lieu
		locationName := strings.ReplaceAll(loc.Location, "_", " ")
		locationName = strings.ReplaceAll(locationName, "-", ", ")

		locLabel := widget.NewLabel("• " + locationName)
		locLabel.TextStyle = fyne.TextStyle{Bold: true}

		items = append(items, locLabel)

		// Ajouter les concerts pour ce lieu
		for _, concert := range loc.Concerts {
			artistLabel := widget.NewLabel("  ♫ " + concert.Artist)
			items = append(items, artistLabel)

			// Afficher quelques dates
			maxDates := 3
			for i, date := range concert.Dates {
				if i >= maxDates {
					remaining := len(concert.Dates) - maxDates
					items = append(items, widget.NewLabel(fmt.Sprintf("      ... et %d autres dates", remaining)))
					break
				}
				dateLabel := widget.NewLabel("      • " + date)
				items = append(items, dateLabel)
			}
		}

		items = append(items, widget.NewSeparator())
	}

	return container.NewVBox(items...)
}
