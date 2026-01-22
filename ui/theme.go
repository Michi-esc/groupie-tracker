package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type CustomTheme struct{}

var _ fyne.Theme = (*CustomTheme)(nil)

func (m CustomTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return BgDark
	case theme.ColorNameButton:
		return AccentCyan
	case theme.ColorNameDisabledButton:
		return color.RGBA{R: 100, G: 100, B: 100, A: 255}
	case theme.ColorNameForeground:
		return TextWhite
	case theme.ColorNameHover:
		return AccentPink
	case theme.ColorNameInputBackground:
		return color.RGBA{R: 40, G: 40, B: 80, A: 255}
	case theme.ColorNamePrimary:
		return AccentCyan
	case theme.ColorNameFocus:
		return AccentCyan
	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

func (m CustomTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m CustomTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (m CustomTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
