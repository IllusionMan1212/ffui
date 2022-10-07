package main

import "github.com/charmbracelet/lipgloss"

var (
	PrimaryColor = lipgloss.AdaptiveColor{
		Dark:  "#EEE",
		Light: "#151515",
	}

	SecondaryColor = lipgloss.AdaptiveColor{
		Dark:  "#CE53BC",
		Light: "#CE53BC",
	}

	AccentColor = lipgloss.AdaptiveColor{
		Dark:  "#1C6DD0",
		Light: "#1C6DD0",
	}

	BlurredConfig = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			MarginTop(1)

	FocusedConfig = lipgloss.NewStyle().
			Foreground(AccentColor).
			MarginTop(1)

	BlurredOption = lipgloss.NewStyle().
			Foreground(PrimaryColor)

	FocusedOption = lipgloss.NewStyle().
			Foreground(SecondaryColor)
)
