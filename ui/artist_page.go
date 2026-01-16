package ui

import (
	"fmt"
	"groupie-tracker/models"
	"net/url"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

// ArtistPage affiche les détails d'un artiste avec design moderne
func NewArtistPage(artist models.Artist, onBack func()) fyne.CanvasObject {
	// === BOUTON RETOUR ===
	backBtn := widget.NewButton("← Back", onBack)
	backBtn.Importance = widget.MediumImportance

	// === HEADER ===
	headerBg := canvas.NewRectangle(BgDarker)
	header := container.NewMax(
		headerBg,
		container.NewVBox(
			widget.NewLabel(""), // Spacer
			backBtn,
			widget.NewLabel(""), // Spacer
		),
	)
	header.Resize(fyne.NewSize(0, 80))

	// === CONTENU PRINCIPAL ===
	// Image de l'artiste
	uri, _ := storage.ParseURI(artist.Image)
	img := canvas.NewImageFromURI(uri)
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(350, 350))

	// Titre
	titleText := canvas.NewText(artist.Name, TextWhite)
	titleText.TextStyle = fyne.TextStyle{Bold: true}
	titleText.TextSize = 32
	titleText.Alignment = fyne.TextAlignCenter

	// Badge année création
	yearBadge := canvas.NewText(fmt.Sprintf("Created %d", artist.CreationDate), AccentCyan)
	yearBadge.TextSize = 14
	yearBadge.Alignment = fyne.TextAlignCenter

	// Premier album
	albumText := canvas.NewText(fmt.Sprintf("💿 Premier album: %s", artist.FirstAlbum), TextLight)
	albumText.TextSize = 12
	albumText.Alignment = fyne.TextAlignCenter

	// Section membres
	membersLabel := canvas.NewText("👥 Membres du groupe", TextWhite)
	membersLabel.TextStyle = fyne.TextStyle{Bold: true}
	membersLabel.TextSize = 16

	membersList := container.NewVBox()
	for _, member := range artist.Members {
		memberItem := canvas.NewText(member, TextLight)
		memberItem.TextSize = 13
		membersList.Add(container.NewPadded(memberItem))
	}

	membersSection := container.New(
		layout.NewMaxLayout(),
		container.NewPadded(
			container.NewVBox(
				membersLabel,
				widget.NewLabel(""),
				membersList,
			),
		),
	)

	// === CONTENU SCROLLABLE ===
	mainContent := container.NewVBox(
		widget.NewLabel(""), // Spacer
		container.NewCenter(img),
		container.NewCenter(titleText),
		container.NewCenter(yearBadge),
		container.NewCenter(albumText),
		widget.NewLabel(""), // Spacer
		membersSection,
		widget.NewLabel(""), // Spacer
	)

	// Charger les concerts
	concertContent := loadConcertContent(artist.ID)
	if concertContent != nil {
		mainContent.Add(concertContent)
	}

	mainContent.Add(widget.NewLabel("")) // Spacer final

	scroll := container.NewScroll(mainContent)

	// === LAYOUT FINAL avec BorderLayout ===
	return container.New(
		layout.NewBorderLayout(header, nil, nil, nil),
		header,
		scroll,
	)
}

// loadConcertContent charge et retourne le contenu des concerts
func loadConcertContent(artistID int) fyne.CanvasObject {
	relations, err := models.FetchRelations()
	if err != nil {
		errorLabel := canvas.NewText("Error loading concerts", TextLight)
		errorLabel.TextSize = 12
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
		noDataLabel := canvas.NewText("Aucun concert programmé pour le moment", TextLight)
		noDataLabel.TextSize = 12
		return noDataLabel
	}

	// Trier les lieux
	locations := make([]string, 0, len(datesLocations))
	for location := range datesLocations {
		locations = append(locations, location)
	}
	sort.Strings(locations)

	// Sections concerts
	headerLabel := canvas.NewText(fmt.Sprintf("%d Locations", len(locations)), TextWhite)
	headerLabel.TextStyle = fyne.TextStyle{Bold: true}
	headerLabel.TextSize = 16

	locationsList := container.NewVBox(
		headerLabel,
		widget.NewLabel(""),
	)

	for _, location := range locations {
		dates := datesLocations[location]
		locationItem := createLocationItem(location, dates)
		locationsList.Add(locationItem)
	}

	return container.NewPadded(locationsList)
}

