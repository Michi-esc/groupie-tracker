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

func NewArtistPage(artist models.Artist, onBack func()) fyne.CanvasObject {
	backBtn := widget.NewButton("← Back", onBack)
	backBtn.Importance = widget.MediumImportance

	headerBg := canvas.NewRectangle(BgDarker)
	header := container.NewMax(
		headerBg,
		container.NewVBox(
			widget.NewLabel(""),
			backBtn,
			widget.NewLabel(""),
		),
	)
	header.Resize(fyne.NewSize(0, 80))

	uri, _ := storage.ParseURI(artist.Image)
	img := canvas.NewImageFromURI(uri)
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(350, 350))

	titleText := canvas.NewText(artist.Name, TextWhite)
	titleText.TextStyle = fyne.TextStyle{Bold: true}
	titleText.TextSize = 32
	titleText.Alignment = fyne.TextAlignCenter

	yearBadge := canvas.NewText(fmt.Sprintf("Created %d", artist.CreationDate), AccentCyan)
	yearBadge.TextSize = 14
	yearBadge.Alignment = fyne.TextAlignCenter

	albumText := canvas.NewText(fmt.Sprintf("First album: %s", artist.FirstAlbum), TextLight)
	albumText.TextSize = 12
	albumText.Alignment = fyne.TextAlignCenter

	membersLabel := canvas.NewText("Band Members", TextWhite)
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

	mainContent := container.NewVBox(
		widget.NewLabel(""),
		container.NewCenter(img),
		container.NewCenter(titleText),
		container.NewCenter(yearBadge),
		container.NewCenter(albumText),
		widget.NewLabel(""),
		membersSection,
		widget.NewLabel(""),
	)

	concertContent := loadConcertContent(artist.ID)
	if concertContent != nil {
		mainContent.Add(concertContent)
	}

	mainContent.Add(widget.NewLabel(""))

	scroll := container.NewScroll(mainContent)

	return container.New(
		layout.NewBorderLayout(header, nil, nil, nil),
		header,
		scroll,
	)
}

func loadConcertContent(artistID int) fyne.CanvasObject {
	relations, err := models.FetchRelations()
	if err != nil {
		errorLabel := canvas.NewText("Error loading concerts", TextLight)
		errorLabel.TextSize = 12
		return errorLabel
	}

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

	locations := make([]string, 0, len(datesLocations))
	for location := range datesLocations {
		locations = append(locations, location)
	}
	sort.Strings(locations)

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

func createLocationItem(location string, dates []string) *fyne.Container {
	formattedLoc := formatLocation(location)

	countryFlag := getCountryFlag(formattedLoc)
	locationTitle := canvas.NewText(countryFlag+" "+formattedLoc, TextWhite)
	locationTitle.TextStyle = fyne.TextStyle{Bold: true}
	locationTitle.TextSize = 14

	datesList := container.NewVBox(
		canvas.NewText("Dates de concert:", TextLight),
	)
	for _, date := range dates {
		dateItem := canvas.NewText(date, TextWhite)
		dateItem.TextSize = 16
		dateItem.TextStyle = fyne.TextStyle{Bold: true}
		datesList.Add(container.NewPadded(dateItem))
	}

	mapBtn := widget.NewButton("Voir sur Maps", func() {
		mapURL := fmt.Sprintf("https://www.google.com/maps/search/%s", url.QueryEscape(formatLocationForMap(location)))
		if parsedURL, err := url.Parse(mapURL); err == nil {
			fyne.CurrentApp().OpenURL(parsedURL)
		}
	})
	mapBtn.Importance = widget.LowImportance

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

func formatLocationForMap(location string) string {
	location = strings.ReplaceAll(location, "_", " ")
	location = strings.ReplaceAll(location, "-", ",")
	return location
}

func getCountryFlag(location string) string {
	location = strings.ToLower(strings.TrimSpace(location))

	parts := strings.Split(location, ",")
	if len(parts) > 0 {
		location = strings.TrimSpace(parts[len(parts)-1])
	}

	return ""
}
