package ui

import "image/color"

// palette couleurs
var (
	BgDark      = color.RGBA{R: 15, G: 12, B: 41, A: 255}
	BgDarker    = color.RGBA{R: 10, G: 8, B: 30, A: 255}
	AccentCyan  = color.RGBA{R: 0, G: 212, B: 255, A: 255}
	AccentPink  = color.RGBA{R: 255, G: 0, B: 110, A: 255}
	TextWhite   = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	TextLight   = color.RGBA{R: 200, G: 200, B: 200, A: 255}
	CardBg      = color.RGBA{R: 25, G: 22, B: 60, A: 255}
	CardBgLight = color.RGBA{R: 35, G: 32, B: 70, A: 255}
	TextBlack   = color.RGBA{R: 0, G: 0, B: 0, A: 255}
)

// ContrastColor returns black or white depending on the perceived
// luminance of the provided color to ensure readable text.
func ContrastColor(c color.Color) color.Color {
	r, g, b, _ := c.RGBA()
	// r,g,b are in 0..65535 range
	rf := float64(r) / 65535.0
	gf := float64(g) / 65535.0
	bf := float64(b) / 65535.0
	// standard luminance formula
	lum := 0.299*rf + 0.587*gf + 0.114*bf
	if lum > 0.5 {
		return TextBlack
	}
	return TextWhite
}
