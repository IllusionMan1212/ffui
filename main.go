package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gabriel-vasile/mimetype"
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

	p := tea.NewProgram(ffui, tea.WithAltScreen())

	ffui.Program = p

	f, err := tea.LogToFile("ffui.log", "ffui")
	defer f.Close()

	if err != nil {
		log.Fatal(err)
	}

	final, err := p.Run()

	if err != nil {
		fmt.Println(err)
		log.Fatal(err)
	}

	finalModel, _ := final.(Model)

	if finalModel.DryRun {
		for _, file := range finalModel.Files {
			parentDir := filepath.Dir(file)
			fileName := filepath.Base(file)
			extensionIndex := strings.LastIndex(fileName, ".")
			newFileName := fileName[:extensionIndex]
			extension := fileName[extensionIndex:]
			outFileFullPath := filepath.Join(parentDir, newFileName+fmt.Sprintf("_[%s]_[%s]", finalModel.ParsedConfig.VideoEncoder, finalModel.ParsedConfig.AudioEncoder)+extension)

			cmd := exec.Command("ffmpeg", buildFFmpegCmdArgs(file, outFileFullPath, finalModel.ParsedConfig)...)

			fmt.Println(fmt.Sprintf("%s %s", Checkmark, cmd.String()))
		}
	}
}

func buildFFmpegCmdArgs(fullFilePath string, outFullFilePath string, cfg ParsedConfig, additionalArgs ...string) []string {
	args := make([]string, 0, 10)
	// Input file
	args = append(args, "-i")
	args = append(args, fullFilePath)

	// Encoding parameters
	args = append(args, "-c:v")
	args = append(args, cfg.VideoEncoder)

	switch cfg.VideoEncoder {
	case "copy":
		// No options
	case "libx264", "libx265":
		args = append(args, "-crf")
		args = append(args, cfg.CRF)

		args = append(args, "-preset")
		args = append(args, cfg.Preset)
	case "libvpx-vp9":
		args = append(args, "-crf")
		args = append(args, cfg.CRF)
	case "librav1e":
		// No options
	case "libsvtav1":
		args = append(args, "-crf")
		args = append(args, cfg.CRF)

		// TODO: preset is a number for this encoder (-2 to 13)
	}

	switch cfg.AudioEncoder {
	case "None":
		args = append(args, "-an")
	case "copy", "aac", "libopus":
		args = append(args, "-c:a")
		args = append(args, cfg.AudioEncoder)
	}

	args = append(args, additionalArgs...)

	// Output file
	args = append(args, outFullFilePath)

	return args
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
	newFileFullPath := filepath.Join(parentDir, newFileName+fmt.Sprintf("_[%s]_[%s]", cfg.VideoEncoder, cfg.AudioEncoder)+extension)

	if _, err := os.Stat(newFileFullPath); err == nil {
		if cfg.SkipEncodedVid {
			log.Printf("Skipping \"%s\" because it already exists with the exact same encodings (crf and preset might be different though)", newFileFullPath)
			teaP.Send(finishedEncodingVideo{})
			return
		} else {
			os.Remove(newFileFullPath)
		}
	}

	cmdArgs := buildFFmpegCmdArgs(fullFilePath, newFileFullPath, cfg, "-progress", "unix://"+getProgressSocket(fullFilePath, teaP))
	cmd := exec.Command("ffmpeg", cmdArgs...)
	teaP.Send(ffmpegProcessStart{cmd})
	err = cmd.Run()

	if err != nil {
		teaP.Send(errQuitMsg{msg: fmt.Sprintf("FFmpeg exited with error code: %s\n\nError: %v", err, cmd.Stderr)})
	}
}
