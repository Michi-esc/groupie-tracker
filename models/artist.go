package models

// Artist représente un artiste avec toutes ses informations
type Artist struct {
	ID           int                 `json:"id"`
	Image        string              `json:"image"`
	Name         string              `json:"name"`
	Members      []string            `json:"members"`
	CreationDate int                 `json:"creationDate"`
	FirstAlbum   string              `json:"firstAlbum"`
	Locations    []string            `json:"locations,omitempty"`
	ConcertDates []string            `json:"concertDates,omitempty"`
	Relations    map[string][]string `json:"relations,omitempty"`
}
