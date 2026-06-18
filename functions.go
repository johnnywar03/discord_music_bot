package main

import (
	"context"
	"errors"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/snowflake/v2"
)

func joinVoiceChannel(ctx context.Context, client *bot.Client, guildID snowflake.ID, userID snowflake.ID) (err error) {
	// Check if joined a voice channel
	if checkJoinedVoiceChannel(client, guildID) {
		return nil
	}
	// Check the voice state of the application command issuer.
	// Requires cache.FlagVoiceStates to be enabled (see main.go).
	voiceState, ok := client.Caches.VoiceState(guildID, userID)
	if !ok || voiceState.ChannelID == nil {
		println("Error in getting voice state, user is not in a voice channel")
		return errors.New("user is not in a voice channel")
	}

	// Join the voice channel
	conn := client.VoiceManager.CreateConn(guildID)
	if err = conn.Open(ctx, *voiceState.ChannelID, false, true); err != nil {
		println("Failed to join a voice channel, ", err.Error())
		return err
	}

	return nil
}

func leaveVoiceChannel(ctx context.Context, client *bot.Client, guildID snowflake.ID) {
	// Check the bot voice connection
	conn := client.VoiceManager.GetConn(guildID)
	if conn != nil {
		// disgo's Conn.Close does not return an error.
		conn.Close(ctx)
	}
}

func checkJoinedVoiceChannel(client *bot.Client, guildID snowflake.ID) (isJoined bool) {
	conn := client.VoiceManager.GetConn(guildID)
	return conn != nil && conn.ChannelID() != nil
}
