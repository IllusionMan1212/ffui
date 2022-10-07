package main

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gabriel-vasile/mimetype"
)

type finishedEncodingVideo struct{}
type updateProgress struct {
	progress float64
}

type initUi struct {
	fileCount int
	files     []string
}

func (m *model) statFile() tea.Msg {
	if m.IsDirectory {
		entries, _ := os.ReadDir(m.Path)

		sort.Sort(DirEntrySlice(entries))

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

func encodeVideo() tea.Msg {
	return encodeVideoMsg{}
}

type quitMsg struct{}

func (m model) gracefullyQuit() tea.Msg {
	return quitMsg{}
}

type initCfgMsg struct{}

func initCfg() tea.Msg {
	return initCfgMsg{}
}
