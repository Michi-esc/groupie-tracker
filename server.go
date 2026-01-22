package main

import (
	"encoding/json"
	"fmt"
	"groupie-tracker/models"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type ArtistWithDetails struct {
	ID             int                 `json:"id"`
	Image          string              `json:"image"`
	Name           string              `json:"name"`
	Members        []string            `json:"members"`
	CreationDate   int                 `json:"creationDate"`
	FirstAlbum     string              `json:"firstAlbum"`
	FirstAlbumYear int                 `json:"firstAlbumYear"`
	Locations      []string            `json:"locations"`
	DatesLocations map[string][]string `json:"datesLocations"`
}

func serveur() {
	http.HandleFunc("/api/artists", handleGetArtists)

	fs := http.FileServer(http.Dir("./web"))
	http.Handle("/", fs)

	host := "localhost"
	port := "3000"
	addr := ":" + port

	fmt.Printf("🚀 Serveur lancé sur http://%s:%s\n", host, port)
	fmt.Println("📱 Ouvrez votre navigateur à cette adresse")
	fmt.Println("⏹️  Appuyez sur Ctrl+C pour arrêter le serveur")

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal("Erreur lors du démarrage du serveur:", err)
	}
}

func handleGetArtists(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	artists, err := models.FetchArtists()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	relations, err := models.FetchRelations()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	artistsWithDetails := make([]ArtistWithDetails, 0, len(artists))

	for _, artist := range artists {
		details := ArtistWithDetails{
			ID:           artist.ID,
			Image:        artist.Image,
			Name:         artist.Name,
			Members:      artist.Members,
			CreationDate: artist.CreationDate,
			FirstAlbum:   artist.FirstAlbum,
		}

		details.FirstAlbumYear = extractYear(artist.FirstAlbum)

		for _, rel := range relations.Index {
			if rel.ID == artist.ID {
				details.DatesLocations = rel.DatesLocations
				for loc := range rel.DatesLocations {
					details.Locations = append(details.Locations, loc)
				}
				break
			}
		}

		artistsWithDetails = append(artistsWithDetails, details)
	}

	json.NewEncoder(w).Encode(artistsWithDetails)
}

func extractYear(dateStr string) int {
	parts := strings.Split(dateStr, "-")
	if len(parts) == 3 {
		year, err := strconv.Atoi(parts[2])
		if err == nil {
			return year
		}
	}
	return 0
}
