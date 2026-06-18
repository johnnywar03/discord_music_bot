package main

import (
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"sync"
)

// ffmpegOpusEncoder runs ffmpeg to transcode an audio file into Opus frames
// wrapped in an OGG container, then parses the OGG stream to yield individual
// Opus packets. This replaces the unmaintained github.com/jonas747/dca
// dependency with ~80 lines of straightforward OGG page parsing.
type ffmpegOpusEncoder struct {
	cmd    *exec.Cmd
	frames chan []byte  // opus packets extracted from the OGG stream
	done   chan struct{} // closed by Close() to unblock the parser goroutine
	once   sync.Once
	err    error // set by the parser goroutine before closing frames
}

// newFFmpegOpusEncoder starts an ffmpeg process that encodes filePath to Opus
// inside an OGG container and streams the output to a background goroutine that
// extracts individual Opus packets.
func newFFmpegOpusEncoder(filePath string, bitrate int) (*ffmpegOpusEncoder, error) {
	cmd := exec.Command("ffmpeg",
		"-hide_banner", "-loglevel", "error",
		"-i", filePath,
		"-c:a", "libopus",
		"-b:a", fmt.Sprintf("%dk", bitrate),
		"-application", "lowdelay",
		"-frame_duration", "20",
		"-vn",
		"-f", "ogg",
		"pipe:1",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("ffmpeg start: %w", err)
	}

	enc := &ffmpegOpusEncoder{
		cmd:    cmd,
		frames: make(chan []byte, 64),
		done:   make(chan struct{}),
	}

	go func() {
		enc.err = parseOggOpusFrames(stdout, enc.frames, enc.done)
		close(enc.frames)
	}()

	return enc, nil
}

// NextFrame returns the next Opus frame. It blocks until a frame is available
// or the encoder is exhausted / closed, in which case it returns io.EOF.
func (e *ffmpegOpusEncoder) NextFrame() ([]byte, error) {
	frame, ok := <-e.frames
	if !ok {
		if e.err != nil {
			return nil, e.err
		}
		return nil, io.EOF
	}
	return frame, nil
}

// Close terminates the ffmpeg process and releases all resources. Safe to call
// multiple times.
func (e *ffmpegOpusEncoder) Close() {
	e.once.Do(func() {
		close(e.done)
		if e.cmd.Process != nil {
			_ = e.cmd.Process.Kill()
		}
		_ = e.cmd.Wait()
	})
}

// parseOggOpusFrames reads an OGG byte stream from r, reconstructs Opus
// packets from the OGG page/segment structure, and sends each audio packet
// (skipping the mandatory OpusHead and OpusTags header packets) to frames.
//
// The parser exits cleanly when r reaches EOF or when done is closed.
func parseOggOpusFrames(r io.Reader, frames chan<- []byte, done <-chan struct{}) error {
	var hdr [27]byte
	packetCount := 0
	var packet []byte

	for {
		// --- Read the 27-byte OGG page header ---
		if _, err := io.ReadFull(r, hdr[:]); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return nil // normal termination
			}
			return fmt.Errorf("ogg header read: %w", err)
		}
		if string(hdr[:4]) != "OggS" {
			return fmt.Errorf("invalid OGG sync pattern: %x", hdr[:4])
		}

		// --- Read the segment table ---
		nSegs := int(hdr[26])
		segTable := make([]byte, nSegs)
		if _, err := io.ReadFull(r, segTable); err != nil {
			return fmt.Errorf("ogg segment table read: %w", err)
		}

		// --- Read segments and reconstruct packets ---
		// An OGG packet is complete when a segment is shorter than 255 bytes.
		// Segments of exactly 255 bytes signal continuation.
		for _, segLen := range segTable {
			if segLen > 0 {
				seg := make([]byte, int(segLen))
				if _, err := io.ReadFull(r, seg); err != nil {
					return fmt.Errorf("ogg segment data read: %w", err)
				}
				packet = append(packet, seg...)
			}

			if segLen < 255 {
				// Packet boundary reached.
				packetCount++
				// The first two packets in an Opus OGG stream are always
				// OpusHead (identification) and OpusTags (comments). Skip them.
				if packetCount > 2 && len(packet) > 0 {
					frame := make([]byte, len(packet))
					copy(frame, packet)
					select {
					case frames <- frame:
					case <-done:
						slog.Debug("ogg parser: done signal received, exiting")
						return nil
					}
				}
				packet = packet[:0]
			}
		}
	}
}
