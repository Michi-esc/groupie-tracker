package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"groupie-tracker/models"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

var geocodeCache = make(map[string]*LocationCoords)
var cacheMutex = &sync.Mutex{}

type LocationCoords struct {
	Lat      float64
	Lon      float64
	Location string
	Concerts []ConcertInfo
}

// concert info
type ConcertInfo struct {
	Artist string
	Dates  []string
}

type GeocodingResponse struct {
	Lat string `json:"lat"`
	Lon string `json:"lon"`
}

// page carte
func NewMapPageWithWindow(win *Window, artists []models.Artist, onBack func()) {
	loadingLabel := widget.NewLabel("Loading map...")
	// Créer une barre de chargement simple
	loadingLabel := widget.NewLabel(T().Loading)
	loadingBar := widget.NewProgressBarInfinite()

	backButton := widget.NewButton("← Back", onBack)
	// Créer le bouton retour
	backButton := widget.NewButton(T().Back, onBack)
	backButton.Importance = widget.HighImportance

	tempContainer := container.NewVBox(
		backButton,
		widget.NewLabel(T().Map),
		loadingBar,
		loadingLabel,
	)

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
		log.Printf("✓ Relations loaded: %d artists\n", len(relations.Index))

		locations, err := models.FetchLocations()
		if err != nil {
			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   T().Error,
				Content: fmt.Sprintf("Impossible de charger les locations: %v", err),
			})
			return
		}
		log.Printf("✓ Locations loaded: %d location entries\n", len(locations.Index))

		fyne.Do(func() {
			loadingLabel.SetText("Geocoding locations...")
		})

			loadingLabel.SetText(T().Loading)
		})

		// on construit une map des lieux avec leurs coords via geocodage (en parallèle limité)
		locationsMap := make(map[string]*models.LocationCoords)
		var wg sync.WaitGroup
		var mu sync.Mutex

		// semaphore pour limiter les goroutines parallèles (Nominatim a un rate limit strict)
		// utiliser seulement 2 goroutines pour respecter le rate limit
		semaphore := make(chan struct{}, 2)

		// collect unique locations first
		uniqueLocations := make(map[string]bool)
		for _, loc := range locations.Index {
			for _, place := range loc.Locations {
				if !uniqueLocations[place] {
					uniqueLocations[place] = true
					wg.Add(1)
					go func(place string) {
						defer wg.Done()
						semaphore <- struct{}{}        // acquire
						defer func() { <-semaphore }() // release

						coords := geocodeLocationFast(place)
						if coords != nil && coords.Latitude != 0 && coords.Longitude != 0 {
							mu.Lock()
							locationsMap[place] = coords
							mu.Unlock()
							log.Printf("✓ Geocoded: %s -> (%.4f, %.4f)\n", place, coords.Latitude, coords.Longitude)
						} else {
							log.Printf("✗ Geocode failed for: %s\n", place)
						}
					}(place)
				}
			}
		}
		wg.Wait()
		log.Printf("✓ Locations map built: %d unique places\n", len(locationsMap))

		// associe chaque lieu aux concerts
		var concertLocations []*models.LocationCoords
		concertsByLocation := make(map[string][]ConcertInfo)

		matchedCount := 0
		for _, artist := range artists {
			for _, rel := range relations.Index {
				if rel.ID == artist.ID {
					log.Printf("  ✓ Found relation for artist %d (%s) with %d locations\n", artist.ID, artist.Name, len(rel.DatesLocations))
					for location, dates := range rel.DatesLocations {
						if locCoords, ok := locationsMap[location]; ok {
							log.Printf("    ✓ Location found: %s (lat=%.4f, lon=%.4f)\n", location, locCoords.Latitude, locCoords.Longitude)
							concertsByLocation[location] = append(concertsByLocation[location], ConcertInfo{
								Artist: artist.Name,
								Dates:  dates,
							})
							matchedCount++
						} else {
							log.Printf("    ✗ Location NOT in map: %s\n", location)
						}
					}
					break
				}
			}
		}

		totalLocations := len(uniqueLocations)
		fyne.Do(func() {
			loadingLabel.SetText(fmt.Sprintf("Geocoding: 0/%d", totalLocations))
		})

		numWorkers := 16
		locationChan := make(chan string, len(uniqueLocations))
		resultChan := make(chan *LocationCoords, len(uniqueLocations))
		var wg sync.WaitGroup

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for location := range locationChan {
					coords := geocodeLocationFast(location)
					if coords != nil {
						coords.Concerts = concertsByLocation[location]
						resultChan <- coords
					}
				}
			}()
		}

		go func() {
			for location := range uniqueLocations {
				locationChan <- location
			}
			close(locationChan)
		}()

		successCount := 0
		var locations []*LocationCoords
		var locMutex sync.Mutex

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

		wg.Wait()
		close(resultChan)

		time.Sleep(200 * time.Millisecond)

		if len(locations) == 0 {
		log.Printf("✓ Matched %d artist-location pairs\n", matchedCount)

		// on garde les lieux uniques
		seen := make(map[string]bool)
		for locName, loc := range locationsMap {
			if !seen[locName] {
				seen[locName] = true
				concertLocations = append(concertLocations, loc)
			}
		}
		log.Printf("✓ Final concert locations count: %d\n", len(concertLocations))

		if len(concertLocations) == 0 {
			log.Printf("✗ ERROR: No concert locations found!\n")
			log.Printf("  - locationsMap size: %d\n", len(locationsMap))
			log.Printf("  - concertsByLocation size: %d\n", len(concertsByLocation))
			log.Printf("  - artists count: %d\n", len(artists))
			log.Printf("  - relations count: %d\n", len(relations.Index))
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
		locationsList = createLocationsListFromAPI(concertLocations, concertsByLocation)
		scrollLocations := container.NewScroll(locationsList)
		scrollLocations.SetMinSize(fyne.NewSize(600, 600))

		infoLabel := widget.NewLabel(fmt.Sprintf("%d locations found", len(locations)))
		// petit résumé du nombre de lieux
		infoLabel := widget.NewLabel(fmt.Sprintf("%d "+T().Location, len(concertLocations)))
		infoLabel.Alignment = fyne.TextAlignCenter
		infoLabel.TextStyle = fyne.TextStyle{Bold: true}

		title := widget.NewLabel("Concerts Map")
		// titre de la page
		title := widget.NewLabel(T().ConcertLocations)
		title.TextStyle = fyne.TextStyle{Bold: true}
		title.Alignment = fyne.TextAlignCenter

		contentDisplay := container.NewHSplit(scrollMap, scrollLocations)
		// carte + liste côte à côte
		contentDisplay := container.NewHSplit(mapCanvas, scrollLocations)

		// border final
		finalContent := container.NewBorder(
			container.NewVBox(backButton, title, infoLabel),
			nil, nil, nil,
			contentDisplay,
		)

		log.Println("Affichage de la carte avec", len(locations), "lieux")
		// Mettre à jour le contenu de la window depuis le thread UI
		log.Println("Affichage de la carte avec", len(concertLocations), "lieux")
		fyne.Do(func() {
			win.SetContent(finalContent)
		})
	}()
}

