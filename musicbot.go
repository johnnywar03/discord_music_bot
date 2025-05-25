package main

import (
	"io"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
)

type MusicBot struct {
	IsPlaying        bool
	NowPlaying       *Video
	StreamingSession *dca.StreamingSession
}

func playVideo(session *discordgo.Session, guildId string) {
	// Check if the bot is playing video
	if musicBot.IsPlaying {
		return
	}
	// Check if the bot is in voice channel
	if !checkJoinedVoiceChannel(session, guildId) {
		return
	}

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

		select {
		case err = <-done:
			if err != nil && err != io.EOF {
				println("Error: playing video, ", err.Error())
				break
			}
		}

		voiceConnection.Speaking(false)
		musicBot.StreamingSession = nil
		// Remove played video
		videoQueue.deleteFirst()
		// Clean up dca encode session
		dcaEncodeSession.Cleanup()
	}

	// Clean up the music bot after playing all video
	musicBot.IsPlaying = false
	musicBot.NowPlaying = nil
	voiceConnection.Disconnect()

	// Remove all downloaded video
	remove(thisFilePath + "\\video")
}
