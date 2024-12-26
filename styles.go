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

	DisabledColor = lipgloss.AdaptiveColor{
		Dark:  "#444",
		Light: "#AAA",
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
	DisabledStartButton = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(DisabledColor).
				Foreground(DisabledColor).
				MarginTop(1).
				Padding(0, 2).
				Align(lipgloss.Center).
				Render("Start Encoding!")

	FocusedNextButton = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(AccentColor).
				Foreground(AccentColor).
				MarginTop(1).
				Padding(0, 2).
				Align(lipgloss.Center).
				Bold(true).
				Render("Next")
	BlurredNextButton = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(PrimaryColor).
				Foreground(PrimaryColor).
				MarginTop(1).
				Padding(0, 2).
				Align(lipgloss.Center).
				Render("Next")

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

	FocusedSelectAllButton = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(AccentColor).
				Foreground(AccentColor).
				MarginTop(1).
				Padding(0, 2).
				Align(lipgloss.Center).
				Bold(true)
	BlurredSelectAllButton = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(PrimaryColor).
				Foreground(PrimaryColor).
				MarginTop(1).
				Padding(0, 2).
				Align(lipgloss.Center).
				Bold(true)
)