// createLocationItem crée une carte pour un lieu et ses dates
func createLocationItem(location string, dates []string) *fyne.Container {
	// Formater le lieu
	formattedLoc := formatLocation(location)

	// Titre du lieu avec drapeau
	countryFlag := getCountryFlag(formattedLoc)
	locationTitle := canvas.NewText(countryFlag+" "+formattedLoc, TextWhite)
	locationTitle.TextStyle = fyne.TextStyle{Bold: true}
	locationTitle.TextSize = 14

	// Dates
	datesList := container.NewVBox(
		canvas.NewText("Dates de concert:", TextLight),
	)
	for _, date := range dates {
		dateItem := canvas.NewText("🎫 "+date, TextWhite)
		dateItem.TextSize = 16
		dateItem.TextStyle = fyne.TextStyle{Bold: true}
		datesList.Add(container.NewPadded(dateItem))
	}

	// Bouton Google Maps
	mapBtn := widget.NewButton("Voir sur Maps", func() {
		mapURL := fmt.Sprintf("https://www.google.com/maps/search/%s", url.QueryEscape(formatLocationForMap(location)))
		if parsedURL, err := url.Parse(mapURL); err == nil {
			fyne.CurrentApp().OpenURL(parsedURL)
		}
	})
	mapBtn.Importance = widget.LowImportance

	// Card avec fond sombre
	cardContent := container.NewVBox(
		locationTitle,
		widget.NewLabel(""),
		datesList,
		widget.NewLabel(""),
		mapBtn,
	)

	cardBg := canvas.NewRectangle(CardBgLight)
	cardBorder := canvas.NewRectangle(AccentPink)
	cardBorder.StrokeWidth = 2

	return container.New(
		layout.NewMaxLayout(),
		cardBorder,
		cardBg,
		container.NewPadded(cardContent),
	)
}

// formatLocation formate un lieu pour l'affichage
func formatLocation(location string) string {
	location = strings.ReplaceAll(location, "_", " ")
	location = strings.ReplaceAll(location, "-", ", ")

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
	location = strings.ReplaceAll(location, "_", " ")
	location = strings.ReplaceAll(location, "-", ",")
	return location
}

// getCountryFlag retourne l'emoji du drapeau du pays
func getCountryFlag(location string) string {
	location = strings.ToLower(strings.TrimSpace(location))

	// Extraire le dernier mot (le pays)
	parts := strings.Split(location, ",")
	if len(parts) > 0 {
		location = strings.TrimSpace(parts[len(parts)-1])
	}

	flags := map[string]string{
		"usa":         "🇺🇸",
		"uk":          "🇬🇧",
		"france":      "🇫🇷",
		"germany":     "🇩🇪",
		"spain":       "🇪🇸",
		"italy":       "🇮🇹",
		"japan":       "🇯🇵",
		"canada":      "🇨🇦",
		"australia":   "🇦🇺",
		"brazil":      "🇧🇷",
		"mexico":      "🇲🇽",
		"netherlands": "🇳🇱",
		"belgium":     "🇧🇪",
		"switzerland": "🇨🇭",
		"sweden":      "🇸🇪",
		"norway":      "🇳🇴",
		"denmark":     "🇩🇰",
		"finland":     "🇫🇮",
		"portugal":    "🇵🇹",
		"ireland":     "🇮🇪",
		"poland":      "🇵🇱",
		"austria":     "🇦🇹",
		"czech":       "🇨🇿",
		"russia":      "🇷🇺",
		"china":       "🇨🇳",
		"korea":       "🇰🇷",
		"india":       "🇮🇳",
		"argentina":   "🇦🇷",
		"chile":       "🇨🇱",
		"colombia":    "🇨🇴",
		"peru":        "🇵🇪",
		"zealand":     "🇳🇿",
		"africa":      "🇿🇦",
		"israel":      "🇮🇱",
		"turkey":      "🇹🇷",
		"greece":      "🇬🇷",
	}

	// Chercher une correspondance
	for key, flag := range flags {
		if strings.Contains(location, key) {
			return flag
		}
	}

	return "🌍" // Drapeau par défaut
}