func geocodeLocationFast(location string) *LocationCoords {
	cacheMutex.Lock()
	if cached, exists := geocodeCache[location]; exists {
		cacheMutex.Unlock()
// compat
func geocodeLocationFast(location string) *models.LocationCoords {
	// Use Nominatim API to geocode location names
	// location format: "city-country" e.g., "london-uk", "new_york-usa"

	// Check cache first
	if cached := models.GetCachedCoords(location); cached != nil {
		return cached
	}
	cacheMutex.Unlock()

	cleanLocation := strings.ReplaceAll(location, "_", " ")
	cleanLocation = strings.ReplaceAll(cleanLocation, "-", ", ")

	// Apply location mappings for problematic/outdated location names
	normalizedLocation := normalizeLocation(location)

	fallbackCoords := getApproxCoordinates(cleanLocation)
	if fallbackCoords != nil {
		cacheMutex.Lock()
		geocodeCache[location] = fallbackCoords
		cacheMutex.Unlock()
		return fallbackCoords
	}

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

	select {
	case result := <-resultChan:
		if result != nil {
			cacheMutex.Lock()
			geocodeCache[location] = result
			cacheMutex.Unlock()
			return result
		}
	case <-time.After(3 * time.Second):
	}

	coords := getApproxCoordinates(cleanLocation)
	if coords != nil {
		cacheMutex.Lock()
		geocodeCache[location] = coords
		cacheMutex.Unlock()
		return coords
	}

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

func tryGeocodeAPI(cleanLocation, urlFmt, apiName string) *LocationCoords {
	url := fmt.Sprintf(urlFmt, strings.ReplaceAll(cleanLocation, " ", "+"))

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
	// replace underscore with space for better geocoding
	query := strings.ReplaceAll(normalizedLocation, "_", " ")

	// create the Nominatim API URL
	url := fmt.Sprintf("https://nominatim.openstreetmap.org/search?q=%s&format=json&limit=1",
		strings.ReplaceAll(query, " ", "+"))

	// Retry logic with exponential backoff
	maxAttempts := 3
	var resp *http.Response
	var err error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Create request with context timeout (10 seconds per attempt)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		req, _ := http.NewRequest("GET", url, nil)
		req = req.WithContext(ctx)
		req.Header.Set("User-Agent", "groupie-tracker/1.0")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err = client.Do(req)
		cancel()

		if err == nil && resp.StatusCode == 200 {
			break // Success
		}

		if err != nil {
			log.Printf("[GEOCODE FAIL] %s attempt %d/%d: %v\n", location, attempt, maxAttempts, err)
		} else {
			log.Printf("[GEOCODE HTTP %d] %s attempt %d/%d\n", resp.StatusCode, location, attempt, maxAttempts)
			resp.Body.Close()
		}

		// Exponential backoff: 500ms, 1s, 2s
		if attempt < maxAttempts {
			backoffDuration := time.Duration(500*int(math.Pow(2, float64(attempt-1)))) * time.Millisecond
			log.Printf("[GEOCODE RETRY] %s in %v\n", location, backoffDuration)
			time.Sleep(backoffDuration)
		}
	}

	if err != nil || resp == nil || resp.StatusCode != 200 {
		log.Printf("[GEOCODE FAILED] %s - all retries exhausted\n", location)
		return nil
	}
	defer resp.Body.Close()

	var results []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		log.Printf("[GEOCODE DECODE ERR] %s: %v\n", location, err)
		return nil
	}

	if len(results) == 0 {
		log.Printf("[GEOCODE NO RESULTS] %s\n", location)
		return nil
	}

	result := results[0]
	lat, _ := strconv.ParseFloat(fmt.Sprintf("%v", result["lat"]), 64)
	lon, _ := strconv.ParseFloat(fmt.Sprintf("%v", result["lon"]), 64)

	coords := &models.LocationCoords{
		Lieux:     location,
		Latitude:  lat,
		Longitude: lon,
	}

	return result
}

func getApproxCoordinates(location string) *LocationCoords {
	location = strings.ToLower(strings.TrimSpace(location))

	parts := strings.Split(location, ",")
	if len(parts) > 0 {
		location = strings.TrimSpace(parts[len(parts)-1])
	}

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
	// Cache the result
	models.CacheCoords(location, coords)

	// respecter le rate limit de Nominatim (1 req/sec minimum)
	time.Sleep(800 * time.Millisecond)

	return coords
}

// normalizeLocation maps outdated or problematic location names to current valid ones
func normalizeLocation(location string) string {
	// Mapping of problematic location names to valid ones
	locationMap := map[string]string{
		"willemstad-netherlands_antilles": "willemstad-curacao",
		"netherlands_antilles":            "curacao",
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

func createMapCanvas(locations []*LocationCoords) fyne.CanvasObject {
	if normalized, exists := locationMap[strings.ToLower(location)]; exists {
		log.Printf("[LOCATION NORMALIZED] %s -> %s\n", location, normalized)
		return normalized
	}
	return location
}

// dessine carte
func createMapCanvasFromAPI(locations []*models.LocationCoords) fyne.CanvasObject {
	if len(locations) == 0 {
		return canvas.NewText(T().NoLocations, ContrastColor(BgDarker))
	}

	minLat := locations[0].Lat
	maxLat := locations[0].Lat
	minLon := locations[0].Lon
	maxLon := locations[0].Lon

	// Implementation using OpenStreetMap tiles (slippy map)
	// tile size is 256x256
	const tileSize = 256

	// determine bounding box
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

	// padding
	latRange := maxLat - minLat
	lonRange := maxLon - minLon

	if latRange < 0.1 {
		latRange = 0.1
	if latRange < 0.01 {
		latRange = 0.01
	}
	if lonRange < 0.01 {
		lonRange = 0.01
	}
	padLat := latRange * 0.1
	padLon := lonRange * 0.1
	minLat -= padLat
	maxLat += padLat
	minLon -= padLon
	maxLon += padLon

	latRange = maxLat - minLat
	lonRange = maxLon - minLon

	// target canvas display size
	mapWidth := float32(1000)
	mapHeight := float32(600)

	// choose zoom to roughly fit bounding box into mapWidth/mapHeight
	// test zooms from 1..12 and pick the one where tile pixel span is >= map size
	// (12 is reasonable; 18 would load 1000+ tiles and timeout)
	chooseZoom := func() int {
		for z := 12; z >= 1; z-- {
			tx1, ty1 := latLonToTileXY(maxLat, minLon, z)
			tx2, ty2 := latLonToTileXY(minLat, maxLon, z)
			dx := math.Abs(tx2-tx1) * float64(tileSize)
			dy := math.Abs(ty2-ty1) * float64(tileSize)
			if dx >= float64(mapWidth) || dy >= float64(mapHeight) {
				return z
			}
		}
		return 2
	}

	zoom := chooseZoom()

	// compute tile ranges
	tx1, ty1 := latLonToTileXY(maxLat, minLon, zoom)
	tx2, ty2 := latLonToTileXY(minLat, maxLon, zoom)
	xMin := int(math.Floor(math.Min(tx1, tx2)))
	xMax := int(math.Ceil(math.Max(tx1, tx2)))
	yMin := int(math.Floor(math.Min(ty1, ty2)))
	yMax := int(math.Ceil(math.Max(ty1, ty2)))

	tilesX := xMax - xMin + 1
	tilesY := yMax - yMin + 1

	totalWidth := float32(tilesX * tileSize)
	totalHeight := float32(tilesY * tileSize)

	// container for tiles and overlays
	mapContainer := container.New(layout.NewMaxLayout())

	// background
	mapBg := canvas.NewRectangle(BgDarker)
	mapBg.SetMinSize(fyne.NewSize(mapWidth, mapHeight))

	mapContainer := container.New(layout.NewMaxLayout())
	mapContainer.Add(mapBg)

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

	for _, loc := range locations {
		x := toLon(loc.Lon)
		y := toLat(loc.Lat)
	// a dedicated container for absolute placement
	tileContainer := container.NewWithoutLayout()

	// fetch tiles (concurrently) into memory cache, then add UI objects on main thread
	var wg sync.WaitGroup
	client := &http.Client{Timeout: 10 * time.Second}
	tilesData := make(map[string][]byte)
	var tilesMu sync.Mutex

	// Limit concurrent goroutines to respect OSM tile server rate limits
	// OSM politely requests max 2-3 concurrent connections per IP
	maxConcurrent := 2
	semaphore := make(chan struct{}, maxConcurrent)

	for x := xMin; x <= xMax; x++ {
		for y := yMin; y <= yMax; y++ {
			wg.Add(1)
			go func(tx, ty int) {
				defer wg.Done()
				semaphore <- struct{}{}        // acquire
				defer func() { <-semaphore }() // release

				// Small delay between requests to respect OSM rate limiting
				time.Sleep(100 * time.Millisecond)

				// try reading cache or downloading
				path := tileCachePath(zoom, tx, ty)
				if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
					tilesMu.Lock()
					tilesData[fmt.Sprintf("%d_%d", tx, ty)] = b
					tilesMu.Unlock()
					return
				}
				// download
				resBytes := getTileBytes(client, zoom, tx, ty)
				if len(resBytes) > 0 {
					_ = os.WriteFile(path, resBytes, 0o644)
					tilesMu.Lock()
					tilesData[fmt.Sprintf("%d_%d", tx, ty)] = resBytes
					tilesMu.Unlock()
				}
			}(x, y)
		}
	}
	wg.Wait()

	// now create and add UI objects on main thread
	fyne.DoAndWait(func() {
		for x := xMin; x <= xMax; x++ {
			for y := yMin; y <= yMax; y++ {
				key := fmt.Sprintf("%d_%d", x, y)
				var res fyne.Resource
				if b, ok := tilesData[key]; ok && len(b) > 0 {
					res = fyne.NewStaticResource(fmt.Sprintf("%d_%d_%d.png", zoom, x, y), b)
				} else {
					res = fyne.NewStaticResource(fmt.Sprintf("%d_%d_%d.png", zoom, x, y), []byte{})
				}
				img := canvas.NewImageFromResource(res)
				img.FillMode = canvas.ImageFillContain
				img.Move(fyne.NewPos(float32((x-xMin)*tileSize), float32((y-yMin)*tileSize)))
				img.Resize(fyne.NewSize(tileSize, tileSize))
				tileContainer.Add(img)
			}
		}

		// add markers
		for _, loc := range locations {
			tx, ty := latLonToTileXY(loc.Latitude, loc.Longitude, zoom)
			px := float32((tx - float64(xMin)) * float64(tileSize))
			py := float32((ty - float64(yMin)) * float64(tileSize))

		marker := canvas.NewCircle(AccentPink)
		marker.StrokeWidth = 1
		marker.StrokeColor = AccentCyan
		marker.Move(fyne.NewPos(x-5, y-5))
		marker.Resize(fyne.NewSize(10, 10))

		mapContainer.Add(marker)

		if len(locations) <= 50 || (len(locations) > 50 && indexOfLocation(locations, loc)%3 == 0) {
			locationText := canvas.NewText(strings.Split(loc.Location, ",")[0], TextLight)
			locationText.TextSize = 9
			locationText.Move(fyne.NewPos(x+10, y-5))
			mapContainer.Add(locationText)
		}
	}

	gridColor := color.RGBA{R: 100, G: 100, B: 100, A: 30}

	for i := 0; i <= 3; i++ {
		x := float32(i) * mapWidth / 3
		line := canvas.NewLine(gridColor)
		line.StrokeWidth = 0.5
		line.Position1 = fyne.NewPos(x, 0)
		line.Position2 = fyne.NewPos(x, mapHeight)
		mapContainer.Add(line)
			marker := canvas.NewCircle(AccentPink)
			marker.StrokeWidth = 1
			marker.StrokeColor = AccentCyan
			marker.Resize(fyne.NewSize(12, 12))
			marker.Move(fyne.NewPos(px-6, py-6))
			tileContainer.Add(marker)

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
				locationText.Move(fyne.NewPos(px+10, py-5))
				tileContainer.Add(locationText)
			}
		}

		tileContainer.Resize(fyne.NewSize(totalWidth, totalHeight))
		mapContainer.Add(tileContainer)
	})

	// wrap in scroll so user can pan
	scroll := container.NewScroll(mapContainer)
	scroll.SetMinSize(fyne.NewSize(mapWidth, mapHeight))
	return scroll
}

// cache tiles on disk under $TMP/groupie-tiles
var tileCacheDir string
var tileCacheInit sync.Once

func tileCachePath(zoom, x, y int) string {
	tileCacheInit.Do(func() {
		tmp := os.TempDir()
		tileCacheDir = filepath.Join(tmp, "groupie-tiles")
		os.MkdirAll(tileCacheDir, 0o755)
	})
	return filepath.Join(tileCacheDir, fmt.Sprintf("%d_%d_%d.png", zoom, x, y))
}

func getTileResource(client *http.Client, zoom, x, y int) fyne.Resource {
	path := tileCachePath(zoom, x, y)
	if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
		return fyne.NewStaticResource(fmt.Sprintf("%d_%d_%d.png", zoom, x, y), b)
	}

	for i := 0; i <= 3; i++ {
		y := float32(i) * mapHeight / 3
		line := canvas.NewLine(gridColor)
		line.StrokeWidth = 0.5
		line.Position1 = fyne.NewPos(0, y)
		line.Position2 = fyne.NewPos(mapWidth, y)
		mapContainer.Add(line)
	}

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
	// fallback to downloading via getTileBytes
	if b := getTileBytes(client, zoom, x, y); len(b) > 0 {
		_ = os.WriteFile(path, b, 0o644)
		return fyne.NewStaticResource(fmt.Sprintf("%d_%d_%d.png", zoom, x, y), b)
	}
	return fyne.NewStaticResource("empty", []byte{})
}

// get raw bytes for a tile by trying several OSM servers
func getTileBytes(client *http.Client, zoom, x, y int) []byte {
	urls := []string{
		fmt.Sprintf("https://a.tile.openstreetmap.org/%d/%d/%d.png", zoom, x, y),
		fmt.Sprintf("https://b.tile.openstreetmap.org/%d/%d/%d.png", zoom, x, y),
		fmt.Sprintf("https://c.tile.openstreetmap.org/%d/%d/%d.png", zoom, x, y),
	}
	for i, u := range urls {
		if i > 0 {
			// Add small delay between servers to avoid rate limiting
			time.Sleep(100 * time.Millisecond)
		}
		req, _ := http.NewRequest("GET", u, nil)
		req.Header.Set("User-Agent", "groupie-tracker/1.0")
		req.Header.Set("Accept", "*/*")
		req.Header.Set("Accept-Encoding", "gzip, deflate")
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		if resp.StatusCode != 200 {
			resp.Body.Close()
			continue
		}
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}
		return data
	}
	return nil
}

func indexOfLocation(locations []*LocationCoords, target *LocationCoords) int {
// convert lat/lon to slippy map tile coords (floating)
func latLonToTileXY(lat, lon float64, zoom int) (float64, float64) {
	latRad := lat * math.Pi / 180.0
	n := math.Pow(2, float64(zoom))
	x := (lon + 180.0) / 360.0 * n
	y := (1.0 - math.Log(math.Tan(latRad)+1.0/math.Cos(latRad))/math.Pi) / 2.0 * n
	return x, y
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

func generateStaticMapURL(locations []*LocationCoords) string {
	if len(locations) == 0 {
		return ""
	}

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

	baseURL := "https://maps.googleapis.com/maps/api/staticmap?"
	params := fmt.Sprintf("center=%.6f,%.6f&zoom=3&size=1200x700&style=feature:all|element:labels|visibility:off", centerLat, centerLon)

	for i, loc := range locations {
		if i >= 50 {
			break
		}
		params += fmt.Sprintf("&markers=color:red|%.6f,%.6f", loc.Lat, loc.Lon)
	}

	return baseURL + params + "&key=AIzaSyB41DRUbKWJHPxagoK4fLi1aZjqsqOlEdE"
}

func createLocationsList(locations []*LocationCoords) *fyne.Container {
// liste lieux
func createLocationsListFromAPI(locations []*models.LocationCoords, concertsByLocation map[string][]ConcertInfo) *fyne.Container {
	var items []fyne.CanvasObject

	titleLabel := widget.NewLabel("Liste des lieux de concerts")
	// titre de la liste
	titleLabel := widget.NewLabel(T().LocationsListTitle)
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	items = append(items, titleLabel)
	items = append(items, widget.NewSeparator())

	for _, loc := range locations {
		locationName := strings.ReplaceAll(loc.Location, "_", " ")
		// conteneur pour chaque lieu
		locationName := strings.ReplaceAll(loc.Lieux, "_", " ")
		locationName = strings.ReplaceAll(locationName, "-", ", ")

		// affiche les coords fournies par l'api
		coordsText := fmt.Sprintf("(%.4f, %.4f)", loc.Latitude, loc.Longitude)
		locLabel := widget.NewLabel("• " + locationName + " " + coordsText)
		locLabel.TextStyle = fyne.TextStyle{Bold: true}

		items = append(items, locLabel)

		for _, concert := range loc.Concerts {
			artistLabel := widget.NewLabel("  " + concert.Artist)
			items = append(items, artistLabel)
		// liste les concerts associés
		if concerts, ok := concertsByLocation[loc.Lieux]; ok {
			for _, concert := range concerts {
				artistLabel := widget.NewLabel("  ♫ " + concert.Artist)
				items = append(items, artistLabel)

			maxDates := 3
			for i, date := range concert.Dates {
				if i >= maxDates {
					remaining := len(concert.Dates) - maxDates
					items = append(items, widget.NewLabel(fmt.Sprintf("      ... et %d autres dates", remaining)))
					break
				}
				dateLabel := widget.NewLabel("      • " + date)
				items = append(items, dateLabel)
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
