package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func getTitle(id string) (title string) {
	cmd := exec.Command(
		"yt-dlp",
		"--skip-download",
		"--get-title",
		"--quiet",
		id,
	)
	output, err := cmd.Output()
	if err != nil {
		println("Failed to get video title, ", err.Error())
		return ""
	}
	return strings.TrimSuffix(string(output), "\n")
}

func search(title string) (videoArray []Video, err error) {
	maxResult := 4
	cmd := exec.Command(
		"yt-dlp",
		"--skip-download",
		"--no-playlist",
		"--flat-playlist",
		"--quiet",
		"--ignore-errors",
		"--get-id",
		"--get-title",
		"--default-search", "ytsearch",
		fmt.Sprintf("ytsearch%d:%s", maxResult, title),
	)
	output, err := cmd.Output()
	if err != nil {
		println("Failed to search video, ", err.Error())
		return nil, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	scanner.Split(bufio.ScanLines)

	var text []string
	for scanner.Scan() {
		text = append(text, scanner.Text())
	}

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

// download fetches the best available audio stream for the given video ID and
// returns the path to the downloaded file.
//
// Previously this extracted audio as mp3 (--audio-format mp3), which meant
// the subsequent dca encode step would transcode mp3 → opus — a lossy-to-lossy
// conversion. Now that we use ffmpeg directly for Opus encoding (see
// encode.go), we download the best audio in its native container and let
// ffmpeg handle the single transcode to Opus.
func download(id string) (filePath string, err error) {
	downloadPath := filepath.Join(thisFilePath, "video")
	if err := os.MkdirAll(downloadPath, 0755); err != nil {
		return "", fmt.Errorf("create download dir: %w", err)
	}

	cmd := exec.Command(
		"yt-dlp",
		"-P", downloadPath,
		"-o", "%(id)s.%(ext)s",
		"-f", "bestaudio",
		"--no-playlist",
		id,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("yt-dlp: %w: %s", err, stderr.String())
	}

	// The actual extension depends on what YouTube serves (typically .webm or
	// .m4a), so we locate the file by globbing for the video ID.
	matches, err := filepath.Glob(filepath.Join(downloadPath, id+".*"))
	if err != nil || len(matches) == 0 {
		return "", fmt.Errorf("downloaded file not found for %s", id)
	}
	return matches[0], nil
}
