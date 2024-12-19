package main

import (
	"fmt"
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
type updateEstimate struct {
	estimate int
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

func gracefullyQuit() tea.Msg {
	return quitMsg{}
}

type initCfgMsg struct{}

func initCfg() tea.Msg {
	return initCfgMsg{}
}

type parsedCfgMsg struct {
	parsedConfig ParsedConfig
	dryRun       bool
}

func (m *model) parseConfig(dryRun bool) tea.Cmd {
	m.DryRun = dryRun

	return func() tea.Msg {
		vEncoder := find(m.Config, "Video Encoder")
		aEncoder := find(m.Config, "Audio Encoder")
		preset := find(m.Config, "Preset")
		crf := find(m.Config, "Constant Rate Factor (CRF)")

		return parsedCfgMsg{
			parsedConfig: ParsedConfig{
				DeleteOldVideo: find(m.Config, "Delete old video(s)?").FocusedOption != 0,
				SkipEncodedVid: find(m.Config, "What should we do about encoded videos?").FocusedOption == 0,
				VideoEncoder:   vEncoder.Opts[vEncoder.FocusedOption],
				AudioEncoder:   aEncoder.Opts[aEncoder.FocusedOption],
				Preset:         preset.Opts[preset.FocusedOption],
				CRF:            crf.Opts[crf.FocusedOption],
			},
			dryRun: dryRun,
		}
	}
}

type errQuitMsg struct {
	msg string
}

func (m *model) cleanUp() tea.Msg {
	fullFilePath := m.Files[len(m.Files)-1]
	fileName := filepath.Base(fullFilePath)

	parentDir := filepath.Dir(fullFilePath)
	extensionIndex := strings.LastIndex(fileName, ".")
	newFileName := fileName[:extensionIndex]
	extension := fileName[extensionIndex:]
	newFileFullPath := filepath.Join(parentDir, newFileName+fmt.Sprintf(" [%s] [%s]", m.ParsedConfig.VideoEncoder, m.ParsedConfig.AudioEncoder)+extension)

	os.Remove(newFileFullPath)

	return tea.Quit()
}
