package main

import (
	ffmpeg "github.com/u2takey/ffmpeg-go"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const AudioExtension = ".mp3"
const VideoExtension = ".mp4"
const SubtitleExtension = ".srt"
const VideoDir = "D:\\Projects\\whisper\\video"
const AudioDir = "D:\\Projects\\whisper\\audio"
const OutputDir = "D:\\Projects\\whisper\\output"

var wg sync.WaitGroup

var maxFilesPerDecode = 5
var maxFilesPerTranscribe = 1
var maxFilesPerPass int

func isOutputFileExist(outputFile string, extension string) bool {
	outputFile = strings.Replace(outputFile, filepath.Ext(outputFile), extension, -1)
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		return true
	}
	return false
}

func createFiles(inputDir string, isCreatingMP3 bool) {
	var extension string
	if isCreatingMP3 {
		extension = VideoExtension
		maxFilesPerPass = maxFilesPerDecode
	} else {
		extension = AudioExtension
		maxFilesPerPass = maxFilesPerTranscribe
	}
	sem := make(chan struct{}, maxFilesPerPass)
	defer close(sem)
	_ = filepath.Walk(inputDir, func(inputFile string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(inputFile) != extension {
			return nil
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			relativePath, _ := filepath.Rel(inputDir, inputFile)
			outputFile := filepath.Join(OutputDir, relativePath)
			outputDir := filepath.Dir(outputFile)

			if isOutputFileExist(outputFile, AudioExtension) || !isCreatingMP3 {
				_ = os.MkdirAll(outputDir, os.ModePerm)
				if isCreatingMP3 {
					sem <- struct{}{}
					start := time.Now()
					log.Printf("Start decode %s", filepath.Base(inputFile))
					err = ffmpeg.Input(inputFile).
						Output(outputFile, ffmpeg.KwArgs{"b:a": "320K", "vn": ""}).OverWriteOutput().Run()
					if err != nil {
						log.Printf("Something went wrong in decode: %s", err)
					}
					log.Printf("Finish decode %s in %s", filepath.Base(inputFile), time.Since(start))
					<-sem
				} else {
					if isOutputFileExist(outputFile, ".mp3"+SubtitleExtension) {
						sem <- struct{}{}
						var activateVenv = ".\\activate.ps1; "
						cmd := exec.Command("powershell", activateVenv+"whisper --language en "+
							"--model medium.en --device cuda --o "+strconv.Quote(outputDir)+" --verbose False -- "+strconv.Quote(inputFile))
						cmd.Dir = "D:\\Projects\\whisper\\.venv\\Scripts"
						cmd.Stdout = os.Stdout
						cmd.Stderr = os.Stderr
						start := time.Now()
						log.Printf("Start transcribe %s", filepath.Base(inputFile))
						err := cmd.Run()
						if err != nil {
							log.Printf("Something went wrong in transcribe: %s", err)
						}
						log.Printf("Finish transcribe %s in %s", filepath.Base(inputFile), time.Since(start))
						<-sem
					}
				}
			}
		}()

		return nil
	})
	wg.Wait()
}

func main() {
	startTime := time.Now()
	createFiles(VideoDir, true)
	createFiles(AudioDir, false)
	elapsed := time.Since(startTime)
	log.Printf("Time to complete: %s\n", elapsed)
}
