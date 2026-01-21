package models

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// struct artiste
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

	// ajoutés après fetch
	LocationsList []string `json:"-"` // lieux de concert
	DatesList     []string `json:"-"` // dates de concert
}

// relations lieux/dates
type RelationData struct {
	Index []struct {
		ID             int                 `json:"id"`
		DatesLocations map[string][]string `json:"datesLocations"`
	} `json:"index"`
}

// lieux avec coords
type LocationData struct {
	Index []struct {
		ID    int      `json:"id"`
		Lieux []string `json:"lieux"`
	} `json:"index"`
}

// coords lieu
type LocationCoords struct {
	ID        int
	Lieux     string // Format: "latitude,longitude"
	Latitude  float64
	Longitude float64
}

var (
	cachedArtists   []Artist
	cachedRelations *RelationData
	cachedLocations *LocationData
	lastFetchTime   time.Time
	cacheDuration   = 5 * time.Minute
	httpClient      = &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			DialContext:         (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
			IdleConnTimeout:     90 * time.Second,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
		},
	}
)

// fetch artistes depuis api
func FetchArtists() ([]Artist, error) {
	// check cache
	if time.Since(lastFetchTime) < cacheDuration && len(cachedArtists) > 0 {
		return cachedArtists, nil
	}

	resp, err := doGet(apiURL("artists"))
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la requête API artists: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("erreur HTTP: %d", resp.StatusCode)
	}

	var artists []Artist
	if err := json.NewDecoder(resp.Body).Decode(&artists); err != nil {
		return nil, fmt.Errorf("erreur lors du décodage JSON: %v", err)
	}

	// update cache
	cachedArtists = artists
	lastFetchTime = time.Now()

	return artists, nil
}

// fetch relations
func FetchRelations() (*RelationData, error) {
	if time.Since(lastFetchTime) < cacheDuration && cachedRelations != nil {
		return cachedRelations, nil
	}

	resp, err := doGet(apiURL("relation"))
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la requête API relations: %v", err)
	}
	defer resp.Body.Close()

	var relations RelationData
	if err := json.NewDecoder(resp.Body).Decode(&relations); err != nil {
		return nil, err
	}

	cachedRelations = &relations
	return &relations, nil
}

// fetch locations avec coords
func FetchLocations() (*LocationData, error) {
	if time.Since(lastFetchTime) < cacheDuration && cachedLocations != nil {
		return cachedLocations, nil
	}

	resp, err := doGet(apiURL("locations"))
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la requête API locations: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("erreur HTTP: %d", resp.StatusCode)
	}

	var locations LocationData
	if err := json.NewDecoder(resp.Body).Decode(&locations); err != nil {
		return nil, fmt.Errorf("erreur lors du décodage JSON locations: %v", err)
	}

	cachedLocations = &locations
	return &locations, nil
}

// apiURL builds the API endpoint from environment variable or default.
func apiURL(p string) string {
	base := os.Getenv("GROUPIE_BASE_URL")
	if base == "" {
		base = "https://groupietrackers.herokuapp.com/api"
	}
	base = strings.TrimRight(base, "/")
	p = strings.TrimLeft(p, "/")
	return fmt.Sprintf("%s/%s", base, p)
}

// doGet performs an HTTP GET with retries for transient errors and 5xx responses.
func doGet(url string) (*http.Response, error) {
	var resp *http.Response
	var err error
	maxAttempts := 3
	backoff := 1 * time.Second
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		start := time.Now()
		resp, err = httpClient.Get(url)
		duration := time.Since(start)

		if err == nil {
			// if server error, close and retry
			if resp.StatusCode >= 500 && resp.StatusCode < 600 {
				resp.Body.Close()
				log.Printf("GET %s -> server error %d (attempt %d/%d) in %s", url, resp.StatusCode, attempt, maxAttempts, duration)
				err = fmt.Errorf("server error: %d", resp.StatusCode)
			} else {
				log.Printf("GET %s -> %d in %s", url, resp.StatusCode, duration)
				return resp, nil
			}
		} else {
			log.Printf("GET %s -> error: %v (attempt %d/%d) in %s", url, err, attempt, maxAttempts, duration)
		}

		if attempt < maxAttempts {
			time.Sleep(backoff)
			backoff *= 2
		}
	}
	return nil, err
}
