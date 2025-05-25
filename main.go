package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

type jsonBotConfig struct {
	APIKey           string
	CommandChannelId string
	PushCommand      bool
}

var botConfig jsonBotConfig
var thisFilePath string
var videoQueue *VideoQueue
var musicBot *MusicBot

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

	// Create discord bot session
	client, err := discordgo.New("Bot " + botConfig.APIKey)
	if err != nil {
		panic(err.Error())
	}

	// Add application command handler
	client.AddHandler(interactionHandler)

	// Set discord bot intents
	client.Identify.Intents = discordgo.IntentsAllWithoutPrivileged

	// Create video queue
	videoQueue = &VideoQueue{}

	// Create music bot instance
	musicBot = &MusicBot{}
	musicBot.IsPlaying = false
	musicBot.NowPlaying = nil

	// Start discord bot
	err = client.Open()
	if err != nil {
		println("Error in opening connection, ", err.Error())
		return
	}
	defer client.Close()

	// Register slash commands
	err = registerCommand(client, botConfig.PushCommand)
	if err != nil {
		panic(err.Error())
	}

	// Press CTRL-C to close discord bot
	println(client.State.User.Username + " is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	client.Close()
}

func interactionHandler(session *discordgo.Session, interactionCreatedEvent *discordgo.InteractionCreate) {
	// Specify a channel for communicate
	if interactionCreatedEvent.ChannelID != botConfig.CommandChannelId {
		session.InteractionRespond(interactionCreatedEvent.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "Wrong channel! Please use specific channel!"},
		})
		return
	}

	switch interactionCreatedEvent.Type {
	case discordgo.InteractionApplicationCommand:
		applicationCommandHandler(session, interactionCreatedEvent)
	case discordgo.InteractionMessageComponent:
		interactionComponentHandler(session, interactionCreatedEvent)
	default:
		panic("Received an unknow CreatedInterationEvent type.")
	}
}

func applicationCommandHandler(session *discordgo.Session, interactionCreatedEvent *discordgo.InteractionCreate) {
	// Get the application command data
	applicationCommandData := interactionCreatedEvent.ApplicationCommandData()
	// A switch case to handle commands
	switch applicationCommandData.Name {
	case "help":
		// Get discord bot owner info
		app, err := session.Application("@me")
		if err != nil {
			println("Error in getting owner info, ", err.Error())
			return
		}
		// Response to the application command
		responseToInteraction(session, interactionCreatedEvent.Interaction, "Ask "+app.Owner.Username)
	case "join":
		err := joinVoiceChannel(session, interactionCreatedEvent)
		if err != nil {
			responseToInteraction(session, interactionCreatedEvent.Interaction, "Failed to join the voice channel.")
			return
		}
		playVideo(session, interactionCreatedEvent.GuildID)
	case "leave":
		leaveVoiceChannel(session, interactionCreatedEvent)
	case "play":
		// Join the voice channel if not connected to voice channel
		err := joinVoiceChannel(session, interactionCreatedEvent)
		if err != nil {
			responseToInteraction(session, interactionCreatedEvent.Interaction, "Failed to join the voice channel.")
			return
		}

		// Response to the interaction first in within 3 second
		responseToInteraction(session, interactionCreatedEvent.Interaction, "Processing...")

		// Extract video id from url
		regex := regexp.MustCompile(`(?:youtube\.com\/(?:[^\/]+\/.+\/|(?:v|e(?:mbed)?)\/|.*[?&]v=)|youtu\.be\/)([^"&?\/\s]{11})`)
		videoId := regex.FindStringSubmatch(interactionCreatedEvent.ApplicationCommandData().Options[0].StringValue())[1]

		// Add video to the queue
		title, err := videoQueue.add(videoId)
		if err != nil {
			responseToInteraction(session, interactionCreatedEvent.Interaction, err.Error())
			return
		}
		// Update the interaction
		updateInteractionResponse(session, interactionCreatedEvent.Interaction, fmt.Sprintf("%s %s", title, " added to queue."))
		playVideo(session, interactionCreatedEvent.GuildID)
	case "search":
		responseToInteraction(session, interactionCreatedEvent.Interaction, "Processing...")
		// Search for videos
		videoArray, err := search(interactionCreatedEvent.ApplicationCommandData().Options[0].StringValue())
		if err != nil {
			updateInteractionResponse(session, interactionCreatedEvent.Interaction, "Error: cannot search.")
			return
		}

		// Convert array to discord select menu options
		var options []discordgo.SelectMenuOption
		for _, video := range videoArray {
			options = append(options, discordgo.SelectMenuOption{
				Label: video.Title,
				Value: video.Id,
			})
		}
		// Add cancel buttion/option to the select menu options
		options = addCancelOption(options)
		selectMenu := &discordgo.SelectMenu{
			CustomID:    "search",
			Placeholder: "Select a video",
			Options:     options,
		}
		// Warp select menu to the actions row
		components := discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{selectMenu},
		}

		updateInteractionResponse(session, interactionCreatedEvent.Interaction, "Please select a video", components)
	case "remove":
		responseToInteraction(session, interactionCreatedEvent.Interaction, "Processing...")
		// Convert linked list to array
		videoArray, err := videoQueue.toArray()
		if err != nil {
			updateInteractionResponse(session, interactionCreatedEvent.Interaction, "The queue is empty.")
			return
		}

		// Convert array to discord select menu options, skip the first element
		var options []discordgo.SelectMenuOption
		for _, video := range videoArray[1:] {
			options = append(options, discordgo.SelectMenuOption{
				Label: video.Title,
				Value: video.Id,
			})
		}
		// Add cancel buttion/option to the select menu options
		options = addCancelOption(options)
		selectMenu := &discordgo.SelectMenu{
			CustomID:    "remove",
			Placeholder: "Select a video",
			Options:     options,
		}
		// Warp select menu to the actions row
		components := discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{selectMenu},
		}

		updateInteractionResponse(session, interactionCreatedEvent.Interaction, "Please select a video", components)
	case "list":
		// Response to the interaction first in within 3 second
		responseToInteraction(session, interactionCreatedEvent.Interaction, "Processing...")

		// Implement queue system first
		listOfVideo := videoQueue.list()
		if listOfVideo == "" {
			updateInteractionResponse(session, interactionCreatedEvent.Interaction, "The queue is empty.")
			return
		}
		// Update the interaction
		updateInteractionResponse(session, interactionCreatedEvent.Interaction, listOfVideo)
	case "clear":
		videoQueue.CurrentVideo = nil
		responseToInteraction(session, interactionCreatedEvent.Interaction, "Queue cleared")
	case "skip":
		// Wait to rewrite
		title := videoQueue.CurrentVideo.Title
		videoQueue.deleteFirst()
		responseToInteraction(session, interactionCreatedEvent.Interaction, "Skipped "+title)
	default:
		println("Received an unknown application command.")
		responseToInteraction(session, interactionCreatedEvent.Interaction, "Error: unknown command.")
	}
}

