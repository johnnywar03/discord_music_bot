package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"syscall"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/godave/golibdave"
	"github.com/disgoorg/snowflake/v2"
)

type jsonBotConfig struct {
	APIKey           string
	CommandChannelId snowflake.ID
	PushCommand      bool
}

var botConfig jsonBotConfig
var thisFilePath string
var videoQueue *VideoQueue
var musicBot *MusicBot
var client *bot.Client

func main() {
	// Get executable directory
	executable, _ := os.Executable()
	thisFilePath = filepath.Dir(executable)
	// Open env.json file
	jsonByte, err := os.ReadFile(thisFilePath + "/env.json")
	if err != nil {
		panic(err.Error())
	}
	err = json.Unmarshal(jsonByte, &botConfig)
	if err != nil {
		panic(err.Error())
	}

	// Create video queue
	videoQueue = &VideoQueue{}

	// Create music bot instance
	musicBot = NewMusicBot()

	// Create discord bot client
	client, err = disgo.New(botConfig.APIKey,
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(gateway.IntentsNonPrivileged),
		),
		// VoiceStates need to be cached so joinVoiceChannel can look up which
		// channel the command issuer is currently in.
		bot.WithCacheConfigOpts(
			cache.WithCaches(cache.FlagVoiceStates),
		),
		// Discord now requires the E2EE/DAVE protocol for voice connections.
		// The noop DAVE session (disgo's default) can no longer complete the
		// voice gateway handshake, so we wire in the real libdave-backed
		// implementation from disgoorg/godave instead. This requires CGO to
		// be enabled and the libdave native library to be installed at build
		// time (see github.com/disgoorg/godave's libdave_install script).
		bot.WithVoiceManagerConfigOpts(
			voice.WithDaveSessionCreateFunc(golibdave.NewSession),
		),
		bot.WithEventListenerFunc(onReady),
		bot.WithEventListenerFunc(onApplicationCommand),
		bot.WithEventListenerFunc(onComponentInteraction),
	)
	if err != nil {
		panic(err.Error())
	}
	defer client.Close(context.Background())

	// Register slash commands
	err = registerCommand(client, botConfig.PushCommand)
	if err != nil {
		panic(err.Error())
	}

	// Start discord bot
	err = client.OpenGateway(context.Background())
	if err != nil {
		println("Error in opening connection, ", err.Error())
		return
	}

	// Press CTRL-C to close discord bot
	println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

func onReady(event *events.Ready) {
	println(event.User.Username + " is ready.")
}

