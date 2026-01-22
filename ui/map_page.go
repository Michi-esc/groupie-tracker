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

		// petit résumé du nombre de lieux
		infoLabel := widget.NewLabel(fmt.Sprintf("%d "+T().Location, len(concertLocations)))
		infoLabel.Alignment = fyne.TextAlignCenter
		infoLabel.TextStyle = fyne.TextStyle{Bold: true}

		// titre de la page
		title := widget.NewLabel(T().ConcertLocations)
		title.TextStyle = fyne.TextStyle{Bold: true}
		title.Alignment = fyne.TextAlignCenter

		// carte + liste côte à côte
		contentDisplay := container.NewHSplit(mapCanvas, scrollLocations)

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
func createMapCanvasFromAPI(locations []*models.LocationCoords) fyne.CanvasObject {
	if len(locations) == 0 {
		return canvas.NewText(T().NoLocations, ContrastColor(BgDarker))
	}
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
	mapContainer.Add(mapBg)

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
