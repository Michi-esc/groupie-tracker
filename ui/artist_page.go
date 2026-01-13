package ui

import (
	"fmt"
	"groupie-tracker/models"
	"image/color"
	"net/url"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

// ArtistPage affiche les détails d'un artiste
func NewArtistPage(artist models.Artist, onBack func()) fyne.CanvasObject {
	// Créer le bouton retour avec un fond blanc (rectangle)
	backBtn := widget.NewButton("← Retour", onBack)

	// Créer un fond blanc pour le bouton
	bgRect := canvas.NewRectangle(color.RGBA{R: 255, G: 255, B: 255, A: 255}) // Blanc

	// Container pour le bouton avec fond
	backBtnContainer := container.NewStack(
		bgRect,
		container.NewPadded(backBtn),
	)

	// Image de l'artiste
	uri, _ := storage.ParseURI(artist.Image)
	img := canvas.NewImageFromURI(uri)
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(300, 300))

	// Nom de l'artiste
	title := widget.NewLabelWithStyle(artist.Name, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	// Informations de base
	infoText := fmt.Sprintf(`
📅 Année de création: %d
🎤 Nombre de membres: %d
💿 Premier album: %s

👥 Membres:
%s
`,
		artist.CreationDate,
		len(artist.Members),
		artist.FirstAlbum,
		strings.Join(artist.Members, "\n"),
	)

	info := widget.NewLabel(infoText)
	info.Wrapping = fyne.TextWrapWord

	// Section concerts
	concertsLabel := widget.NewLabelWithStyle("🎸 Concerts & Lieux", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Charger les informations de concerts de manière synchrone
	concertContent := loadConcertContent(artist.ID, artist.Name)

	// Contenu scrollable (sans le bouton retour)
	scrollContent := container.NewVBox(
		container.NewCenter(img),
		title,
		widget.NewSeparator(),
		info,
		widget.NewSeparator(),
		concertsLabel,
		concertContent,
	)

	scroll := container.NewScroll(scrollContent)

	// Layout avec bouton retour fixe en haut à gauche
	mainLayout := container.NewStack(
		scroll, // Contenu scrollable en fond
		container.NewPadded( // Padding pour positionner le bouton
			container.NewVBox(
				container.NewHBox(
					backBtnContainer, // Bouton en haut à gauche
				),
			),
		),
	)

	return mainLayout
}

// loadConcertContent charge et retourne le contenu des concerts avec carte
func loadConcertContent(artistID int, artistName string) fyne.CanvasObject {
	relations, err := models.FetchRelations()
	if err != nil {
		errorLabel := widget.NewLabel("❌ Erreur lors du chargement des concerts: " + err.Error())
		return errorLabel
	}

	// Trouver les relations pour cet artiste
	var datesLocations map[string][]string
	for _, rel := range relations.Index {
		if rel.ID == artistID {
			datesLocations = rel.DatesLocations
			break
		}
	}

	if datesLocations == nil || len(datesLocations) == 0 {
		noDataLabel := widget.NewLabel("Aucun concert programmé pour le moment")
		return noDataLabel
	}

	// Trier les lieux par ordre alphabétique
	locations := make([]string, 0, len(datesLocations))
	for location := range datesLocations {
		locations = append(locations, location)
	}
	sort.Strings(locations)

	// Créer le header avec le nombre de lieux
	headerLabel := widget.NewLabel(fmt.Sprintf("📍 %d lieux de concerts", len(locations)))

	// Liste des lieux avec dates et boutons de carte
	locationsList := container.NewVBox()
	for _, location := range locations {
		dates := datesLocations[location]
		locationItem := createLocationItem(location, dates)
		locationsList.Add(locationItem)
		locationsList.Add(widget.NewSeparator())
	}

	// Container principal avec la liste
	mainContainer := container.NewVBox(
		headerLabel,
		widget.NewSeparator(),
		locationsList,
	)

	return mainContainer
}

// formatLocation formate un lieu pour l'affichage
func formatLocation(location string) string {
	// Remplacer les underscores et tirets par des espaces
	location = strings.ReplaceAll(location, "_", " ")
	location = strings.ReplaceAll(location, "-", ", ")

	// Capitaliser chaque mot
	words := strings.Fields(location)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, " ")
}

// formatLocationForMap formate un lieu pour Google Maps
func formatLocationForMap(location string) string {
	// Remplacer underscores par espaces et tirets par virgules
	location = strings.ReplaceAll(location, "_", " ")
	location = strings.ReplaceAll(location, "-", ",")
	return location
}

// getCountryFlag retourne l'emoji du drapeau du pays
func getCountryFlag(country string) string {
	country = strings.ToLower(strings.TrimSpace(country))

	flags := map[string]string{
		"usa":            "🇺🇸",
		"uk":             "🇬🇧",
		"france":         "🇫🇷",
		"germany":        "🇩🇪",
		"spain":          "🇪🇸",
		"italy":          "🇮🇹",
		"japan":          "🇯🇵",
		"canada":         "🇨🇦",
		"australia":      "🇦🇺",
		"brazil":         "🇧🇷",
		"mexico":         "🇲🇽",
		"netherlands":    "🇳🇱",
		"belgium":        "🇧🇪",
		"switzerland":    "🇨🇭",
		"sweden":         "🇸🇪",
		"norway":         "🇳🇴",
		"denmark":        "🇩🇰",
		"finland":        "🇫🇮",
		"portugal":       "🇵🇹",
		"ireland":        "🇮🇪",
		"poland":         "🇵🇱",
		"austria":        "🇦🇹",
		"czech republic": "🇨🇿",
		"russia":         "🇷🇺",
		"china":          "🇨🇳",
		"south korea":    "🇰🇷",
		"india":          "🇮🇳",
		"argentina":      "🇦🇷",
		"chile":          "🇨🇱",
		"colombia":       "🇨🇴",
		"peru":           "🇵🇪",
		"new zealand":    "🇳🇿",
		"south africa":   "🇿🇦",
		"israel":         "🇮🇱",
		"turkey":         "🇹🇷",
		"greece":         "🇬🇷",
		"hungary":        "🇭🇺",
		"romania":        "🇷🇴",
		"ukraine":        "🇺🇦",
		"croatia":        "🇭🇷",
		"serbia":         "🇷🇸",
		"bulgaria":       "🇧🇬",
		"slovakia":       "🇸🇰",
		"slovenia":       "🇸🇮",
		"estonia":        "🇪🇪",
		"latvia":         "🇱🇻",
		"lithuania":      "🇱🇹",
		"luxembourg":     "🇱🇺",
		"iceland":        "🇮🇸",
		"malta":          "🇲🇹",
		"cyprus":         "🇨🇾",
	}

	if flag, ok := flags[country]; ok {
		return flag
	}

	return "🌍" // Drapeau par défaut si pays non trouvé
}

// createLocationItem crée un élément de liste pour un lieu avec ses dates
func createLocationItem(location string, dates []string) fyne.CanvasObject {
	// Formater le lieu
	formattedLocation := formatLocation(location)
	formattedForMap := formatLocationForMap(location)

	parts := strings.Split(formattedLocation, ", ")
	city := parts[0]
	country := ""
	countryFlag := ""
	if len(parts) > 1 {
		country = parts[len(parts)-1]
		countryFlag = getCountryFlag(country)
	}

	// Titre du lieu avec drapeau (en noir)
	titleText := fmt.Sprintf("📍 %s", city)
	if country != "" {
		titleText += fmt.Sprintf(" %s %s", countryFlag, country)
	}
	locationLabel := widget.NewLabel(titleText)
	locationLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Informations sur les dates (en noir)
	datesInfo := widget.NewLabel(fmt.Sprintf("   📅 %d concert(s)", len(dates)))

	// Liste des dates (limiter à 5)
	var datesDisplay []string
	if len(dates) > 5 {
		datesDisplay = dates[:5]
	} else {
		datesDisplay = dates
	}

	// Créer un container pour les dates
	datesContainer := container.NewVBox()
	for _, date := range datesDisplay {
		dateLabel := widget.NewLabel("      • " + date)
		datesContainer.Add(dateLabel)
	}

	// Ajouter "et X autres" si nécessaire
	if len(dates) > 5 {
		moreLabel := widget.NewLabel(fmt.Sprintf("      ... et %d autres dates", len(dates)-5))
		datesContainer.Add(moreLabel)
	}

	// Bouton pour voir sur la carte
	mapButton := widget.NewButton("🗺️ Voir sur la carte", func() {
		searchQuery := url.QueryEscape(formattedForMap)
		mapURL := "https://www.google.com/maps/search/" + searchQuery
		parsedURL, err := url.Parse(mapURL)
		if err == nil {
			_ = fyne.CurrentApp().OpenURL(parsedURL)
		}
	})

	// Assembler le tout
	itemContent := container.NewVBox(
		locationLabel,
		datesInfo,
		datesContainer,
		mapButton,
	)

	return itemContent
}