func onApplicationCommand(event *events.ApplicationCommandInteractionCreate) {
	// Specify a channel for communicate
	if event.Channel().ID() != botConfig.CommandChannelId {
		_ = event.CreateMessage(discord.NewMessageCreate().WithContent("Wrong channel! Please use specific channel!"))
		return
	}

	data := event.SlashCommandInteractionData()
	// A switch case to handle commands
	switch data.CommandName() {
	case "help":
		// Get discord bot owner info
		app, err := client.Rest.GetCurrentApplication()
		if err != nil {
			println("Error in getting owner info, ", err.Error())
			return
		}
		// Response to the application command
		_ = event.CreateMessage(discord.NewMessageCreate().WithContent("Ask " + app.Owner.Username))
	case "join":
		_ = event.CreateMessage(discord.NewMessageCreate().WithContent("Joining..."))
		guildID := *event.GuildID()
		userID := event.User().ID
		go func() {
			err := joinVoiceChannel(context.Background(), client, guildID, userID)
			if err != nil {
				updateInteractionResponse(event, "Failed to join the voice channel.")
				return
			}
			updateInteractionResponse(event, "Joined!")
			musicBot.playVideo(context.Background(), client, guildID)
		}()
	case "leave":
		_ = event.CreateMessage(discord.NewMessageCreate().WithContent("Processing..."))
		musicBot.stop()
		if err := remove(thisFilePath + "/video"); err != nil {
			slog.Warn("failed to clean up video directory", "error", err)
		}
		updateInteractionResponse(event, "Bye!")
	case "play":
		guildID := *event.GuildID()
		_ = event.CreateMessage(discord.NewMessageCreate().WithContent("Processing..."))
		userID := event.User().ID
		go func() {
			err := joinVoiceChannel(context.Background(), client, guildID, userID)
			if err != nil {
				updateInteractionResponse(event, "Failed to join the voice channel.")
				return
			}
			regex := regexp.MustCompile(`(?:youtube\.com\/(?:[^\/]+\/.+\/|(?:v|e(?:mbed)?)\/|.*[?&]v=)|youtu\.be\/)([^"&?\/\s]{11})`)
			videoId := regex.FindStringSubmatch(data.String("url"))[1]
			title, err := videoQueue.add(videoId)
			if err != nil {
				updateInteractionResponse(event, err.Error())
				return
			}
			updateInteractionResponse(event, fmt.Sprintf("%s added to queue.", title))
			musicBot.playVideo(context.Background(), client, guildID)
		}()
	case "search":
		_ = event.CreateMessage(discord.NewMessageCreate().WithContent("Processing..."))
		// search() shells out to yt-dlp to query YouTube, which can block for a
		// long time. Run it off the gateway goroutine so heartbeat ACKs keep
		// being processed.
		go func() {
			videoArray, err := search(data.String("name"))
			if err != nil {
				updateInteractionResponse(event, "Error: cannot search.")
				return
			}

			// Convert array to discord select menu options
			var options []discord.StringSelectMenuOption
			for _, video := range videoArray {
				options = append(options, discord.NewStringSelectMenuOption(video.Title, video.Id))
			}
			// Add cancel option to the select menu options
			options = addCancelOption(options)
			selectMenu := discord.NewStringSelectMenu("search", "Select a video", options...)

			updateInteractionResponseWithMenu(event, "Please select a video", selectMenu)
		}()
	case "remove":
		_ = event.CreateMessage(discord.NewMessageCreate().WithContent("Processing..."))
		// Convert linked list to array
		videoArray, err := videoQueue.toArray()
		if err != nil {
			updateInteractionResponse(event, "The queue is empty.")
			return
		}

		// Convert array to discord select menu options, skip the first element
		var options []discord.StringSelectMenuOption
		for _, video := range videoArray[1:] {
			options = append(options, discord.NewStringSelectMenuOption(video.Title, video.Id))
		}
		// Add cancel option to the select menu options
		options = addCancelOption(options)
		selectMenu := discord.NewStringSelectMenu("remove", "Select a video", options...)

		updateInteractionResponseWithMenu(event, "Please select a video", selectMenu)
	case "list":
		// Response to the interaction first in within 3 second
		_ = event.CreateMessage(discord.NewMessageCreate().WithContent("Processing..."))

		// Implement queue system first
		listOfVideo := videoQueue.list()
		if listOfVideo == "" {
			updateInteractionResponse(event, "The queue is empty.")
			return
		}
		// Update the interaction
		updateInteractionResponse(event, listOfVideo)
	case "clear":
		musicBot.stop()
		if err := remove(thisFilePath + "/video"); err != nil {
			slog.Warn("failed to clean up video directory", "error", err)
		}
		_ = event.CreateMessage(discord.NewMessageCreate().WithContent("Queue cleared"))
	case "skip":
		// If music bot is not playing
		nowPlaying := musicBot.NowPlaying()
		if !musicBot.IsPlaying() || nowPlaying == nil {
			_ = event.CreateMessage(discord.NewMessageCreate().WithContent("No video is currently playing."))
			return
		}
		_ = event.CreateMessage(discord.NewMessageCreate().WithContent("Skipping..."))
		// Skip the video
		musicBot.skip()
	case "nowplaying":
		nowPlaying := musicBot.NowPlaying()
		if nowPlaying == nil {
			_ = event.CreateMessage(discord.NewMessageCreate().WithContent("No video is currently playing."))
			return
		}
		_ = event.CreateMessage(discord.NewMessageCreate().WithContentf("Now Playing:\n%s", nowPlaying.Title))
	default:
		println("Received an unknown application command.")
		_ = event.CreateMessage(discord.NewMessageCreate().WithContent("Error: unknown command."))
	}
}

