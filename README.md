# Groupie Tracker - Interface UI (Partie Personne 2)

## 📁 Structure du projet

```
/
├── models/
│   └── artist.go          → Structure de données Artist
├── ui/
│   ├── window.go          → Fenêtre principale
│   ├── artist_list.go     → Grille scrollable des artistes
│   ├── artist_card.go     → Carte artiste individuelle
│   ├── artist_page.go     → Page détail artiste
│   └── shortcuts.go       → Raccourcis clavier (Ctrl+F, ESC, Ctrl+Q)
├── main.go                → Point d'entrée avec exemple
└── go.mod                 → Dépendances
```

## ✅ Fonctionnalités implémentées

### 1. Fenêtre principale (`ui/window.go`)
- Création de la fenêtre 1200x800
- Gestion du contenu dynamique
- Navigation fluide entre les vues

### 2. Liste des artistes (`ui/artist_list.go`)
- Grille responsive (220x280 par carte)
- Scrollable
- Click handler pour afficher le détail

### 3. Carte artiste (`ui/artist_card.go`)
- Affichage : Nom + Année de création
- Cliquable pour voir les détails
- TODO : Ajouter l'affichage de l'image

### 4. Page détail (`ui/artist_page.go`)
- Affiche toutes les informations de l'artiste
- Membres, dates, lieux, premier album
- Bouton retour fonctionnel
- Scrollable pour le contenu long

### 5. Raccourcis clavier (`ui/shortcuts.go`)
- **ESC** : Retour à la liste
- **Ctrl + Q** : Quitter l'application
- **Ctrl + F** : Focus recherche (callback fourni)

## 🚀 Installation et lancement

```bash
# Installer les dépendances
go mod tidy

# Lancer l'application
go run main.go
```

## 🔌 Intégration avec le backend

Le code actuel utilise des données de test dans `getDummyArtists()`.

**Pour connecter au vrai backend**, remplacer dans `main.go` :

```go
// AVANT (données test)
artists := getDummyArtists()

// APRÈS (vrai API)
artists, err := api.GetArtists()
if err != nil {
    dialog.ShowError(err, appUI.Window)
    return
}
```

## 📝 Notes importantes

### Aucun appel API direct
Tout passe par des fonctions backend du type :
```go
artists, err := api.GetArtists()
```

### Loader automatique
Le spinner s'affiche pendant le chargement :
```go
spinner := widget.NewProgressBarInfinite()
appUI.SetContent(container.NewCenter(spinner))
```

### Gestion d'erreurs
Utiliser les dialogues Fyne :
```go
dialog.ShowError(err, appUI.Window)
```

## 🎨 Améliorations possibles

1. **Images d'artistes** : Ajouter `canvas.NewImageFromURI()` dans les cartes
2. **Recherche** : Implémenter la fonction `onSearch` avec un `widget.Entry`
3. **Filtres** : Ajouter des filtres par année, membres, etc.
4. **Style** : Personnaliser les couleurs avec un thème Fyne
5. **Animations** : Ajouter des transitions entre les vues

## 📦 Dépendances

- **Fyne v2.4.3** : Framework UI cross-platform
- Go 1.21+

---

**Prêt pour l'intégration avec le backend ! 🎉**
