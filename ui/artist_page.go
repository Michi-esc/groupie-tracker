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

// page détails artiste
func NewArtistPage(artist models.Artist, onBack func()) fyne.CanvasObject {
	// bouton retour
	backBtn := widget.NewButton(T().Back, onBack)
	backBtn.Importance = widget.MediumImportance

	// header
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

	// contenu
	// image
	uri, _ := storage.ParseURI(artist.Image)
	img := canvas.NewImageFromURI(uri)
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(350, 350))

	// titre
	titleText := canvas.NewText(artist.Name, TextWhite)
	titleText.TextStyle = fyne.TextStyle{Bold: true}
	titleText.TextSize = 32
	titleText.Alignment = fyne.TextAlignCenter

	// Badge année création
	yearBadge := canvas.NewText(fmt.Sprintf(T().Created, artist.CreationDate), AccentCyan)
	yearBadge.TextSize = 14
	yearBadge.Alignment = fyne.TextAlignCenter

	// album
	albumText := canvas.NewText(fmt.Sprintf(T().FirstAlbumLabel, artist.FirstAlbum), TextLight)
	albumText.TextSize = 12
	albumText.Alignment = fyne.TextAlignCenter

	// membres
	membersLabel := canvas.NewText(T().GroupMembers, ContrastColor(CardBg))
	membersLabel.TextStyle = fyne.TextStyle{Bold: true}
	membersLabel.TextSize = 16

	membersList := container.NewVBox()
	for _, member := range artist.Members {
		memberItem := canvas.NewText(member, ContrastColor(CardBg))
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

	// contenu scrollable
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

	// on charge les concerts
	concertContent := loadConcertContent(artist.ID)
	if concertContent != nil {
		mainContent.Add(concertContent)
	}

	mainContent.Add(widget.NewLabel("")) // Spacer final

	scroll := container.NewScroll(mainContent)

	// mise en page finale avec border layout
	return container.New(
		layout.NewBorderLayout(header, nil, nil, nil),
		header,
		scroll,
	)
}

// load concerts
func loadConcertContent(artistID int) fyne.CanvasObject {
	relations, err := models.FetchRelations()
	if err != nil {
		errorLabel := canvas.NewText(fmt.Sprintf(T().Error+": %v", err), ContrastColor(CardBg))
		errorLabel.TextSize = 12
		return errorLabel
	}

	// on cherche les relations pour cet artiste
	var datesLocations map[string][]string
	for _, rel := range relations.Index {
		if rel.ID == artistID {
			datesLocations = rel.DatesLocations
			break
		}
	}

	if datesLocations == nil || len(datesLocations) == 0 {
		noDataLabel := canvas.NewText(T().NoConcerts, ContrastColor(CardBg))
		noDataLabel.TextSize = 12
		return noDataLabel
	}

	// on trie les lieux
	locations := make([]string, 0, len(datesLocations))
	for location := range datesLocations {
		locations = append(locations, location)
	}
	sort.Strings(locations)

	// header de la section concerts
	headerLabel := canvas.NewText(fmt.Sprintf("%d "+T().Location, len(locations)), ContrastColor(CardBg))
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

// carte lieu+dates
func createLocationItem(location string, dates []string) *fyne.Container {
	// on reformate le nom du lieu
	formattedLoc := formatLocation(location)

	// titre du lieu + drapeau
	countryFlag := getCountryFlag(formattedLoc)
	locationTitle := canvas.NewText(countryFlag+" "+formattedLoc, ContrastColor(CardBgLight))
	locationTitle.TextStyle = fyne.TextStyle{Bold: true}
	locationTitle.TextSize = 14

	// liste des dates
	datesList := container.NewVBox(
		canvas.NewText(T().DatesLabel, ContrastColor(CardBgLight)),
	)
	for _, date := range dates {
		dateItem := canvas.NewText("🎫 "+date, ContrastColor(CardBgLight))
		dateItem.TextSize = 16
		dateItem.TextStyle = fyne.TextStyle{Bold: true}
		datesList.Add(container.NewPadded(dateItem))
	}

	// bouton vers google maps
	mapBtn := widget.NewButton(T().ViewOnMaps, func() {
		mapURL := fmt.Sprintf("https://www.google.com/maps/search/%s", url.QueryEscape(formatLocationForMap(location)))
		if parsedURL, err := url.Parse(mapURL); err == nil {
			fyne.CurrentApp().OpenURL(parsedURL)
		}
	})
	mapBtn.Importance = widget.LowImportance

	// carte sombre pour chaque lieu
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

// format lieu
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

// format pour maps
func formatLocationForMap(location string) string {
	location = strings.ReplaceAll(location, "_", " ")
	location = strings.ReplaceAll(location, "-", ",")
	return location
}

// emoji drapeau
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
