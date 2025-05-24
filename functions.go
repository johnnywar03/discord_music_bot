package main

import (
	"github.com/bwmarrin/discordgo"
)

func joinVoiceChannel(session *discordgo.Session, interactionCreatedEvent *discordgo.InteractionCreate) {
	// Check the voice state of the application command issuer
	voiceState, err := session.State.VoiceState(interactionCreatedEvent.GuildID, interactionCreatedEvent.Member.User.ID)
	if err != nil {
		println("Error in getting voice state, ", err.Error())
		responseToInteraction(session, interactionCreatedEvent.Interaction, "You must join a channel to use this command!")
		return
	}

	// Join the voice channel
	_, err = session.ChannelVoiceJoin(voiceState.GuildID, voiceState.ChannelID, false, true)
	if err != nil {
		println("Failed to join a voice channel, ", err.Error())
		responseToInteraction(session, interactionCreatedEvent.Interaction, "Failed to join a voice channel.")
	}
	responseToInteraction(session, interactionCreatedEvent.Interaction, "Hi!")
}

func leaveVoiceChannel(session *discordgo.Session, interactionCreatedEvent *discordgo.InteractionCreate) {
	// Check the bot voice connection
	voiceConnection, joined := session.VoiceConnections[interactionCreatedEvent.GuildID]
	if joined {
		// Disconnect if joined a voice channel
		err := voiceConnection.Disconnect()
		if err != nil {
			println("Failed to leave the voice channel, ", err.Error())
			responseToInteraction(session, interactionCreatedEvent.Interaction, "Failed to leave voice channel.")
		}
		responseToInteraction(session, interactionCreatedEvent.Interaction, "Bye!")
	} else if !joined {
		responseToInteraction(session, interactionCreatedEvent.Interaction, "The bot has already left.")
	}
}