func interactionComponentHandler(session *discordgo.Session, interactionCreatedEvent *discordgo.InteractionCreate) {
	// Get interaction component data
	componentData := interactionCreatedEvent.MessageComponentData()
	// Use switch case to handle different command
	switch componentData.CustomID {
	case "remove":
		// Handle cancel option
		if componentData.Values[0] == "cancel" {
			updateComponentReponse(session, interactionCreatedEvent.Interaction, "Action cancel.")
			return
		}
		updateComponentReponse(session, interactionCreatedEvent.Interaction, "Processing...")
		id := componentData.Values[0]
		// Delete video from queue
		title, err := videoQueue.deleteSpecific(id)
		if err != nil {
			updateComponentReponse(session, interactionCreatedEvent.Interaction, "Error: failed to delete video from the queue.")
			return
		}
		updateInteractionResponse(session, interactionCreatedEvent.Interaction, title+" removed from the queue.")
	case "search":
		// Handle cancel option
		if componentData.Values[0] == "cancel" {
			updateComponentReponse(session, interactionCreatedEvent.Interaction, "Action cancel.")
			return
		}
		updateComponentReponse(session, interactionCreatedEvent.Interaction, "Processing...")
		id := componentData.Values[0]
		// Add video to the queue
		title, err := videoQueue.add(id)
		if err != nil {
			updateComponentReponse(session, interactionCreatedEvent.Interaction, "Error: failed to add video to the queue.")
			return
		}
		updateInteractionResponse(session, interactionCreatedEvent.Interaction, title+" added to queue.")
		err = joinVoiceChannel(session, interactionCreatedEvent)
		if err != nil {
			sendMessageToChannel(session, "Failed to join the voice channel.")
			return
		}
		playVideo(session, interactionCreatedEvent.GuildID)
	default:
		println("Received an unknown interaction component.")
		updateComponentReponse(session, interactionCreatedEvent.Interaction, "Error: unknown interaction.")
	}
}

func sendMessageToChannel(session *discordgo.Session, content string) {
	session.ChannelMessageSend(botConfig.CommandChannelId, content)
}

func responseToInteraction(session *discordgo.Session, interaction *discordgo.Interaction, content string) {
	session.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
}

func updateInteractionResponse(session *discordgo.Session, interaction *discordgo.Interaction, content string, components ...discordgo.MessageComponent) {
	session.InteractionResponseEdit(interaction, &discordgo.WebhookEdit{
		Content:    &content,
		Components: &components,
	})
}

func updateComponentReponse(session *discordgo.Session, interaction *discordgo.Interaction, content string) {
	session.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
}

func addCancelOption(option []discordgo.SelectMenuOption) (options []discordgo.SelectMenuOption) {
	return append(option, discordgo.SelectMenuOption{
		Label: "Cancel",
		Value: "cancel",
	})
}
