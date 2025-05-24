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
		panic(err)
	}
	err = json.Unmarshal(jsonByte, &botConfig)
	if err != nil {
		panic(err)
	}

	// Create discord bot session
	client, err := discordgo.New("Bot " + botConfig.APIKey)
	if err != nil {
		panic(err)
	}

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
		panic(err)
	}

	// Press CTRL-C to close discord bot
	println(client.State.User.Username + " is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	client.Close()
}
