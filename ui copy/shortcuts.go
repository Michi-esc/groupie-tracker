package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

// RegisterShortcuts enregistre les raccourcis clavier obligatoires
// Raccourcis disponibles:
//   - Ctrl + F : Focus recherche
//   - ESC      : Retour
//   - Ctrl + Q : Quitter
func RegisterShortcuts(w fyne.Window, onBack func(), onSearch func()) {
	// ESC : Retour à la liste
	w.Canvas().AddShortcut(&desktop.CustomShortcut{
		KeyName: fyne.KeyEscape,
	}, func(shortcut fyne.Shortcut) {
		onBack()
	})

	// Ctrl + Q : Quitter l'application
	w.Canvas().AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyQ,
		Modifier: fyne.KeyModifierControl,
	}, func(shortcut fyne.Shortcut) {
		w.Close()
	})

	// Ctrl + F : Focus sur la recherche
	w.Canvas().AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyF,
		Modifier: fyne.KeyModifierControl,
	}, func(shortcut fyne.Shortcut) {
		if onSearch != nil {
			onSearch()
		}
	})
}
