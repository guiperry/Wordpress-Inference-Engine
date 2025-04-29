// In ui/theme.go
package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// HighContrastTheme defines a custom high-contrast theme.
type HighContrastTheme struct{}

// Ensure HighContrastTheme implements fyne.Theme
var _ fyne.Theme = (*HighContrastTheme)(nil)

// Color returns the specified color for the theme.
func (t *HighContrastTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		// Dark background
		return color.NRGBA{R: 0x1e, G: 0x1e, B: 0x1e, A: 0xff} // Dark Grey
	case theme.ColorNameForeground:
		// White text
		return color.White
	case theme.ColorNameButton:
		// Darker Purple for standard buttons
		return color.NRGBA{R: 0x6a, G: 0x1b, B: 0x9a, A: 0xff}
	case theme.ColorNamePrimary:
		// Brighter Purple for important buttons/accents
		return color.NRGBA{R: 0x8a, G: 0x2b, B: 0xe2, A: 0xff} // BlueViolet
	case theme.ColorNameHover:
		// Slightly lighter purple for hover
		return color.NRGBA{R: 0x7b, G: 0x1f, B: 0xae, A: 0xff}
	case theme.ColorNamePlaceHolder:
		// Slightly dimmer white for placeholder text
		return color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff}
	case theme.ColorNameScrollBar:
		// Make scrollbar slightly visible
		return color.NRGBA{R: 0x44, G: 0x44, B: 0x44, A: 0xff}
	case theme.ColorNameShadow:
		// Darker shadow for contrast
		return color.NRGBA{R: 0x0, G: 0x0, B: 0x0, A: 0x66}
	default:
		// Fallback to the standard dark theme for other colors
		return theme.DarkTheme().Color(name, variant)
	}
}

// Font returns the specified font for the theme.
func (t *HighContrastTheme) Font(style fyne.TextStyle) fyne.Resource {
	// Use standard dark theme fonts
	return theme.DarkTheme().Font(style)
}

// Icon returns the specified icon for the theme.
func (t *HighContrastTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	// Use standard dark theme icons
	return theme.DarkTheme().Icon(name)
}

// Size returns the specified size for the theme.
func (t *HighContrastTheme) Size(name fyne.ThemeSizeName) float32 {
	// Use standard dark theme sizes
	return theme.DarkTheme().Size(name)
}
