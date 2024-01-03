package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gabriel-vasile/mimetype"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("No directory or file provided.")
	}

	path := os.Args[1]

	if path == "" {
		log.Fatal("No directory or file provided.")
	}

	absolutePath, err := filepath.Abs(path)
	if err != nil {
		log.Fatal(err)
	}

	fileInfo, err := os.Stat(absolutePath)
	if err != nil {
		log.Fatal(err)
	}

	ffui := initialModel(fileInfo, absolutePath)

	p := tea.NewProgram(ffui)

	ffui.Program = p

	f, err := tea.LogToFile("ffui.log", "ffui")
	defer f.Close()

	if err != nil {
		log.Fatal(err)
	}

	err = p.Start()

	if err != nil {
		log.Fatal(err)
	}
}

func encode(fullFilePath string, fileName string, teaP *tea.Program, cfg ParsedConfig) {
	mType, err := mimetype.DetectFile(fullFilePath)
	if err != nil {
		log.Fatal(err)
	}

	if !strings.HasPrefix(mType.String(), "video/") {
		log.Fatalf("%s is not a valid video file\n", fileName)
	}

	parentDir := filepath.Dir(fullFilePath)
	extensionIndex := strings.LastIndex(fileName, ".")
	newFileName := fileName[:extensionIndex]
	extension := fileName[extensionIndex:]
	newFileFullPath := filepath.Join(parentDir, newFileName+fmt.Sprintf(" [%s] [%s]", cfg.VideoEncoder, cfg.AudioEncoder)+extension)

	if _, err := os.Stat(newFileFullPath); err == nil {
		if cfg.SkipEncodedVid {
			log.Printf("%s already exists with the exact same encodings (crf and preset might be different though), skipping.", newFileFullPath)
			teaP.Send(finishedEncodingVideo{})
			return
		} else {
			os.Remove(newFileFullPath)
		}
	}

	quotedFilePath := fmt.Sprintf("\"%s\"", fullFilePath)
	quotedNewFilePath := fmt.Sprintf("\"%s\"", newFileFullPath)

	err = ffmpeg.Input(quotedFilePath).
		Output(quotedNewFilePath, ffmpeg.KwArgs{
			"c:v":    cfg.VideoEncoder,
			"crf":    cfg.CRF,
			"preset": cfg.Preset,
			"c:a":    cfg.AudioEncoder,
			"b:a":    "128k"}).
		GlobalArgs("-progress", "unix://"+getProgressSocket(fullFilePath, teaP)).
		Run()

	if err != nil {
		teaP.Send(errQuitMsg{msg: fmt.Sprintf("FFmpeg quit with error: %s", err)})
	}
}
