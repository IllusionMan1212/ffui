package main

import (
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

func encode(fullFilePath string, fileName string, teaP *tea.Program) {
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
	newFileFullPath := filepath.Join(parentDir, newFileName+" [x265]"+extension)

	err = ffmpeg.Input(fullFilePath).
		Output(newFileFullPath, ffmpeg.KwArgs{
			"c:v":    "libx265",
			"crf":    "30",
			"preset": "fast",
			"c:a":    "aac",
			"b:a":    "128k"}).
		GlobalArgs("-progress", "unix://"+getProgressSocket(fullFilePath, teaP)).
		Run()
}
