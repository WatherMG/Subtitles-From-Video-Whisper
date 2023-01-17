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
const videoPath = "D:\\Projects\\whisper\\video"
const audioPath = "D:\\Projects\\whisper\\audio"
const subtitlePath = "D:\\Projects\\whisper\\subtitles"

var wg sync.WaitGroup

var maxFilesPerDecode = 12
var maxFilesPerTranscribe = 1
var maxFilesPerPass int

var countFiles int

func isOutputFileExist(outputFile string, extension string) (string, bool) {
	outputFile = strings.Replace(outputFile, filepath.Ext(outputFile), extension, -1)
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		return outputFile, true
	}
	return outputFile, false
}

func getOutputDir(relativePath string, isCreatingMP3 bool) string {
	dir := subtitlePath
	if isCreatingMP3 {
		dir = audioPath
	}
	return filepath.Join(dir, relativePath)
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
	//defer os.RemoveAll(audioPath)
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
			outputFile := getOutputDir(relativePath, isCreatingMP3)
			outputDir := filepath.Dir(outputFile)
			outputFile, isFileExist := isOutputFileExist(outputFile, AudioExtension)

			if isFileExist || !isCreatingMP3 {
				_ = os.MkdirAll(outputDir, os.ModePerm)
				if isCreatingMP3 {
					sem <- struct{}{}
					start := time.Now()
					log.Printf("Start decode %s", filepath.Base(inputFile))
					err = ffmpeg.Input(inputFile).
						Output(outputFile, ffmpeg.KwArgs{"b:a": "320K", "vn": ""}).Run()
					if err != nil {
						log.Printf("Something went wrong in decode: %s", err)
						log.Fatal(err)
					}
					log.Printf("Finish decode %s in %s", filepath.Base(inputFile), time.Since(start))
					countFiles += 1
					<-sem
				} else {
					_, isFileExist := isOutputFileExist(outputFile, AudioExtension+SubtitleExtension)
					if isFileExist {
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
		//fmt.Printf("visited file or dir: %q\n", inputFile)
		return nil
	})
	//log.Println("Now remove all files from audio dir")
	wg.Wait()
}

func main() {
	startTime := time.Now()
	createFiles(videoPath, true)
	createFiles(audioPath, false)
	elapsed := time.Since(startTime)
	log.Printf("Time to complete is: %s\n", elapsed)
	log.Printf("Created audio files: %d\n", countFiles)
}
