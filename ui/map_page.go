package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"groupie-tracker/models"
	"image/color"
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
	// Créer une barre de chargement simple
	loadingLabel := widget.NewLabel(T().Loading)
	loadingBar := widget.NewProgressBarInfinite()

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

		log.Println("Creating map canvas...")
		// création de la carte canvas
		var mapCanvas fyne.CanvasObject
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Erreur dans createMapCanvas: %v\n", r)
				mapCanvas = canvas.NewText("Erreur lors de la création de la carte", ContrastColor(BgDarker))
			}
		}()

		mapCanvas = createMapCanvasFromAPI(concertLocations, concertsByLocation)
		log.Println("Map canvas created successfully")

		// petit résumé du nombre de lieux
		infoLabel := widget.NewLabel(fmt.Sprintf("%d "+T().Location, len(concertLocations)))
		infoLabel.Alignment = fyne.TextAlignCenter
		infoLabel.TextStyle = fyne.TextStyle{Bold: true}

		// titre de la page
		title := widget.NewLabel(T().ConcertLocations)
		title.TextStyle = fyne.TextStyle{Bold: true}
		title.Alignment = fyne.TextAlignCenter

		// border final
		finalContent := container.NewBorder(
			container.NewVBox(backButton, title, infoLabel),
			nil, nil, nil,
			mapCanvas,
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
	// Use Nominatim API to geocode location names
	// location format: "city-country" e.g., "london-uk", "new_york-usa"

	// Check cache first
	if cached := models.GetCachedCoords(location); cached != nil {
		return cached
	}

	// Apply location mappings for problematic/outdated location names
	normalizedLocation := normalizeLocation(location)

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

	// Cache the result
	models.CacheCoords(location, coords)

	// respecter le rate limit de Nominatim (1 req/sec minimum)
	time.Sleep(800 * time.Millisecond)

	return coords
}

func getApproxCoordinates(location string) *models.LocationCoords {
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
	}

	if coords, ok := countryCoords[location]; ok {
		return &models.LocationCoords{
			Lieux:     location,
			Latitude:  coords[0],
			Longitude: coords[1],
		}
	}

	return nil
}

// normalizeLocation maps outdated or problematic location names to current valid ones
func normalizeLocation(location string) string {
	// Mapping of problematic location names to valid ones
	locationMap := map[string]string{
		"willemstad-netherlands_antilles": "willemstad-curacao",
		"netherlands_antilles":            "curacao",
	}

	if normalized, exists := locationMap[strings.ToLower(location)]; exists {
		log.Printf("[LOCATION NORMALIZED] %s -> %s\n", location, normalized)
		return normalized
	}
	return location
}

