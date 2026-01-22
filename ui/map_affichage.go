package ui

// current lang
var CurrentLang = "fr"

// cached translations (pre-loaded)
var cachedTranslations = map[string]Translations{}

// lang strings
type Translations struct {
	// common
	Back    string
	Search  string
	Loading string
	Error   string

	// window
	WindowTitle string

	// artist list
	Artists            string
	ShowMap            string
	SearchPlaceholder  string
	Filters            string
	ResetFilters       string
	CreationYear       string
	FirstAlbum         string
	Members            string
	Location           string
	NoResults          string
	Min                string
	Max                string
	ShowDetails        string
	DatesLabel         string
	ViewOnMaps         string
	LocationsListTitle string
	NoLocations        string
	MoreDatesFmt       string

	// artist page
	Created         string
	FirstAlbumLabel string
	GroupMembers    string
	Concerts        string
	NoConcerts      string

	// map page
	Map              string
	ConcertLocations string
	SelectLocation   string
}

var Fr = Translations{
	Back:    "← Retour",
	Search:  "Rechercher",
	Loading: "Chargement...",
	Error:   "Erreur",

	WindowTitle: "Groupie Tracker",

	Artists:            "Artistes",
	ShowMap:            "🗺️ Voir la Carte",
	SearchPlaceholder:  "Rechercher un artiste, membre, album...",
	Filters:            "Filtres",
	ResetFilters:       "Réinitialiser",
	CreationYear:       "Année de création",
	FirstAlbum:         "Premier album",
	Members:            "Membres",
	Location:           "Lieu",
	NoResults:          "Aucun artiste trouvé",
	Min:                "Min",
	Max:                "Max",
	ShowDetails:        "Voir les détails",
	DatesLabel:         "Dates de concert:",
	ViewOnMaps:         "Voir sur Maps",
	LocationsListTitle: "Liste des lieux de concerts",
	NoLocations:        "Aucun lieu de concert",
	MoreDatesFmt:       "... et %d autres dates",

	Created:         "Créé en %d",
	FirstAlbumLabel: "💿 Premier album: %s",
	GroupMembers:    "👥 Membres du groupe",
	Concerts:        "🎤 Concerts",
	NoConcerts:      "Aucune information de concert disponible",

	Map:              "Carte",
	ConcertLocations: "🗺️ Lieux de Concerts",
	SelectLocation:   "Sélectionnez un lieu pour voir les détails",
}

var En = Translations{
	Back:    "← Back",
	Search:  "Search",
	Loading: "Loading...",
	Error:   "Error",

	WindowTitle: "Groupie Tracker",

	Artists:            "Artists",
	ShowMap:            "🗺️ Show Map",
	SearchPlaceholder:  "Search artist, member, album...",
	Filters:            "Filters",
	ResetFilters:       "Reset",
	CreationYear:       "Creation Year",
	FirstAlbum:         "First Album",
	Members:            "Members",
	Location:           "Location",
	NoResults:          "No artists found",
	Min:                "Min",
	Max:                "Max",
	ShowDetails:        "Show details",
	DatesLabel:         "Concert dates:",
	ViewOnMaps:         "View on Maps",
	LocationsListTitle: "List of concert locations",
	NoLocations:        "No concert locations",
	MoreDatesFmt:       "... and %d more dates",

	Created:         "Created %d",
	FirstAlbumLabel: "💿 First album: %s",
	GroupMembers:    "👥 Group Members",
	Concerts:        "🎤 Concerts",
	NoConcerts:      "No concert information available",

	Map:              "Map",
	ConcertLocations: "🗺️ Concert Locations",
	SelectLocation:   "Select a location to see details",
}

// init translations cache (call once at startup)
func InitTranslations() {
	cachedTranslations["fr"] = Fr
	cachedTranslations["en"] = En
}

// get current translations (pre-loaded from cache)
func T() Translations {
	if trans, ok := cachedTranslations[CurrentLang]; ok {
		return trans
	}
	// fallback to dynamic lookup if cache not initialized
	if CurrentLang == "en" {
		return En
	}
	return Fr
}

// toggle lang
func ToggleLang() {
	if CurrentLang == "fr" {
		CurrentLang = "en"
	} else {
		CurrentLang = "fr"
	}
}
