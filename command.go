package main

import (
	"github.com/bwmarrin/discordgo"
)

var commands = []*discordgo.ApplicationCommand{

	{
		Name:        "help",
		Description: "Help for how to use this bot",
	},

	{
		Name:        "join",
		Description: "Join voice channel",
	},

	{
		Name:        "search",
		Description: "Search video from youtube",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "name",
				Description: "Name of the video",
				Required:    true,
			},
		},
	},

	{
		Name:        "play",
		Description: "Play video from youtube",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "url",
				Description: "URL of the video",
				Required:    true,
			},
		},
	},

	{
		Name:        "leave",
		Description: "Leave voice channel",
	},

	{
		Name:        "remove",
		Description: "Remove music in the queue",
	},

	{
		Name:        "clear",
		Description: "Remove all music in the queue",
	},

	{
		Name:        "list",
		Description: "List music queue",
	},

	{
		Name:        "skip",
		Description: "Skip the music",
	},
}

func registerCommand(client *discordgo.Session, pushCommand bool) (err error) {
	if !pushCommand {
		println("Skipping register commands...")
		return err
	} else if pushCommand {
		println("Registering commands...")
		_, err := client.ApplicationCommandBulkOverwrite(client.State.Application.ID, "", commands)
		return err
	}
	return
}
