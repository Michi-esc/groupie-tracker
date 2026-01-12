package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	// Servir les fichiers statiques du dossier web
	fs := http.FileServer(http.Dir("./web"))
	http.Handle("/", fs)

	port := ":3000"
	fmt.Printf("🚀 Serveur lancé sur http://localhost%s\n", port)
	fmt.Println("📱 Ouvrez votre navigateur à cette adresse")
	fmt.Println("⏹️  Appuyez sur Ctrl+C pour arrêter le serveur")

	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal("Erreur lors du démarrage du serveur:", err)
	}
}
