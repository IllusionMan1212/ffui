package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

type Screen int

var checkmark = lipgloss.NewStyle().Foreground(lipgloss.Color("#22FF33")).Render("✔")
var x = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF2233")).Render("✖️")

const (
	Cfg Screen = iota
	Main
)

type model struct {
	IsDirectory           bool
	Path                  string
	CurrentFileName       string
	FileCount             int
	Files                 []string
	Spinner               spinner.Model
	SingleFileProgressBar progress.Model
	SingleFileProgress    float64
	TotalProgressBar      progress.Model
	TotalProgress         float64
	Estimate              int
	Program               *tea.Program
	Quitting              bool
	Screen                Screen
	Config                map[int]Config
	FocusIndex            int
	ParsedConfig          ParsedConfig
	DryRun                bool
	ErrQuit               bool
	ErrQuitMessage        string
}

func initialModel(fileInfo os.FileInfo, absolutePath string) *model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	cfg := make(map[int]Config)

	for i := 0; i < len(Configs); i++ {
		cfg[i] = Configs[i]
	}

	return &model{
		IsDirectory:           fileInfo.IsDir(),
		CurrentFileName:       fileInfo.Name(),
		Path:                  absolutePath,
		FileCount:             0,
		Files:                 make([]string, 0),
		Spinner:               s,
		SingleFileProgressBar: progress.New(progress.WithGradient("#1010ff", "#00ff00")),
		SingleFileProgress:    0.0,
		TotalProgressBar:      progress.New(progress.WithDefaultGradient()),
		TotalProgress:         0.0,
		Quitting:              false,
		Config:                cfg,
		ErrQuit:               false,
		ErrQuitMessage:        "",
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.Spinner.Tick, initCfg)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
	case errQuitMsg:
		m.ErrQuitMessage = msg.msg
		m.ErrQuit = true
		return m, m.cleanUp
	}

	switch m.Screen {
	case Cfg:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			key := msg.String()
			switch key {
			case "enter", " ":
				// parse config and switch to main screen if we're focused on the start button
				if m.FocusIndex == len(Configs) {
					return m, m.parseConfig(false)
				}
				// parse config, print it and exit
				if m.FocusIndex == len(Configs)+1 {
					return m, m.parseConfig(true)
				}
			case "tab", "shift+tab", "up", "down":
				if key == "up" || key == "shift+tab" {
					m.FocusIndex--
				} else {
					m.FocusIndex++
				}

				if m.FocusIndex >= len(Configs)+2 {
					m.FocusIndex = 0
				} else if m.FocusIndex < 0 {
					m.FocusIndex = len(Configs) + 1
				}
			case "left", "right":
				if cfg, ok := m.Config[m.FocusIndex]; ok {
					if key == "right" {
						cfg.FocusedOption++
					} else {
						cfg.FocusedOption--
					}

					if cfg.FocusedOption >= len(cfg.Opts) {
						cfg.FocusedOption = 0
					} else if cfg.FocusedOption < 0 {
						cfg.FocusedOption = len(cfg.Opts) - 1
					}

					m.Config[m.FocusIndex] = cfg
				}
			}
		case quitMsg:
			m.Quitting = true
			return m, tea.Quit
		case parsedCfgMsg:
			m.Screen = Main
			m.ParsedConfig = msg.parsedConfig
			return m, tea.Batch(m.Spinner.Tick, m.statFile)
		}
	case Main:
		switch msg := msg.(type) {
		case initUi:
			m.FileCount = msg.fileCount
			m.Files = msg.files

			if m.DryRun {
				return m, gracefullyQuit
			}

			return m, encodeVideo
		case encodeVideoMsg:
			m.CurrentFileName = filepath.Base(m.Files[len(m.Files)-1])

			go func() {
				fullFilePath := m.Files[len(m.Files)-1]
				encode(fullFilePath, filepath.Base(fullFilePath), m.Program, m.ParsedConfig)
			}()

			return m, nil
		case finishedEncodingVideo:
			if m.ParsedConfig.DeleteOldVideo {
				m.CurrentFileName = fmt.Sprintf("Deleting: %s", filepath.Base(m.Files[len(m.Files)-1]))
				os.Remove(m.Files[len(m.Files)-1])
			}

			m.Files = m.Files[:len(m.Files)-1]

			if len(m.Files) == 0 {
				return m, gracefullyQuit
			}

			return m, encodeVideo
		case updateProgress:
			m.SingleFileProgress = msg.progress
			m.TotalProgress = (1.0/float64(m.FileCount))*msg.progress + (1.0 / float64(m.FileCount) * float64(m.FileCount-len(m.Files)))

			singleProgressCmd := m.SingleFileProgressBar.SetPercent(msg.progress)
			totalProgressCmd := m.TotalProgressBar.SetPercent(m.TotalProgress)

			return m, tea.Batch(singleProgressCmd, totalProgressCmd)
		case updateEstimate:
			m.Estimate = msg.estimate

			return m, tea.Batch()
		case quitMsg:
			m.Quitting = true
			return m, tea.Quit
		case progress.FrameMsg:
			singleProgressModel, singleProgressCmd := m.SingleFileProgressBar.Update(msg)
			m.SingleFileProgressBar = singleProgressModel.(progress.Model)

			totalProgressModel, totalProgressCmd := m.TotalProgressBar.Update(msg)
			m.TotalProgressBar = totalProgressModel.(progress.Model)

			return m, tea.Batch(singleProgressCmd, totalProgressCmd)
		}

		m.Spinner, cmd = m.Spinner.Update(msg)
	}

	return m, cmd
}

