package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Screen int

var Checkmark = lipgloss.NewStyle().Foreground(lipgloss.Color("#22FF33")).Render("✔")
var X = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF2233")).Render("✖️")

const (
	None Screen = iota
	Cfg
	Files
	Main
)

type File struct {
	Path     string
	Selected bool
}

type Model struct {
	IsDirectory           bool
	Path                  string
	CurrentFileName       string
	ViewportFocused       bool
	FileCount             int
	Files                 []File
	Viewport              viewport.Model
	Spinner               spinner.Model
	SingleFileProgressBar progress.Model
	SingleFileProgress    float64
	TotalProgressBar      progress.Model
	TotalProgress         float64
	Estimate              int
	Program               *tea.Program
	Command               *exec.Cmd
	Quitting              bool
	Cancelled             bool
	Screen                Screen
	Config                []Config
	VisibleConfig         []Config
	FocusIndex            int
	ChoiceIndex           int
	ParsedConfig          ParsedConfig
	DryRun                bool
	ErrQuit               bool
	ErrQuitMessage        string
}

// We're returning a pointer here so we can embed the tea.Program on the original model
// instead of a copy.
func initialModel(fileInfo os.FileInfo, absolutePath string) *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	codecsStr, err := exec.Command("ffmpeg", "-encoders").Output()
	if err != nil {
		log.Println(err)
	}

	codecs := strings.Split(strings.Split(string(codecsStr), " ------\n")[1], "\n")

	for i, codec := range codecs {
		if codec == "" {
			continue
		}
		codecs[i] = strings.Fields(codec)[1]

		// hardcoded indexes are never bad YEP :)))
		if contains(SupportedVideoEncoders, codecs[i]) {
			Configs[2].Opts = append(Configs[2].Opts, codecs[i])
		} else if contains(SupportedAudioEncoders, codecs[i]) {
			Configs[3].Opts = append(Configs[3].Opts, codecs[i])
		}
	}

	return &Model{
		IsDirectory:           fileInfo.IsDir(),
		CurrentFileName:       fileInfo.Name(),
		Path:                  absolutePath,
		FileCount:             0,
		Files:                 make([]File, 0),
		Viewport:              viewport.New(0, 0),
		Screen:                Cfg,
		Spinner:               s,
		SingleFileProgressBar: progress.New(progress.WithGradient("#1010ff", "#00ff00")),
		SingleFileProgress:    0.0,
		TotalProgressBar:      progress.New(progress.WithDefaultGradient()),
		TotalProgress:         0.0,
		Quitting:              false,
		Config:                Configs,
		VisibleConfig:         getVisibleConfigs(Configs),
		ErrQuit:               false,
		ErrQuitMessage:        "",
	}
}

