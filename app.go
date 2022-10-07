package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Screen int
type FfuiBool int

var checkmark = lipgloss.NewStyle().Foreground(lipgloss.Color("#22FF33")).Render("✔")

var Configs = []Config{
	{Name: "Delete unencoded video(s)?", Opts: []string{"No", "Yes"}},
}

const (
	Cfg Screen = iota
	Main
)

const (
	False FfuiBool = iota
	True
)

type Config struct {
	Name          string
	Opts          []string
	FocusedOption int
}

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
	Program               *tea.Program
	Quitting              bool
	Screen                Screen
	Config                map[int]Config
	FocusIndex            int
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
	}

	switch m.Screen {
	case Cfg:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			key := msg.String()
			switch key {
			case "enter", " ":
				// switch to main screen if we're focused on the start button
				if m.FocusIndex == len(Configs) {
					m.Screen = Main
					return m, tea.Batch(m.Spinner.Tick, m.statFile)
				}
			case "tab", "shift+tab", "up", "down":
				if key == "up" || key == "shift+tab" {
					m.FocusIndex--
				} else {
					m.FocusIndex++
				}

				// TODO: make this len(Configs) + 1 after start button is added and change the else statement accordingly
				if m.FocusIndex >= len(Configs) {
					m.FocusIndex = 0
				} else if m.FocusIndex < 0 {
					m.FocusIndex = len(Configs) - 1
				}

				// TODO: move to start button
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
		}
	case Main:
		switch msg := msg.(type) {
		case initUi:
			m.FileCount = msg.fileCount
			m.Files = msg.files

			return m, encodeVideo
		case encodeVideoMsg:
			m.CurrentFileName = filepath.Base(m.Files[len(m.Files)-1])

			go func() {
				fullFilePath := m.Files[len(m.Files)-1]
				encode(fullFilePath, filepath.Base(fullFilePath), m.Program)
			}()

			return m, nil
		case finishedEncodingVideo:
			m.Files = m.Files[:len(m.Files)-1]

			if len(m.Files) == 0 {
				return m, m.gracefullyQuit
			}

			return m, encodeVideo
		case updateProgress:
			m.SingleFileProgress = msg.progress
			m.TotalProgress = (1.0/float64(m.FileCount))*msg.progress + (1.0 / float64(m.FileCount) * float64(m.FileCount-len(m.Files)))

			singleProgressCmd := m.SingleFileProgressBar.SetPercent(msg.progress)
			totalProgressCmd := m.TotalProgressBar.SetPercent(m.TotalProgress)

			return m, tea.Batch(singleProgressCmd, totalProgressCmd)
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

	// TODO: add start button

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

	view := fmt.Sprintf("\nEncoding \"%s\"...\n%s", m.CurrentFileName, progress)

	return view
}

func (m model) View() string {
	if m.Quitting {
		return fmt.Sprintf("%s %d/%d files encoded\n", checkmark, m.FileCount-len(m.Files), m.FileCount)
	}

	switch m.Screen {
	case Cfg:
		return CfgScreenView(m)
	case Main:
		return MainScreenView(m)
	}

	return "Error: Invalid Screen"
}