func CfgScreenView(m model) string {
	view := ""

	for i, cfg := range Configs {
		opts := ""

		for j, opt := range cfg.Opts {
			if m.Config[i].FocusedOption == j {
				opts += FocusedOption.Render(opt)
			} else {
				opts += BlurredOption.Render(opt)
			}

			if j != len(cfg.Opts)-1 {
				opts += ", "
			}
		}

		if i == m.FocusIndex {
			view += fmt.Sprintf("%s { %s }\n", FocusedConfig.Render(cfg.Name), opts)
		} else {
			view += fmt.Sprintf("%s { %s }\n", BlurredConfig.Render(cfg.Name), opts)
		}
	}

	if m.FocusIndex == len(Configs) {
		view += FocusedStartButton
	} else {
		view += BlurredStartButton
	}

	if m.FocusIndex == len(Configs)+1 {
		view += FocusedDryRunButton
	} else {
		view += BlurredDryRunButton
	}

	view += "\n"

	return view
}

func MainScreenView(m model) string {
	progress := ""

	if m.IsDirectory {
		progress = fmt.Sprintf("%s %d/%d files encoded\nFile Progress: %s\n\nTotal Progress: %s",
			m.Spinner.View(),
			m.FileCount-len(m.Files),
			m.FileCount,
			m.SingleFileProgressBar.View(),
			m.TotalProgressBar.View())
	} else {
		progress = fmt.Sprintf("%s %d/%d files encoded\n%s",
			m.Spinner.View(),
			m.FileCount-len(m.Files),
			m.FileCount,
			m.SingleFileProgressBar.View())
	}

	eta := formatEstimate(m.Estimate)

	view := fmt.Sprintf("\nEncoding \"%s\"... ETA: %s\n%s", m.CurrentFileName, eta, progress)

	return view
}

func formatEstimate(estimate int) string {
	if estimate >= 86400 {
		day := estimate / 86400
		hour := (estimate % 86400) / 3600
		min := (estimate % 3600) / 60
		sec := estimate % 60
		return fmt.Sprintf("%dd%dh%dm%ds", day, hour, min, sec)
	} else if estimate >= 3600 {
		hour := estimate / 3600
		min := (estimate % 3600) / 60
		sec := estimate % 60
		return fmt.Sprintf("%dh%dm%ds", hour, min, sec)
	} else if estimate >= 60 {
		min := estimate / 60
		sec := estimate % 60
		return fmt.Sprintf("%dm%ds", min, sec)
	}

	return fmt.Sprintf("%ds", estimate)
}

func (m model) View() string {
	if m.Quitting {
		if m.DryRun {
			fullFilePath := m.Files[len(m.Files)-1]
			fileName := filepath.Base(fullFilePath)
			extensionIndex := strings.LastIndex(fileName, ".")
			extension := fileName[extensionIndex:]
			newFullFilePath := "./test-out" + extension

			cli := ffmpeg.Input(fullFilePath).Output(newFullFilePath, ffmpeg.KwArgs{
				"c:v":    m.ParsedConfig.VideoEncoder,
				"crf":    m.ParsedConfig.CRF,
				"preset": m.ParsedConfig.Preset,
				"c:a":    m.ParsedConfig.AudioEncoder,
				"b:a":    "128k"}).
				Compile()

			return fmt.Sprintf("%s %s\n", checkmark, strings.Join(cli.Args, " "))
		} else {
			return fmt.Sprintf("%s %d/%d files encoded\n", checkmark, m.FileCount-len(m.Files), m.FileCount)
		}
	}

	if m.ErrQuit {
		return fmt.Sprintf("%s %s\n", x, m.ErrQuitMessage)
	}

	switch m.Screen {
	case Cfg:
		return CfgScreenView(m)
	case Main:
		return MainScreenView(m)
	}

	return "Error: Invalid Screen"
}