func (m Model) updateConfigFocusedOptions() {
	for _, visCfg := range m.VisibleConfig {
		for i := range m.Config {
			if m.Config[i].Name == visCfg.Name {
				m.Config[i].FocusedOption = visCfg.FocusedOption
			}
		}
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.Spinner.Tick, m.statFiles)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case errQuitMsg:
		m.ErrQuitMessage = msg.msg
		m.ErrQuit = true
		return m, tea.Sequence(tea.ExitAltScreen, m.cleanUp)
	case filesStatMsg:
		m.FileCount = msg.fileCount
		m.Files = msg.files

		return m, nil
	case tea.WindowSizeMsg:
		m.Viewport.Width = msg.Width
		m.Viewport.Height = msg.Height - lipgloss.Height(FilesScreenViewHeader(m))
	}

	switch m.Screen {
	case Files:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			key := msg.String()
			switch key {
			case "ctrl+c", "esc":
				return m, tea.Quit
			case "enter", " ":
				if !m.ViewportFocused && key == "enter" {
					selectAll := !every(m.Files, func(e File) bool { return e.Selected })

					if m.ChoiceIndex == 0 {
						for i := range m.Files {
							m.Files[i].Selected = selectAll
						}
					} else {
						m.Files = filter(m.Files, func(f File) bool {
							return f.Selected
						})

						m.Screen = Main
						m.FileCount = len(m.Files)

						return m, tea.Batch(m.Spinner.Tick, encodeVideo)
					}
				} else if m.ViewportFocused {
					m.Files[m.FocusIndex].Selected = !m.Files[m.FocusIndex].Selected

					// Reset the choice to "Select All" button when no files are selected
					if !anyOf(m.Files, func(f File) bool { return f.Selected }) {
						m.ChoiceIndex = 0
					}
				}

				m.SetViewportContent()
			case "g":
				if m.ViewportFocused {
					m.FocusIndex = 0
					m.SetViewportContent()
					m.Viewport.GotoTop()
				}
			case "G":
				if m.ViewportFocused {
					m.FocusIndex = len(m.Files) - 1
					m.SetViewportContent()
					m.Viewport.GotoBottom()
				}
			case "left", "right", "h", "l":
				if !m.ViewportFocused && anyOf(m.Files, func(f File) bool { return f.Selected }) {
					if key == "right" || key == "l" {
						m.ChoiceIndex++
					} else {
						m.ChoiceIndex--
					}

					if m.ChoiceIndex > 1 {
						m.ChoiceIndex = 0
					} else if m.ChoiceIndex < 0 {
						m.ChoiceIndex = 1
					}
				}
			case "tab", "shift+tab":
				m.ViewportFocused = !m.ViewportFocused
				m.SetViewportContent()
			case "up", "down", "j", "k":
				if m.ViewportFocused {
					if key == "up" || key == "k" {
						m.FocusIndex--
					} else {
						m.FocusIndex++
					}

					if (key == "down" || key == "j") && m.FocusIndex >= m.Viewport.Height+m.Viewport.YOffset {
						m.Viewport.LineDown(1)
					}

					if (key == "up" || key == "k") && m.FocusIndex < m.Viewport.YOffset {
						m.Viewport.LineUp(1)
					}

					if m.FocusIndex >= len(m.Files) {
						m.Viewport.GotoTop()
						m.FocusIndex = 0
					} else if m.FocusIndex < 0 {
						m.Viewport.GotoBottom()
						m.FocusIndex = len(m.Files) - 1
					}

					m.SetViewportContent()
				}
			}
		}
	case Cfg:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			key := msg.String()
			switch key {
			case "ctrl+c", "esc":
				return m, tea.Quit
			case "enter", " ":
				// parse config and switch to main screen if we're focused on the start button
				if m.FocusIndex == len(m.VisibleConfig) {
					if m.ChoiceIndex == 0 {
						return m, m.parseConfig(false)
					} else if m.ChoiceIndex == 1 {
						return m, m.parseConfig(true)
					}
				}
			case "g":
				m.FocusIndex = 0
			case "G":
				m.FocusIndex = len(m.VisibleConfig)
			case "tab", "shift+tab", "up", "down", "j", "k":
				if key == "up" || key == "shift+tab" || key == "k" {
					m.FocusIndex--
				} else {
					m.FocusIndex++
				}

				if m.FocusIndex > len(m.VisibleConfig) {
					m.FocusIndex = 0
				} else if m.FocusIndex < 0 {
					m.FocusIndex = len(m.VisibleConfig)
				}
			case "left", "right", "h", "l":
				// If we're not hovering a button
				if m.FocusIndex < len(m.VisibleConfig) {
					cfg := &m.VisibleConfig[m.FocusIndex]
					if key == "right" || key == "l" {
						cfg.FocusedOption++
					} else {
						cfg.FocusedOption--
					}

					if cfg.FocusedOption >= len(cfg.Opts) {
						cfg.FocusedOption = 0
					} else if cfg.FocusedOption < 0 {
						cfg.FocusedOption = len(cfg.Opts) - 1
					}

					m.updateConfigFocusedOptions()
					m.VisibleConfig = getVisibleConfigs(m.Config)
				} else {
					if key == "right" || key == "l" {
						m.ChoiceIndex++
					} else {
						m.ChoiceIndex--
					}

					if m.ChoiceIndex > 1 {
						m.ChoiceIndex = 0
					} else if m.ChoiceIndex < 0 {
						m.ChoiceIndex = 1
					}
				}
			}
		case quitMsg:
			m.Quitting = true
			return m, tea.Quit
		case parsedCfgMsg:
			m.ParsedConfig = msg.parsedConfig

			if m.DryRun {
				return m, tea.Batch(tea.ExitAltScreen, tea.Quit)
			}

			if m.IsDirectory {
				m.Screen = Files
				m.FocusIndex = 0
				m.ChoiceIndex = 0

				m.SetViewportContent()
			} else {
				m.Screen = Main

				return m, tea.Batch(m.Spinner.Tick, encodeVideo)
			}
		}
	case Main:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			key := msg.String()
			switch key {
			case "ctrl+c":
				m.Cancelled = true
				err := m.Command.Process.Signal(os.Interrupt)
				if err != nil {
					log.Println("An error occurred when sending SIGINT to the ffmpeg process:")
					log.Println(err)
				}
				return m, tea.Sequence(tea.ExitAltScreen, tea.Quit)
			}
		case encodeVideoMsg:
			m.CurrentFileName = filepath.Base(m.Files[len(m.Files)-1].Path)

			go func() {
				file := m.Files[len(m.Files)-1]
				encode(file, filepath.Base(file.Path), m.Program, m.ParsedConfig)
			}()

			return m, nil
		case ffmpegProcessStart:
			log.Printf("Running command: %s\n", msg.cmd.String())
			m.Command = msg.cmd
			return m, nil
		case finishedEncodingVideo:
			if m.ParsedConfig.DeleteOldVideo {
				m.CurrentFileName = fmt.Sprintf("Deleting: %s", filepath.Base(m.Files[len(m.Files)-1].Path))
				os.Remove(m.Files[len(m.Files)-1].Path)
			}

			m.Files = m.Files[:len(m.Files)-1]

			if len(m.Files) == 0 {
				return m, tea.Sequence(tea.ExitAltScreen, gracefullyQuit)
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

func (m *Model) SetViewportContent() {
	var files string

	for i, file := range m.Files {
		selection := " "
		if file.Selected {
			selection = "x"
		}
		if m.ViewportFocused && m.FocusIndex == i {
			files += fmt.Sprintf(FocusedConfig.UnsetMarginTop().Render("[%s] %s"), selection, filepath.Base(file.Path))
		} else {
			files += fmt.Sprintf(BlurredConfig.UnsetMarginTop().Render("[%s] %s"), selection, filepath.Base(file.Path))
		}

		files += "\n"
	}

	m.Viewport.SetContent(files)
}

func parseConfig(cfg []Config) ParsedConfig {
	vEncoder := find(cfg, "Video Encoder")
	aEncoder := find(cfg, "Audio Encoder")
	preset := find(cfg, "Preset")
	crf := find(cfg, "Constant Rate Factor (CRF)")

	return ParsedConfig{
		DeleteOldVideo:        find(cfg, "Delete old video(s)?").FocusedOption != 0,
		IgnoreConflictingName: find(cfg, "On name conflict?").FocusedOption == 0,
		VideoEncoder:          vEncoder.Opts[vEncoder.FocusedOption],
		AudioEncoder:          aEncoder.Opts[aEncoder.FocusedOption],
		Preset:                preset.Opts[preset.FocusedOption],
		CRF:                   crf.Opts[crf.FocusedOption],
	}
}

func FilesScreenViewHeader(m Model) string {
	view := lipgloss.NewStyle().Margin(1, 0).Render("Select the files you wish to encode.")
	var buttons string
	var selectAllBtnText string

	allSelected := every(m.Files, func(e File) bool { return e.Selected })
	noneSelected := !anyOf(m.Files, func(e File) bool { return e.Selected })

	if allSelected {
		selectAllBtnText = "Deselect All"
	} else {
		selectAllBtnText = "Select All"
	}

	blurredStartButton := BlurredStartButton
	focusedStartButton := FocusedStartButton

	if noneSelected {
		blurredStartButton = DisabledStartButton
		focusedStartButton = DisabledStartButton
	}

	if !m.ViewportFocused {
		if m.ChoiceIndex == 0 {
			buttons = lipgloss.JoinHorizontal(0, FocusedSelectAllButton.Render(selectAllBtnText), blurredStartButton)
		} else {
			buttons = lipgloss.JoinHorizontal(0, BlurredSelectAllButton.Render(selectAllBtnText), focusedStartButton)
		}
	} else {
		buttons = lipgloss.JoinHorizontal(0, BlurredSelectAllButton.Render(selectAllBtnText), blurredStartButton)
	}

	view += buttons

	return view
}

func FilesScreenView(m Model) string {
	// TODO: files should be a grid (?)

	return lipgloss.JoinVertical(0, FilesScreenViewHeader(m), m.Viewport.View())
}

func CfgScreenView(m Model) string {
	view := ""

	for i, cfg := range m.VisibleConfig {
		opts := ""

		for j, opt := range cfg.Opts {
			if m.VisibleConfig[i].FocusedOption == j {
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

	var startButton string
	var dryRunButton string

	if m.FocusIndex == len(m.VisibleConfig) && m.ChoiceIndex == 0 {
		if m.IsDirectory {
			startButton = FocusedNextButton
		} else {
			startButton = FocusedStartButton
		}
	} else {
		if m.IsDirectory {
			startButton = BlurredNextButton
		} else {
			startButton = BlurredStartButton
		}
	}

	if m.FocusIndex == len(m.VisibleConfig) && m.ChoiceIndex == 1 {
		dryRunButton = FocusedDryRunButton
	} else {
		dryRunButton = BlurredDryRunButton
	}

	view += lipgloss.JoinHorizontal(0, startButton, dryRunButton)
	view += "\n"

	return view
}

func MainScreenView(m Model) string {
	progress := ""

	if m.IsDirectory && m.FileCount > 1 {
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

func (m Model) View() string {
	if m.DryRun {
		return ""
	}

	if m.Quitting {
		return fmt.Sprintf("%s %d/%d files encoded\n", Checkmark, m.FileCount-len(m.Files), m.FileCount)
	} else if m.Cancelled {
		// TODO: should we clean up the file ourselves?
		return fmt.Sprintf("%s Encoding cancelled. Stopped ffmpeg process.\n   Make sure to clean up the created file\n", X)
	}

	if m.ErrQuit {
		return fmt.Sprintf("%s %s\n", X, m.ErrQuitMessage)
	}

	switch m.Screen {
	case Files:
		return FilesScreenView(m)
	case Cfg:
		return CfgScreenView(m)
	case Main:
		return MainScreenView(m)
	}

	return "Error: Invalid Screen"
}
