package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
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

type ffmpegProcessStart struct {
	cmd *exec.Cmd
}

type filesStatMsg struct {
	fileCount int
	files     []string
}

func (m *Model) statFiles() tea.Msg {
	if m.IsDirectory {
		entries, _ := os.ReadDir(m.Path)

		sort.Sort(DirEntrySlice(entries))

		for _, entry := range entries {
			if entry.IsDir() || !entry.Type().IsRegular() {
				continue
			}

			fullFilePath := filepath.Join(m.Path, entry.Name())

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

		if len(m.Files) == 0 {
			return errQuitMsg{"Chosen directory has no video files"}
		}
	} else {
		m.FileCount = 1
		m.Files = append(m.Files, m.Path)
	}

	return filesStatMsg{
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

type parsedCfgMsg struct {
	parsedConfig ParsedConfig
	dryRun       bool
}

func (m *Model) parseConfig(dryRun bool) tea.Cmd {
	m.DryRun = dryRun

	return func() tea.Msg {
		vEncoder := find(m.Config, "Video Encoder")
		aEncoder := find(m.Config, "Audio Encoder")
		preset := find(m.Config, "Preset")
		crf := find(m.Config, "Constant Rate Factor (CRF)")

		return parsedCfgMsg{
			parsedConfig: ParsedConfig{
				DeleteOldVideo:        find(m.Config, "Delete old video(s)?").FocusedOption != 0,
				IgnoreConflictingName: find(m.Config, "On name conflict?").FocusedOption == 0,
				VideoEncoder:          vEncoder.Opts[vEncoder.FocusedOption],
				AudioEncoder:          aEncoder.Opts[aEncoder.FocusedOption],
				Preset:                preset.Opts[preset.FocusedOption],
				CRF:                   crf.Opts[crf.FocusedOption],
			},
			dryRun: dryRun,
		}
	}
}

type errQuitMsg struct {
	msg string
}

func (m *Model) cleanUp() tea.Msg {
	if len(m.Files) == 0 {
		return tea.Quit()
	}

	fullFilePath := m.Files[len(m.Files)-1]
	fileName := filepath.Base(fullFilePath)

	parentDir := filepath.Dir(fullFilePath)
	extensionIndex := strings.LastIndex(fileName, ".")
	newFileName := fileName[:extensionIndex]
	extension := fileName[extensionIndex:]
	newFileFullPath := filepath.Join(parentDir, newFileName+fmt.Sprintf("_[%s]_[%s]", m.ParsedConfig.VideoEncoder, m.ParsedConfig.AudioEncoder)+extension)

	os.Remove(newFileFullPath)

	return tea.Quit()
}
