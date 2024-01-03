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

	FocusedStartButton = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(AccentColor).
				Foreground(AccentColor).
				MarginTop(1).
				Padding(0, 2).
				Align(lipgloss.Center).
				Bold(true).
				Render("Start Encoding!")
	BlurredStartButton = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(PrimaryColor).
				Foreground(PrimaryColor).
				MarginTop(1).
				Padding(0, 2).
				Align(lipgloss.Center).
				Render("Start Encoding!")

	FocusedDryRunButton = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(AccentColor).
				Foreground(AccentColor).
				MarginTop(1).
				Padding(0, 2).
				Align(lipgloss.Center).
				Bold(true).
				Render("Print FFmpeg command and exit")
	BlurredDryRunButton = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(PrimaryColor).
				Foreground(PrimaryColor).
				MarginTop(1).
				Padding(0, 2).
				Align(lipgloss.Center).
				Render("Print FFmpeg command and exit")
)
