package main

import (
	"fmt"
	"io"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
)

type MusicBot struct {
	IsPlaying        bool
	NowPlaying       *Video
	StreamingSession *dca.StreamingSession
	Mutex            sync.Mutex
	SkipChannel      chan bool
	StopChannel      chan bool
}

func (musicbot *MusicBot) playVideo(session *discordgo.Session, guildId string) {
	// Check if the bot is playing video
	if musicBot.IsPlaying {
		return
	}
	// Check if the bot is in voice channel
	if !checkJoinedVoiceChannel(session, guildId) {
		return
	}

	musicBot.Mutex.Lock()

	voiceConnection := session.VoiceConnections[guildId]

	for videoQueue.CurrentVideo != nil {
		// Download video
		musicBot.NowPlaying = videoQueue.getTheFirst()
		videoPath, err := download(musicBot.NowPlaying.Id)
		if err != nil {
			musicBot.IsPlaying = false
			break
		}

		// Encode the video using dca
		options := dca.StdEncodeOptions
		options.RawOutput = true
		options.Bitrate = 96
		options.Application = "lowdelay"

		dcaEncodeSession, err := dca.EncodeFile(videoPath, options)
		if err != nil {
			println("Fail to encode the video, ", err.Error())
			musicBot.IsPlaying = false
			break
		}

		// Send now playing message to the channel
		sendMessageToChannel(session, "Now playing: "+musicBot.NowPlaying.Title)

		// Set music bot ready to speak
		musicBot.IsPlaying = true
		voiceConnection.Speaking(true)

		// Play the video
		done := make(chan error)
		musicBot.StreamingSession = dca.NewStream(dcaEncodeSession, voiceConnection, done)

		skipped := false
		stopped := false

		// Wait for completion, skip, or stop
		select {
		case err = <-done:
			if err != nil && err != io.EOF {
				println("Error: playing video, ", err.Error())
			}
		case <-musicBot.SkipChannel:
			skipped = true
		case <-musicBot.StopChannel:
			stopped = true
		}

		voiceConnection.Speaking(false)

		// Stop the streaming session if it's still running
		if musicBot.StreamingSession != nil {
			musicBot.StreamingSession.SetPaused(true)
			musicBot.StreamingSession = nil
		}

		// Clean up dca encode session
		dcaEncodeSession.Cleanup()

		// Remove played/skipped video
		videoQueue.deleteFirst()

		// If stopped, break out of the loop
		if stopped {
			sendMessageToChannel(session, "Stopped: "+musicbot.NowPlaying.Title)
			videoQueue.CurrentVideo = nil
			break
		}

		// If skipped, send skip message
		if skipped {
			sendMessageToChannel(session, "Skipped: "+musicBot.NowPlaying.Title)
		}
	}

	// Clean up the music bot after playing all video
	musicBot.IsPlaying = false
	musicBot.NowPlaying = nil

	// Remove all downloaded video
	remove(thisFilePath + "/video")

	// Unlock mutex
	musicBot.Mutex.Unlock()
	leaveVoiceChannel(session, guildId)
}

func (musicBot *MusicBot) skip() {
	if musicBot.IsPlaying && musicBot.StreamingSession != nil {
		select {
		case musicBot.SkipChannel <- true:
		default:
		}
	}
}

func (musicBot *MusicBot) stop() {
	if musicBot.IsPlaying && musicBot.StreamingSession != nil {
		select {
		case musicBot.StopChannel <- true:
		default:
		}
	}
}

func (musicBot *MusicBot) nowPlaying(session *discordgo.Session, interactionCreatedEvent *discordgo.InteractionCreate) {
	responseToInteraction(session, interactionCreatedEvent.Interaction, fmt.Sprintf("Now Playing:\n%s", musicBot.NowPlaying.Title))
}
