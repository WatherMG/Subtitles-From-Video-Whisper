package main

import (
	ffmpeg "github.com/u2takey/ffmpeg-go"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const AudioExtension = ".mp3"
const VideoExtension = ".mp4"
const VideoRoot = "D:\\Projects\\whisper\\video"
const AudioRoot = "D:\\Projects\\whisper\\audio"

func getFilesPath(root, pattern string) ([]string, error) {
	var matches []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if matched, err := filepath.Match(pattern, filepath.Base(path)); err != nil {
			return err
		} else if matched {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func setFileExtension(parent string, filePath string, old string, new string) string {
	return parent + "\\" + filepath.Base(strings.Replace(filePath, old, new, 1))
}

func getOutputDir(file string, from string, to string) string {
	return strings.Replace(filepath.Dir(file), from, to, 1)
}

func makeFile(path string, subtitle bool) {
	var existFiles = 0
	var createdFiles = 0
	var outputDir string
	var typeOfFile = "*" + VideoExtension
	if subtitle {
		typeOfFile = "*" + AudioExtension
	}
	files, err := getFilesPath(path, typeOfFile)
	if err == nil {
		var existFile error
		for _, file := range files {
			if subtitle {
				outputDir = getOutputDir(file, "audio", "subtitles")
				_, existFile = os.Stat(setFileExtension(outputDir, file, AudioExtension, AudioExtension+".srt"))
			} else {
				outputDir = getOutputDir(file, "video", "audio")
				_, existFile = os.Stat(setFileExtension(outputDir, file, VideoExtension, AudioExtension))
			}
			aErr := os.Mkdir(outputDir, 0777)
			if aErr != nil && !os.IsExist(aErr) {
				log.Fatal(aErr)
			}
			if os.IsNotExist(existFile) {
				if subtitle {
					var back = "cd ..\\; "
					var venv = ".\\activate.ps1; " + back + back
					var language = "--language en "
					var model = "--model tiny.en "
					var device = "--device cuda "
					var out = `--output_dir "` + outputDir + `"`
					var whisper = "whisper " + language + model + device + out + ` -- "` + file + `"`
					cmd := exec.Command("powershell", venv+whisper)
					cmd.Dir = "D:\\Projects\\whisper\\.venv\\Scripts"
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					log.Printf("Start transcribe %s ", filepath.Base(file))
					err := cmd.Run()
					if err != nil {
						log.Printf("Something went wrong: %s", err)
					}
					log.Printf("File %s is transcribed", filepath.Base(file))
				} else {
					err := ffmpeg.Input(file).
						Output(setFileExtension(outputDir, file, VideoExtension, AudioExtension), ffmpeg.KwArgs{"b:a": "320K", "vn": ""}).
						OverWriteOutput().Run()
					createdFiles += 1
					if err != nil {
						println(err)
					}
				}
			} else {
				existFiles += 1
			}
		}
	}
	println("Created audio:", createdFiles)
	println("Skipped video:", existFiles)
}

func main() {

	start := time.Now()

	makeFile(VideoRoot, false)
	makeFile(AudioRoot, true)

	elapsed := time.Since(start)
	log.Printf("Time to complete %s", elapsed)

}
