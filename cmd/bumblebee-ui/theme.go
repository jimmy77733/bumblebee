//go:build windows

package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

const prefLightTheme = "light_theme"

type bumbleTheme struct {
	light bool
}

func newBumbleTheme(light bool) bumbleTheme {
	return bumbleTheme{light: light}
}

func themeToggleLabel(light bool) string {
	if light {
		return "深色主題"
	}
	return "淺色主題"
}

func (t bumbleTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	if t.light {
		return t.lightColor(name)
	}
	return t.darkColor(name)
}

func (t bumbleTheme) lightColor(name fyne.ThemeColorName) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 0xfa, G: 0xf7, B: 0xf2, A: 0xff}
	case theme.ColorNameOverlayBackground:
		return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 0x1c, G: 0x19, B: 0x17, A: 0xff}
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 0xa8, G: 0xa2, B: 0x9e, A: 0xff}
	case theme.ColorNameButton:
		return color.NRGBA{R: 0xee, G: 0xea, B: 0xe2, A: 0xff}
	case theme.ColorNameDisabledButton:
		return color.NRGBA{R: 0xf5, G: 0xf0, B: 0xe8, A: 0xff}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 0x78, G: 0x71, B: 0x6c, A: 0xff}
	case theme.ColorNamePrimary, theme.ColorNameHyperlink:
		return color.NRGBA{R: 0xd9, G: 0x77, B: 0x06, A: 0xff}
	case theme.ColorNameForegroundOnPrimary:
		return color.NRGBA{R: 0xff, G: 0xfb, B: 0xf5, A: 0xff}
	case theme.ColorNameSuccess:
		return color.NRGBA{R: 0x16, G: 0x80, B: 0x3c, A: 0xff}
	case theme.ColorNameForegroundOnSuccess:
		return color.NRGBA{R: 0xf0, G: 0xfd, B: 0xf4, A: 0xff}
	case theme.ColorNameWarning:
		return color.NRGBA{R: 0xd9, G: 0x77, B: 0x06, A: 0xff}
	case theme.ColorNameForegroundOnWarning:
		return color.NRGBA{R: 0x1c, G: 0x19, B: 0x17, A: 0xff}
	case theme.ColorNameError:
		return color.NRGBA{R: 0xdc, G: 0x26, B: 0x26, A: 0xff}
	case theme.ColorNameForegroundOnError:
		return color.NRGBA{R: 0xff, G: 0xf1, B: 0xf2, A: 0xff}
	case theme.ColorNameFocus, theme.ColorNameSelection:
		return color.NRGBA{R: 0xd9, G: 0x77, B: 0x06, A: 0x55}
	case theme.ColorNameSeparator:
		return color.NRGBA{R: 0xe7, G: 0xe0, B: 0xd5, A: 0xff}
	case theme.ColorNameShadow:
		return color.NRGBA{R: 0x1c, G: 0x19, B: 0x17, A: 0x22}
	case theme.ColorNameHeaderBackground:
		return color.NRGBA{R: 0xf5, G: 0xf0, B: 0xe8, A: 0xff}
	}
	return theme.DefaultTheme().Color(name, theme.VariantLight)
}

func (t bumbleTheme) darkColor(name fyne.ThemeColorName) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 0x12, G: 0x10, B: 0x0c, A: 0xff}
	case theme.ColorNameOverlayBackground:
		return color.NRGBA{R: 0x1c, G: 0x19, B: 0x16, A: 0xff}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 0xf5, G: 0xf0, B: 0xe8, A: 0xff}
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 0x78, G: 0x71, B: 0x6c, A: 0xff}
	case theme.ColorNameButton:
		return color.NRGBA{R: 0x29, G: 0x25, B: 0x24, A: 0xff}
	case theme.ColorNameDisabledButton:
		return color.NRGBA{R: 0x1c, G: 0x19, B: 0x16, A: 0xff}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 0x1c, G: 0x19, B: 0x16, A: 0xff}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 0xa8, G: 0xa2, B: 0x9e, A: 0xff}
	case theme.ColorNamePrimary, theme.ColorNameHyperlink:
		return color.NRGBA{R: 0xf5, G: 0x9e, B: 0x0b, A: 0xff}
	case theme.ColorNameForegroundOnPrimary:
		return color.NRGBA{R: 0x1c, G: 0x19, B: 0x17, A: 0xff}
	case theme.ColorNameSuccess:
		return color.NRGBA{R: 0x16, G: 0xa3, B: 0x4a, A: 0xff}
	case theme.ColorNameForegroundOnSuccess:
		return color.NRGBA{R: 0xf0, G: 0xfd, B: 0xf4, A: 0xff}
	case theme.ColorNameWarning:
		return color.NRGBA{R: 0xd9, G: 0x77, B: 0x06, A: 0xff}
	case theme.ColorNameForegroundOnWarning:
		return color.NRGBA{R: 0x1c, G: 0x19, B: 0x17, A: 0xff}
	case theme.ColorNameError:
		return color.NRGBA{R: 0xdc, G: 0x26, B: 0x26, A: 0xff}
	case theme.ColorNameForegroundOnError:
		return color.NRGBA{R: 0xff, G: 0xf1, B: 0xf2, A: 0xff}
	case theme.ColorNameFocus, theme.ColorNameSelection:
		return color.NRGBA{R: 0xf5, G: 0x9e, B: 0x0b, A: 0x66}
	case theme.ColorNameSeparator:
		return color.NRGBA{R: 0x3f, G: 0x3a, B: 0x34, A: 0xff}
	case theme.ColorNameShadow:
		return color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x88}
	case theme.ColorNameHeaderBackground:
		return color.NRGBA{R: 0x16, G: 0x14, B: 0x11, A: 0xff}
	}
	return theme.DefaultTheme().Color(name, theme.VariantDark)
}

func (bumbleTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (bumbleTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (bumbleTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 10
	case theme.SizeNameInnerPadding:
		return 12
	case theme.SizeNameText:
		return 14
	case theme.SizeNameHeadingText:
		return 22
	}
	return theme.DefaultTheme().Size(name)
}
