package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func getProgressSocket(inFileName string, teaP *tea.Program) string {
	cmd := exec.Command("ffprobe", "-show_format", "-show_streams", "-of", "json", inFileName)
	probe, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	totalDuration, err := probeDuration(string(probe))
	if err != nil {
		panic(err)
	}

	return TempSock(totalDuration, teaP)
}

func TempSock(totalDuration float64, teaP *tea.Program) string {
	// serve

	sockFileName := path.Join(os.TempDir(), fmt.Sprintf("%d_sock", rand.Int()))
	l, err := net.Listen("unix", sockFileName)
	if err != nil {
		panic(err)
	}

	go func() {
		re := regexp.MustCompile(`out_time_ms=(\d+)`)
		speedRe := regexp.MustCompile(`speed=(\d+\.\d+x)`)

		fd, err := l.Accept()
		if err != nil {
			log.Fatal("accept error:", err)
		}
		buf := make([]byte, 1024)
		data := ""
		progress := ""
		estimate := 0

		for {
			_, err := fd.Read(buf)
			if err != nil {
				return
			}
			data = string(buf)
			a := re.FindAllStringSubmatch(data, -1)
			cp := ""
			if len(a) > 0 && len(a[len(a)-1]) > 0 {
				c, _ := strconv.Atoi(a[len(a)-1][len(a[len(a)-1])-1])
				cp = fmt.Sprintf("%.2f", float64(c)/totalDuration/1000000)
			}
			if strings.Contains(data, "progress=end") {
				cp = "done"
				teaP.Send(finishedEncodingVideo{})
				return
			}
			if cp == "" {
				cp = ".0"
			}
			if cp != progress {
				progress = cp
				progressFloat, err := strconv.ParseFloat(progress, 64)
				if err == nil {
					teaP.Send(updateProgress{
						progress: progressFloat,
					})
				}
			}

			speedMatch := speedRe.FindString(data)
			if len(speedMatch) > 0 {
				trimmedSpeed := strings.TrimSuffix(strings.Split(speedMatch, "=")[1], "x")
				speed, _ := strconv.ParseFloat(trimmedSpeed, 32)
				progressFloat, _ := strconv.ParseFloat(progress, 64)
				remainingDuration := totalDuration - (totalDuration * progressFloat)
				estimate = int(remainingDuration / speed)
			}

			teaP.Send(updateEstimate{
				estimate: estimate,
			})
		}
	}()

	return sockFileName
}

type probeFormat struct {
	Duration string `json:"duration"`
}

type probeData struct {
	Format probeFormat `json:"format"`
}

func probeDuration(a string) (float64, error) {
	pd := probeData{}
	err := json.Unmarshal([]byte(a), &pd)
	if err != nil {
		return 0, err
	}
	f, err := strconv.ParseFloat(pd.Format.Duration, 64)
	if err != nil {
		return 0, err
	}
	return f, nil
}
