package main

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ytChannel struct {
	id,
	lastCheck string
}

const tickerDuration = 20 * time.Minute

var (
	configPath string
	processing = make(chan struct{}, 7)
	wg         sync.WaitGroup
	ytChannels []ytChannel
)

func downloadVideo(channelInfo ytChannel, videoIndex int) {
	wg.Add(1)
	defer wg.Done()

	processing <- struct{}{}

	vidIdx := strconv.Itoa(videoIndex)

	cmd := exec.Command("yt-dlp",
		"--dateafter",
		channelInfo.lastCheck,
		"--format",
		"bestvideo[height<=1440]+ba",
		"--output",
		"/yt-videos/%(upload_date)s # %(uploader)s # %(id)s.%(ext)s",
		"--playlist-items",
		vidIdx+":"+vidIdx,
		"--remux-video",
		"mp4",
		"--windows-filenames",
		"https://youtube.com/channel/"+channelInfo.id+"/videos")

	cmd.Run()

	<-processing
}

func loadAndSaveYTChannelList() {
	today := time.Now().Format("20060102")

	loadYTChannelList(today)

	saveYTChannelList(today)
}

func loadYTChannelList(currentDate string) {
	ytChannels = nil

	f, _ := os.Open(configPath)
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		text := strings.Split(scanner.Text(), ",")

		ytCn := ytChannel{
			id:        text[0],
			lastCheck: currentDate,
		}

		if len(text) == 2 {
			ytCn.lastCheck = text[1]
		}

		ytChannels = append(ytChannels, ytCn)
	}
}

func mainProcessing() {
	loadAndSaveYTChannelList()

	processYTChannels()
}

func processYTChannels() {
	for _, ytCn := range ytChannels {
		for idx := 1; idx <= 5; idx++ {
			go downloadVideo(ytCn, idx)
		}
	}

	wg.Wait()
}

func saveYTChannelList(currentDate string) {
	if len(ytChannels) == 0 {
		return
	}

	f, _ := os.Create(configPath)
	defer f.Close()

	for _, ytCn := range ytChannels {
		f.WriteString(ytCn.id + "," + currentDate + "\n")
	}
}

func init() {
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)

	configPath = filepath.Join(exeDir, "config")
}

func main() {
	mainProcessing()

	ticker := time.NewTicker(tickerDuration)
	defer ticker.Stop()

	for {
		<-ticker.C

		mainProcessing()

		ticker.Reset(tickerDuration)
	}
}
