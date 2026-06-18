package main

import (
	"context"
	"io"
	"log/slog"
	"sync"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/snowflake/v2"
)

// MusicBot manages audio playback for a single guild. The exported fields from
// the previous version (IsPlaying, NowPlaying, SkipChannel, StopChannel) have
// been replaced with unexported equivalents and proper synchronisation.
type MusicBot struct {
	mu         sync.Mutex
	playing    bool
	nowPlaying *Video
	skipCh     chan struct{} // buffered(1); one signal consumed per track
	stopCh     chan struct{} // buffered(1); breaks the playback loop
}

// NewMusicBot creates a ready-to-use MusicBot.
func NewMusicBot() *MusicBot {
	return &MusicBot{
		skipCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}, 1),
	}
}

// IsPlaying reports whether the bot is currently playing audio.
// Safe to call from any goroutine.
func (mb *MusicBot) IsPlaying() bool {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	return mb.playing
}

// NowPlaying returns the currently playing video, or nil.
// Safe to call from any goroutine.
func (mb *MusicBot) NowPlaying() *Video {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	return mb.nowPlaying
}

// playbackResult describes why a track's playback loop ended.
type playbackResult int

const (
	playbackFinished playbackResult = iota
	playbackSkipped
	playbackStopped
)

// ---------------------------------------------------------------------------
// opusFrameProvider — adapts ffmpegOpusEncoder to disgo's voice.OpusFrameProvider
// ---------------------------------------------------------------------------

// opusFrameProvider bridges the ffmpegOpusEncoder (pull-based Opus frames) to
// disgo's voice.OpusFrameProvider interface. disgo calls ProvideOpusFrame every
// 20 ms from its audio-sender goroutine.
type opusFrameProvider struct {
	encoder *ffmpegOpusEncoder
	skipCh  <-chan struct{}
	stopCh  <-chan struct{}

	done chan playbackResult
	once sync.Once
}

var _ voice.OpusFrameProvider = (*opusFrameProvider)(nil)

func newOpusFrameProvider(enc *ffmpegOpusEncoder, skipCh, stopCh <-chan struct{}) *opusFrameProvider {
	return &opusFrameProvider{
		encoder: enc,
		skipCh:  skipCh,
		stopCh:  stopCh,
		done:    make(chan playbackResult, 1),
	}
}

func (p *opusFrameProvider) finish(r playbackResult) {
	p.once.Do(func() { p.done <- r })
}

// ProvideOpusFrame is called by disgo's audio sender roughly every 20 ms.
func (p *opusFrameProvider) ProvideOpusFrame() ([]byte, error) {
	// Check for skip/stop before reading the next frame so the interruption
	// takes effect within one frame period (~20 ms).
	select {
	case <-p.skipCh:
		p.finish(playbackSkipped)
		return nil, io.EOF
	case <-p.stopCh:
		p.finish(playbackStopped)
		return nil, io.EOF
	default:
	}

	frame, err := p.encoder.NextFrame()
	if err != nil {
		p.finish(playbackFinished)
		return nil, io.EOF
	}
	return frame, nil
}

// Close cleans up the underlying ffmpeg process.
func (p *opusFrameProvider) Close() {
	p.encoder.Close()
}

// ---------------------------------------------------------------------------
// Playback loop
// ---------------------------------------------------------------------------

// playVideo drives the main playback loop: for each video in the queue it
// downloads, encodes to Opus via ffmpeg, and streams the frames to Discord.
//
// The function holds mb.mu for its entire lifetime so that concurrent callers
// (e.g. multiple /play commands) serialise naturally — the second caller sees
// mb.playing == true and returns immediately.
func (mb *MusicBot) playVideo(ctx context.Context, client *bot.Client, guildID snowflake.ID) {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if mb.playing {
		return
	}
	if !checkJoinedVoiceChannel(client, guildID) {
		return
	}

	mb.playing = true

	// Deferred cleanup runs regardless of how the loop exits (normal
	// completion, error, or stop signal). leaveVoiceChannel uses a fresh
	// context because ctx may have been cancelled by then.
	defer func() {
		mb.playing = false
		mb.nowPlaying = nil
		if err := remove(thisFilePath + "/video"); err != nil {
			slog.Warn("failed to clean up video directory", "error", err)
		}
		leaveVoiceChannel(context.Background(), client, guildID)
	}()

	conn := client.VoiceManager.GetConn(guildID)

	for videoQueue.CurrentVideo != nil {
		mb.nowPlaying = videoQueue.getTheFirst()

		// --- Download ---
		videoPath, err := download(mb.nowPlaying.Id)
		if err != nil {
			slog.Error("failed to download video", "id", mb.nowPlaying.Id, "error", err)
			break
		}

		// --- Encode to Opus via ffmpeg (replaces dca.EncodeFile) ---
		encoder, err := newFFmpegOpusEncoder(videoPath, 96)
		if err != nil {
			slog.Error("failed to start ffmpeg encoder", "path", videoPath, "error", err)
			break
		}

		sendMessageToChannel(client, "Now playing: "+mb.nowPlaying.Title)

		// Drain any stale skip signal left over from a previous track so it
		// does not immediately skip this one.
		select {
		case <-mb.skipCh:
		default:
		}

		// Hand frames to the voice connection. disgo pulls frames at a steady
		// 20 ms cadence and toggles the speaking indicator automatically.
		provider := newOpusFrameProvider(encoder, mb.skipCh, mb.stopCh)
		conn.SetOpusFrameProvider(provider)

		// Block until the track finishes, is skipped, or is stopped.
		result := <-provider.done
		provider.Close()

		videoQueue.deleteFirst()

		switch result {
		case playbackStopped:
			sendMessageToChannel(client, "Stopped: "+mb.nowPlaying.Title)
			videoQueue.CurrentVideo = nil
			return
		case playbackSkipped:
			sendMessageToChannel(client, "Skipped: "+mb.nowPlaying.Title)
		default:
			// playbackFinished — advance to next track naturally.
		}
	}
}

// skip signals the provider to end the current track and advance to the next
// one in the queue. The non-blocking send ensures callers never block even if
// the bot is between tracks (during download).
func (mb *MusicBot) skip() {
	select {
	case mb.skipCh <- struct{}{}:
	default:
	}
}

// stop signals the playback loop to break and leave the voice channel. Like
// skip, the send is non-blocking.
func (mb *MusicBot) stop() {
	select {
	case mb.stopCh <- struct{}{}:
	default:
	}
}
