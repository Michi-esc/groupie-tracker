package models

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Artist représente un artiste
type Artist struct {
	ID           int      `json:"id"`
	Image        string   `json:"image"`
	Name         string   `json:"name"`
	Members      []string `json:"members"`
	CreationDate int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
	Locations    string   `json:"locations"`
	ConcertDates string   `json:"concertDates"`
	Relations    string   `json:"relations"`

	// Champs enrichis (remplis après le fetch)
	LocationsList []string `json:"-"` // Liste des lieux de concert
	DatesList     []string `json:"-"` // Liste des dates de concert
}

// RelationData contient les relations entre lieux et dates
type RelationData struct {
	Index []struct {
		ID             int                 `json:"id"`
		DatesLocations map[string][]string `json:"datesLocations"`
	} `json:"index"`
}

var (
	cachedArtists   []Artist
	cachedRelations *RelationData
	lastFetchTime   time.Time
	cacheDuration   = 5 * time.Minute
)

// FetchArtists récupère la liste des artistes depuis l'API
func FetchArtists() ([]Artist, error) {
	// Vérifier le cache
	if time.Since(lastFetchTime) < cacheDuration && len(cachedArtists) > 0 {
		return cachedArtists, nil
	}

	resp, err := http.Get("https://groupietrackers.herokuapp.com/api/artists")
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la requête API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("erreur HTTP: %d", resp.StatusCode)
	}

	var artists []Artist
	if err := json.NewDecoder(resp.Body).Decode(&artists); err != nil {
		return nil, fmt.Errorf("erreur lors du décodage JSON: %v", err)
	}

	// Mettre à jour le cache
	cachedArtists = artists
	lastFetchTime = time.Now()

	return artists, nil
}

// FetchRelations récupère les relations entre lieux et dates
func FetchRelations() (*RelationData, error) {
	if time.Since(lastFetchTime) < cacheDuration && cachedRelations != nil {
		return cachedRelations, nil
	}

	resp, err := http.Get("https://groupietrackers.herokuapp.com/api/relation")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var relations RelationData
	if err := json.NewDecoder(resp.Body).Decode(&relations); err != nil {
		return nil, err
	}

	cachedRelations = &relations
	return &relations, nil
}
