package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func getTitle(id string) (title string) {
	// Run yt-dlp from cmd
	cmd := exec.Command(
		"yt-dlp",
		[]string{
			"--skip-download",
			"--get-title",
			"--quiet",
			id,
		}...,
	)
	// Get cmd output
	output, err := cmd.Output()
	if err != nil {
		println("Failed to get video title, ", err.Error())
		return ""
	}

	return strings.TrimSuffix(string(output), "\n")
}

func search(title string) (videoArray []Video, err error) {
	maxResult := 4
	// Run yt-dlp from cmd
	cmd := exec.Command(
		"yt-dlp",
		[]string{
			"--skip-download",
			"--no-playlist",
			"--flat-playlist",
			"--quiet",
			"--ignore-errors",
			"--get-id",
			"--get-title",
			"--default-search", "ytsearch",
			fmt.Sprintf("ytsearch%d:%s", maxResult, title),
		}...,
	)
	// Get cmd output
	output, err := cmd.Output()
	if err != nil {
		println("Failed to search video, ", err.Error())
		return nil, err
	}

	// Scan the output line by line
	var scanner *bufio.Scanner
	scanner = bufio.NewScanner(bytes.NewReader(output))
	scanner.Split(bufio.ScanLines)

	// Scan the output line by line and convert to array
	var text []string
	for scanner.Scan() {
		text = append(text, scanner.Text())
	}

	// Loop all the scanned text and append video array
	for loopIndex := 0; loopIndex < len(text); loopIndex = loopIndex + 2 {
		if len(strings.TrimSuffix(text[loopIndex+1], "\n")) == 11 {
			videoArray = append(videoArray, Video{
				Id:    strings.TrimSuffix(text[loopIndex+1], "\n"),
				Title: strings.TrimSuffix(text[loopIndex], "\n"),
			})
		}
	}

	return videoArray, nil
}

func download(id string) (filePath string, err error) {
	downloadPath := thisFilePath + "/video/"
	// Run yt-dlp from cmd
	cmd := exec.Command(
		"yt-dlp",
		[]string{
			fmt.Sprintf("-P %s", downloadPath),
			"-o%(id)s.%(ext)s",
			"-x",
			"--audio-format", "mp3",
			"--audio-quality", "128K",
			id,
		}...,
	)
	err = cmd.Run()
	if err != nil {
		println("Error in downloading video: ", err.Error())
		return "", err
	}
	return fmt.Sprintf("%s/video/%s.mp3", thisFilePath, id), nil
}
