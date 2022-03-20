package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gabriel-vasile/mimetype"
)

var checkmark = lipgloss.NewStyle().Foreground(lipgloss.Color("#22FF33")).Render("✔")

type model struct {
	IsDirectory     bool
	Path            string
	CurrentFileName string
	FileCount       int
	Files           []string
	Spinner         spinner.Model
	Progress        float64
	Program         *tea.Program
	Quitting        bool
}

func initialModel(fileInfo os.FileInfo, absolutePath string) *model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &model{
		IsDirectory:     fileInfo.IsDir(),
		CurrentFileName: fileInfo.Name(),
		Path:            absolutePath,
		FileCount:       0,
		Files:           make([]string, 0),
		Spinner:         s,
		Progress:        0.0,
		Quitting:        false,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.Spinner.Tick, m.statFile)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
	case initUi:
		m.FileCount = msg.fileCount
		m.Files = msg.files

		return m, encodeVideo
	case encodeVideoMsg:
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
		m.Progress = msg.progress
		return m, nil
	case quitMsg:
		m.Quitting = true
		return m, tea.Quit
	}

	m.Spinner, cmd = m.Spinner.Update(msg)

	return m, cmd
}

func (m model) View() string {
	if m.Quitting {
		return fmt.Sprintf("%s %d/%d files encoded\n", checkmark, m.FileCount-len(m.Files), m.FileCount)
	}

	return fmt.Sprintf("%s %d/%d files encoded\n%.0f%% done", m.Spinner.View(), m.FileCount-len(m.Files), m.FileCount, m.Progress)
}

type initUi struct {
	fileCount int
	files     []string
}

func (m *model) statFile() tea.Msg {
	if m.IsDirectory {
		entries, _ := os.ReadDir(m.Path)
		for _, entry := range entries {
			fullFilePath := filepath.Join(m.Path, entry.Name())

			if entry.IsDir() {
				continue
			}

			mType, err := mimetype.DetectFile(fullFilePath)
			if err != nil {
				log.Fatal(err)
			}

			if !strings.HasPrefix(mType.String(), "video/") {
				continue
			}

			m.FileCount++
			m.Files = append(m.Files, fullFilePath)
		}
	} else {
		m.FileCount = 1
		m.Files = append(m.Files, m.Path)
	}

	return initUi{
		fileCount: m.FileCount,
		files:     m.Files,
	}
}

type encodeVideoMsg struct{}
type finishedEncodingVideo struct{}
type updateProgress struct {
	progress float64
}
type quitMsg struct{}

func encodeVideo() tea.Msg {
	return encodeVideoMsg{}
}

func (m model) gracefullyQuit() tea.Msg {
	return quitMsg{}
}