func onComponentInteraction(event *events.ComponentInteractionCreate) {
	// Specify a channel for communicate
	if event.Channel().ID() != botConfig.CommandChannelId {
		_ = event.CreateMessage(discord.NewMessageCreate().WithContent("Wrong channel! Please use specific channel!"))
		return
	}

	// Use switch case to handle different command
	switch event.Data.CustomID() {
	case "remove":
		data := event.StringSelectMenuInteractionData()
		// Handle cancel option
		if data.Values[0] == "cancel" {
			_ = event.UpdateMessage(discord.NewMessageUpdate().WithContent("Action cancel.").ClearComponents())
			return
		}
		_ = event.UpdateMessage(discord.NewMessageUpdate().WithContent("Processing...").ClearComponents())
		id := data.Values[0]
		// Delete video from queue
		title, err := videoQueue.deleteSpecific(id)
		if err != nil {
			updateComponentInteractionResponse(event, "Error: failed to delete video from the queue.")
			return
		}
		updateComponentInteractionResponse(event, title+" removed from the queue.")
	case "search":
		data := event.StringSelectMenuInteractionData()
		// Handle cancel option
		if data.Values[0] == "cancel" {
			_ = event.UpdateMessage(discord.NewMessageUpdate().WithContent("Action cancel.").ClearComponents())
			return
		}
		_ = event.UpdateMessage(discord.NewMessageUpdate().WithContent("Processing...").ClearComponents())
		id := data.Values[0]
		// videoQueue.add() shells out to yt-dlp to fetch the title, which can
		// block for a long time. Run the rest of this handler off the gateway
		// goroutine so heartbeat ACKs keep being processed.
		go func() {
			// Add video to the queue
			title, err := videoQueue.add(id)
			if err != nil {
				updateComponentInteractionResponse(event, "Error: failed to add video to the queue.")
				return
			}
			updateComponentInteractionResponse(event, title+" added to queue.")
			guildID := *event.GuildID()

			err = joinVoiceChannel(context.Background(), client, guildID, event.User().ID)
			if err != nil {
				sendMessageToChannel(client, "Failed to join the voice channel.")
				return
			}
			musicBot.playVideo(context.Background(), client, guildID)
		}()
	default:
		println("Received an unknown interaction component.")
		_ = event.UpdateMessage(discord.NewMessageUpdate().WithContent("Error: unknown interaction.").ClearComponents())
	}
}

func sendMessageToChannel(client *bot.Client, content string) {
	_, _ = client.Rest.CreateMessage(botConfig.CommandChannelId, discord.NewMessageCreate().WithContent(content))
}

// updateInteractionResponse edits the original response to an application
// command interaction. This is the equivalent of discordgo's
// InteractionResponseEdit and is used after the initial "Processing..." ack.
func updateInteractionResponse(event *events.ApplicationCommandInteractionCreate, content string) {
	_, _ = client.Rest.UpdateInteractionResponse(client.ApplicationID, event.Token(), discord.NewMessageUpdate().WithContent(content))
}

func updateInteractionResponseWithMenu(event *events.ApplicationCommandInteractionCreate, content string, selectMenu discord.StringSelectMenuComponent) {
	_, _ = client.Rest.UpdateInteractionResponse(client.ApplicationID, event.Token(), discord.NewMessageUpdate().WithContent(content).AddActionRow(selectMenu))
}

func updateComponentInteractionResponse(event *events.ComponentInteractionCreate, content string) {
	_, _ = client.Rest.UpdateInteractionResponse(client.ApplicationID, event.Token(), discord.NewMessageUpdate().WithContent(content).ClearComponents())
}

func addCancelOption(options []discord.StringSelectMenuOption) []discord.StringSelectMenuOption {
	return append(options, discord.NewStringSelectMenuOption("Cancel", "cancel"))
}
