package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func getTitle(id string) (title string) {
	// Check the OS platform, if windows, specify a path to yt-dlp.exe
	var ytdlp string
	if platform == "windows" {
		ytdlp = thisFilePath + "\\yt-dlp.exe"
	} else {
		ytdlp = "yt-dlp"
	}

	// Run yt-dlp from cmd
	cmd := exec.Command(
		ytdlp,
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

	// Convert byte array to string (utf-8)
	if platform != "windows" {
		return string(output)
	}

	// Convert byte array to string (big5) **Not accurate
	decodedString, err := decodeBIG5(output)
	if err != nil {
		return ""
	}

	return strings.TrimSuffix(decodedString, "\n")
}

func search(title string) (videoArray []Video, err error) {
	// Check the OS platform, if windows, specify a path to yt-dlp.exe
	var ytdlp string
	if platform == "windows" {
		ytdlp = thisFilePath + "\\yt-dlp.exe"
	} else {
		ytdlp = "yt-dlp"
	}

	maxResult := 4
	// Run yt-dlp from cmd
	cmd := exec.Command(
		ytdlp,
		[]string{
			"--skip-download",
			"--no-playlist",
			"--quiet",
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
	if platform != "windows" {
		scanner = bufio.NewScanner(bytes.NewReader(output))
		scanner.Split(bufio.ScanLines)

	} else if platform == "windows" {
		decodedString, err := decodeBIG5(output)
		if err != nil {
			return nil, err
		}

		scanner = bufio.NewScanner(strings.NewReader(decodedString))
		scanner.Split(bufio.ScanLines)
	}

	// Scan the output line by line and convert to array
	var text []string
	for scanner.Scan() {
		text = append(text, scanner.Text())
	}

	// Loop all the scanned text and append video array
	for loopIndex := 0; loopIndex < len(text); loopIndex = loopIndex + 2 {
		videoArray = append(videoArray, Video{
			Id:    strings.TrimSuffix(text[loopIndex+1], "\n"),
			Title: strings.TrimSuffix(text[loopIndex], "\n"),
		})
	}

	return videoArray, nil
}
