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
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Screen int

var Checkmark = lipgloss.NewStyle().Foreground(lipgloss.Color("#22FF33")).Render("✔")
var X = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF2233")).Render("✖️")

const (
	None Screen = iota
	Files
	Cfg
	Main
)

type File struct {
	Name   string
	Encode bool
}

type Model struct {
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
		Files:                 make([]string, 0),
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

		if len(m.Files) == 1 {
			m.Screen = Cfg
		} else if len(m.Files) > 1 {
			m.Screen = Files
		}

		return m, nil
	}

	switch m.Screen {
	case Files:
		// TODO: navigate files and select them and stuff
		switch msg := msg.(type) {
		case tea.KeyMsg:
			key := msg.String()
			switch key {
			case "ctrl+c", "esc":
				return m, tea.Quit
			case "enter", "space":
				// TODO: change the file state to Encode = true
				// Also change the files to be of File type
				// t := m.Files[m.FocusIndex]
			case "g":
				m.FocusIndex = 0
			case "G":
				m.FocusIndex = len(m.Files) - 1
			case "tab", "shift+tab", "up", "down", "j", "k":
				if key == "up" || key == "shift+tab" || key == "k" {
					m.FocusIndex--
				} else {
					m.FocusIndex++
				}

				if m.FocusIndex >= len(m.Files) {
					m.FocusIndex = 0
				} else if m.FocusIndex < 0 {
					m.FocusIndex = len(m.Files) - 1
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
			m.Screen = Main
			m.ParsedConfig = msg.parsedConfig
			if m.DryRun {
				return m, tea.Batch(tea.ExitAltScreen, tea.Quit)
			}

			return m, tea.Batch(m.Spinner.Tick, encodeVideo)
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
			m.CurrentFileName = filepath.Base(m.Files[len(m.Files)-1])

			go func() {
				fullFilePath := m.Files[len(m.Files)-1]
				encode(fullFilePath, filepath.Base(fullFilePath), m.Program, m.ParsedConfig)
			}()

			return m, nil
		case ffmpegProcessStart:
			log.Printf("Running command: %s\n", msg.cmd.String())
			m.Command = msg.cmd
			return m, nil
		case finishedEncodingVideo:
			if m.ParsedConfig.DeleteOldVideo {
				m.CurrentFileName = fmt.Sprintf("Deleting: %s", filepath.Base(m.Files[len(m.Files)-1]))
				os.Remove(m.Files[len(m.Files)-1])
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

func FilesScreenView(m Model) string {
	view := ""

	// TODO: Create a nice view where we can select different videos and then select the encoding options and start
	// encoding.
	for i, file := range m.Files {
		if m.FocusIndex == i {
			view += fmt.Sprintf(FocusedConfig.Render("[%s] %s"), " ", filepath.Base(file))
		} else {
			view += fmt.Sprintf(BlurredConfig.Render("[%s] %s"), " ", filepath.Base(file))
		}
	}

	view += "\n"

	return view
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
		startButton = FocusedStartButton
	} else {
		startButton = BlurredStartButton
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