// dessine carte
func createMapCanvasFromAPI(locations []*models.LocationCoords, concertsByLocation map[string][]ConcertInfo) fyne.CanvasObject {
	log.Printf("createMapCanvasFromAPI called with %d locations\n", len(locations))
	if len(locations) == 0 {
		return canvas.NewText(T().NoLocations, ContrastColor(BgDarker))
	}

	// Implementation using OpenStreetMap tiles (slippy map)
	// tile size is 256x256
	const tileSize = 256

	log.Println("Determining bounding box...")
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
	}
	if lonRange < 0.1 {
		lonRange = 0.1
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
	mapWidth := float32(1600)
	mapHeight := float32(900)

	// choose zoom to roughly fit bounding box into mapWidth/mapHeight
	// test zooms from 1..6 and pick the one where tile pixel span is >= map size
	// (6 is reasonable for worldwide view; 12+ would load millions of tiles)
	chooseZoom := func() int {
		for z := 3; z >= 1; z-- {
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
	maxTiles := 16 // cap très bas pour voir le monde entier
	log.Printf("Chosen zoom level: %d\n", zoom)

	// compute tile ranges
	recompute := func(z int) (int, int, int, int, int, int) {
		tx1, ty1 := latLonToTileXY(maxLat, minLon, z)
		tx2, ty2 := latLonToTileXY(minLat, maxLon, z)
		xMin := int(math.Floor(math.Min(tx1, tx2)))
		xMax := int(math.Ceil(math.Max(tx1, tx2)))
		yMin := int(math.Floor(math.Min(ty1, ty2)))
		yMax := int(math.Ceil(math.Max(ty1, ty2)))
		return xMin, xMax, yMin, yMax, xMax - xMin + 1, yMax - yMin + 1
	}

	xMin, xMax, yMin, yMax, tilesX, tilesY := recompute(zoom)
	totalTiles := tilesX * tilesY

	// Clamp tile coordinates to valid range for the zoom level
	// At zoom N, tiles go from 0 to 2^N - 1
	maxTileCoord := int(math.Pow(2, float64(zoom))) - 1
	if xMin < 0 {
		xMin = 0
	}
	if xMax > maxTileCoord {
		xMax = maxTileCoord
	}
	if yMin < 0 {
		yMin = 0
	}
	if yMax > maxTileCoord {
		yMax = maxTileCoord
	}
	tilesX = xMax - xMin + 1
	tilesY = yMax - yMin + 1
	totalTiles = tilesX * tilesY

	if totalTiles > maxTiles {
		for z := zoom - 1; z >= 1; z-- {
			xMin, xMax, yMin, yMax, tilesX, tilesY = recompute(z)

			// Clamp again after recomputing
			maxTileCoord = int(math.Pow(2, float64(z))) - 1
			if xMin < 0 {
				xMin = 0
			}
			if xMax > maxTileCoord {
				xMax = maxTileCoord
			}
			if yMin < 0 {
				yMin = 0
			}
			if yMax > maxTileCoord {
				yMax = maxTileCoord
			}
			tilesX = xMax - xMin + 1
			tilesY = yMax - yMin + 1
			totalTiles = tilesX * tilesY

			if totalTiles <= maxTiles {
				log.Printf("Tile grid too large at zoom %d, clamped to zoom %d (%d tiles)\n", zoom, z, totalTiles)
				zoom = z
				break
			}
		}
	}

	log.Printf("Tile grid: %dx%d (total %d tiles)\n", tilesX, tilesY, totalTiles)

	totalWidth := float32(tilesX * tileSize)
	totalHeight := float32(tilesY * tileSize)

	// container for tiles and overlays
	mapContainer := container.New(layout.NewMaxLayout())

	// a dedicated container for absolute placement
	tileContainer := container.NewWithoutLayout()

	// Create placeholders immediately so map shows instantly
	log.Println("Creating placeholder grid...")
	for x := xMin; x <= xMax; x++ {
		for y := yMin; y <= yMax; y++ {
			rect := canvas.NewRectangle(ContrastColor(BgDarker))
			rect.Move(fyne.NewPos(float32((x-xMin)*tileSize), float32((y-yMin)*tileSize)))
			rect.Resize(fyne.NewSize(tileSize, tileSize))
			tileContainer.Add(rect)
		}
	}

	tileContainer.Resize(fyne.NewSize(totalWidth, totalHeight))
	mapContainer.Add(tileContainer)

	// Separate container for markers (on top of tiles)
	markerContainer := container.NewWithoutLayout()

	// Container for tooltips (on top of everything)
	tooltipContainer := container.NewWithoutLayout()

	// Add markers with hover tooltips
	for _, loc := range locations {
		tx, ty := latLonToTileXY(loc.Latitude, loc.Longitude, zoom)
		px := float32((tx - float64(xMin)) * float64(tileSize))
		py := float32((ty - float64(yMin)) * float64(tileSize))

		marker := canvas.NewCircle(AccentPink)
		marker.StrokeWidth = 1
		marker.StrokeColor = AccentCyan
		marker.Resize(fyne.NewSize(12, 12))
		marker.Move(fyne.NewPos(px-6, py-6))
		markerContainer.Add(marker)

		// Create tooltip text for this location
		locationName := strings.ReplaceAll(loc.Lieux, "_", " ")
		locationName = strings.ReplaceAll(locationName, "-", ", ")

		// Build tooltip content as vertical container
		tooltipContent := container.NewVBox()

		// Location name
		locationText := canvas.NewText(locationName, color.Black)
		locationText.TextSize = 12
		locationText.TextStyle = fyne.TextStyle{Bold: true}
		tooltipContent.Add(locationText)
		tooltipContent.Add(widget.NewSeparator())

		// Get concerts for this location
		if concerts, ok := concertsByLocation[loc.Lieux]; ok {
			for _, concert := range concerts {
				artistText := canvas.NewText(concert.Artist, color.Black)
				artistText.TextSize = 11
				artistText.TextStyle = fyne.TextStyle{Bold: true}
				tooltipContent.Add(artistText)

				for _, date := range concert.Dates {
					dateText := canvas.NewText(date, color.Black)
					dateText.TextSize = 10
					tooltipContent.Add(dateText)
				}
			}
		} else {
			noConcertText := canvas.NewText("(Pas de concerts)", color.Black)
			noConcertText.TextSize = 10
			tooltipContent.Add(noConcertText)
		}

		// Create tooltip box with white background
		tooltip := canvas.NewRectangle(color.White)
		tooltipBox := container.NewStack(tooltip, tooltipContent)
		tooltipBox.Resize(fyne.NewSize(250, 0)) // Width fixed, height auto
		tooltipBox.Move(fyne.NewPos(px+15, py-10))
		tooltipBox.Hide()

		tooltipContainer.Add(tooltipBox)

		// Create custom tappable for hover detection
		tappable := widget.NewButton("", nil)
		tappable.Importance = widget.LowImportance
		tappable.Resize(fyne.NewSize(24, 24))
		tappable.Move(fyne.NewPos(px-12, py-12))

		// Store reference for closure
		currentTooltip := tooltipBox

		tappable.OnTapped = func() {
			if currentTooltip.Visible() {
				currentTooltip.Hide()
			} else {
				currentTooltip.Show()
			}
		}

		markerContainer.Add(tappable)
	}

	markerContainer.Resize(fyne.NewSize(totalWidth, totalHeight))
	mapContainer.Add(markerContainer)

	tooltipContainer.Resize(fyne.NewSize(totalWidth, totalHeight))
	mapContainer.Add(tooltipContainer)

	scroll := container.NewScroll(mapContainer)

	// Now fetch tiles in background and update placeholders when ready
	go func() {
		var wg sync.WaitGroup
		client := &http.Client{Timeout: 5 * time.Second}
		tilesData := make(map[string][]byte)
		var tilesMu sync.Mutex

		maxConcurrent := 4
		semaphore := make(chan struct{}, maxConcurrent)

		tilesDownloaded := 0
		for x := xMin; x <= xMax; x++ {
			for y := yMin; y <= yMax; y++ {
				wg.Add(1)
				go func(tx, ty int) {
					defer wg.Done()
					semaphore <- struct{}{}        // acquire
					defer func() { <-semaphore }() // release

					time.Sleep(30 * time.Millisecond)

					path := tileCachePath(zoom, tx, ty)
					if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
						if looksLikePNG(b) {
							tilesMu.Lock()
							tilesData[fmt.Sprintf("%d_%d", tx, ty)] = b
							tilesMu.Unlock()

							// Update UI immediately
							fyne.Do(func() {
								res := fyne.NewStaticResource(fmt.Sprintf("%d_%d_%d.png", zoom, tx, ty), b)
								img := canvas.NewImageFromResource(res)
								img.FillMode = canvas.ImageFillContain
								img.Move(fyne.NewPos(float32((tx-xMin)*tileSize), float32((ty-yMin)*tileSize)))
								img.Resize(fyne.NewSize(tileSize, tileSize))
								tileContainer.Add(img)
								tileContainer.Refresh()
							})
							return
						}
						_ = os.Remove(path)
					}

					resBytes := getTileBytes(client, zoom, tx, ty)
					if len(resBytes) > 0 {
						_ = os.WriteFile(path, resBytes, 0o644)
						tilesMu.Lock()
						tilesData[fmt.Sprintf("%d_%d", tx, ty)] = resBytes
						tilesDownloaded++
						if tilesDownloaded%10 == 0 || tilesDownloaded == tilesX*tilesY {
							log.Printf("Downloading tiles: %d/%d\n", tilesDownloaded, tilesX*tilesY)
						}
						tilesMu.Unlock()

						// Update UI immediately
						fyne.Do(func() {
							res := fyne.NewStaticResource(fmt.Sprintf("%d_%d_%d.png", zoom, tx, ty), resBytes)
							img := canvas.NewImageFromResource(res)
							img.FillMode = canvas.ImageFillContain
							img.Move(fyne.NewPos(float32((tx-xMin)*tileSize), float32((ty-yMin)*tileSize)))
							img.Resize(fyne.NewSize(tileSize, tileSize))
							tileContainer.Add(img)
							tileContainer.Refresh()
						})
					}
				}(x, y)
			}
		}
		wg.Wait()
		log.Printf("Finished downloading %d tiles (expected %d)\n", len(tilesData), tilesX*tilesY)
	}()

	log.Println("Map canvas completed (loading tiles in background)")
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
		if !looksLikePNG(data) {
			log.Printf("[TILE INVALID] zoom=%d x=%d y=%d: not PNG (maybe rate limited)\n", zoom, x, y)
			continue
		}
		return data
	}
	return nil
}

// quick signature check to avoid passing HTML/error pages to Fyne image loader
func looksLikePNG(b []byte) bool {
	if len(b) < 8 {
		return false
	}
	return string(b[:8]) == "\x89PNG\r\n\x1a\n"
}

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
