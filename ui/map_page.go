package ui

import (
	"encoding/json"
	"fmt"
	"groupie-tracker/models"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
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
	loadingLabel := widget.NewLabel("⏳ Initialisation de la carte...")
	loadingBar := widget.NewProgressBarInfinite()

	// Créer le bouton retour
	backButton := widget.NewButton("← Retour à la liste", onBack)
	backButton.Importance = widget.HighImportance

	// Container temporaire avec chargement
	tempContainer := container.NewVBox(
		backButton,
		widget.NewLabel("🗺️ Carte des Concerts"),
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
			loadingLabel.SetText("⏳ Géocodage des lieux de concerts...")
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
			loadingLabel.SetText(fmt.Sprintf("⏳ Géocodage: 0/%d lieux", totalLocations))
		})

		// Géocoder en parallèle avec des goroutines
		numWorkers := 8
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

		// Goroutine pour compter les résultats
		countChan := make(chan bool)
		go func() {
			for range countChan {
				successCount++
				fyne.Do(func() {
					loadingLabel.SetText(fmt.Sprintf("⏳ Géocodage: %d/%d lieux trouvés", successCount, totalLocations))
				})
			}
		}()

		// Récupérer les coordonnées
		go func() {
			for coords := range resultChan {
				if coords != nil {
					locations = append(locations, coords)
					countChan <- true
				}
			}
			close(countChan)
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

		// Générer l'URL de l'image de carte statique
		fyne.Do(func() {
			loadingLabel.SetText(fmt.Sprintf("🗺️  Génération de la carte avec %d marqueurs...", len(locations)))
		})
		mapImageURL := generateStaticMapURL(locations)

		// Télécharger l'image avec retry
		var resp *http.Response
		var imgErr error
		for retry := 0; retry < 3; retry++ {
			resp, imgErr = http.Get(mapImageURL)
			if imgErr == nil && resp.StatusCode == 200 {
				break
			}
			if resp != nil {
				resp.Body.Close()
			}
			if retry < 2 {
				finalRetry := retry
				fyne.Do(func() {
					loadingLabel.SetText(fmt.Sprintf("🗺️  Génération de la carte... (tentative %d/3)", finalRetry+2))
				})
				time.Sleep(500 * time.Millisecond)
			}
		}

		if imgErr != nil {
			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   "Erreur",
				Content: fmt.Sprintf("Impossible de charger la carte: %v", imgErr),
			})
			return
		}
		defer resp.Body.Close()

		// Lire l'image
		imgData, err := io.ReadAll(resp.Body)
		if err != nil {
			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   "Erreur",
				Content: fmt.Sprintf("Erreur de lecture de l'image: %v", err),
			})
			return
		}

		fyne.Do(func() {
			loadingLabel.SetText("✅ Finalisation...")
		})
		time.Sleep(300 * time.Millisecond)

		// Créer une ressource statique à partir des données
		resource := fyne.NewStaticResource("map.png", imgData)

		// Créer l'image
		mapImage := canvas.NewImageFromResource(resource)
		mapImage.FillMode = canvas.ImageFillContain
		mapImage.SetMinSize(fyne.NewSize(1000, 600))

		// Créer la liste des lieux sous la carte
		locationsList := createLocationsList(locations)

		// Créer le scroll pour la liste
		scrollLocations := container.NewScroll(locationsList)
		scrollLocations.SetMinSize(fyne.NewSize(300, 600))

		// Info
		infoLabel := widget.NewLabel(fmt.Sprintf("✅ %d lieux de concerts trouvés et affichés", len(locations)))
		infoLabel.Alignment = fyne.TextAlignCenter
		infoLabel.TextStyle = fyne.TextStyle{Bold: true}

		// Titre
		title := widget.NewLabel("🗺️ Carte des Concerts")
		title.TextStyle = fyne.TextStyle{Bold: true}
		title.Alignment = fyne.TextAlignCenter

		// Conteneur pour l'image
		imageContainer := container.NewScroll(mapImage)

		// Diviser en deux colonnes : carte à gauche, liste à droite
		contentDisplay := container.NewHSplit(imageContainer, scrollLocations)

		// Créer le container final avec header
		finalContent := container.NewBorder(
			container.NewVBox(backButton, title, infoLabel),
			nil, nil, nil,
			contentDisplay,
		)

		// Mettre à jour le contenu de la window (SetContent est déjà thread-safe)
		win.SetContent(finalContent)
	}()
}

// geocodeLocationFast utilise Nominatim pour obtenir les coordonnées avec cache et timeout
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

	// Appeler l'API Nominatim avec timeout court
	url := fmt.Sprintf("https://nominatim.openstreetmap.org/search?format=json&q=%s&limit=1",
		strings.ReplaceAll(cleanLocation, " ", "+"))

	// Client avec timeout court (2 secondes)
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

	result := &LocationCoords{
		Lat:      lat,
		Lon:      lon,
		Location: location,
	}

	// Mettre en cache
	cacheMutex.Lock()
	geocodeCache[location] = result
	cacheMutex.Unlock()

	return result
}

// generateStaticMapURL génère une URL pour une image de carte statique avec marqueurs
func generateStaticMapURL(locations []*LocationCoords) string {
	// Calculer le centre et le zoom optimal
	var minLat, maxLat, minLon, maxLon float64
	if len(locations) > 0 {
		minLat = locations[0].Lat
		maxLat = locations[0].Lat
		minLon = locations[0].Lon
		maxLon = locations[0].Lon

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
	}

	centerLat := (minLat + maxLat) / 2
	centerLon := (minLon + maxLon) / 2

	// Construire l'URL pour l'API de carte statique
	// Utilisation de staticmap.openstreetmap.de (gratuit)
	baseURL := "https://staticmap.openstreetmap.de/staticmap.php?"
	params := fmt.Sprintf("center=%.6f,%.6f&zoom=3&size=1200x700&maptype=mapnik", centerLat, centerLon)

	// Ajouter les marqueurs (limiter à 50 pour ne pas surcharger l'URL)
	maxMarkers := 50
	for i, loc := range locations {
		if i >= maxMarkers {
			break
		}
		params += fmt.Sprintf("&markers=%.6f,%.6f,red", loc.Lat, loc.Lon)
	}

	return baseURL + params
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

		locLabel := widget.NewLabel("📍 " + locationName)
		locLabel.TextStyle = fyne.TextStyle{Bold: true}

		items = append(items, locLabel)

		// Ajouter les concerts pour ce lieu
		for _, concert := range loc.Concerts {
			artistLabel := widget.NewLabel("  🎵 " + concert.Artist)
			items = append(items, artistLabel)

			// Afficher quelques dates
			maxDates := 3
			for i, date := range concert.Dates {
				if i >= maxDates {
					remaining := len(concert.Dates) - maxDates
					items = append(items, widget.NewLabel(fmt.Sprintf("      ... et %d autres dates", remaining)))
					break
				}
				dateLabel := widget.NewLabel("      📅 " + date)
				items = append(items, dateLabel)
			}
		}

		items = append(items, widget.NewSeparator())
	}

	return container.NewVBox(items...)
}
