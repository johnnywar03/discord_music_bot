package main

import (
	"encoding/json"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

type jsonBotConfig struct {
	APIKey           string
	CommandChannelId string
	PushCommand      bool
}

var botConfig jsonBotConfig

func main() {
	// Get executable directory
	executable, _ := os.Executable()
	thisFilePath := filepath.Dir(executable)
	// Open env.json file
	jsonByte, err := os.ReadFile(thisFilePath + "\\env.json")
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

	// Start discord bot
	err = client.Open()
	if err != nil {
		println("Error in opening connection, ", err.Error())
		return
	}

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
		return
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
		joinVoiceChannel(session, interactionCreatedEvent)
	case "leave":
		leaveVoiceChannel(session, interactionCreatedEvent)
	case "play":
		responseToInteraction(session, interactionCreatedEvent.Interaction, "Underdeveloping")
	case "search":
		responseToInteraction(session, interactionCreatedEvent.Interaction, "Underdeveloping")
	case "remove":
		responseToInteraction(session, interactionCreatedEvent.Interaction, "Underdeveloping")
	case "list":
		responseToInteraction(session, interactionCreatedEvent.Interaction, "Underdeveloping")
	default:
		println("Received an unknown application command.")
		responseToInteraction(session, interactionCreatedEvent.Interaction, "Error: unknown command.")
	}
}

func responseToInteraction(session *discordgo.Session, interaction *discordgo.Interaction, content string, components ...discordgo.MessageComponent) {
	session.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Components: components,
		},
	})
}
