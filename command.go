package main

import (
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
)

var commands = []discord.ApplicationCommandCreate{

	discord.SlashCommandCreate{
		Name:        "help",
		Description: "Help for how to use this bot",
	},

	discord.SlashCommandCreate{
		Name:        "join",
		Description: "Join voice channel",
	},

	discord.SlashCommandCreate{
		Name:        "search",
		Description: "Search video from youtube",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{
				Name:        "name",
				Description: "Name of the video",
				Required:    true,
			},
		},
	},

	discord.SlashCommandCreate{
		Name:        "play",
		Description: "Play video from youtube",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{
				Name:        "url",
				Description: "URL of the video",
				Required:    true,
			},
		},
	},

	discord.SlashCommandCreate{
		Name:        "leave",
		Description: "Leave voice channel",
	},

	discord.SlashCommandCreate{
		Name:        "remove",
		Description: "Remove music in the queue",
	},

	discord.SlashCommandCreate{
		Name:        "clear",
		Description: "Remove all music in the queue",
	},

	discord.SlashCommandCreate{
		Name:        "list",
		Description: "List music queue",
	},

	discord.SlashCommandCreate{
		Name:        "skip",
		Description: "Skip the music",
	},

	discord.SlashCommandCreate{
		Name:        "nowplaying",
		Description: "Get now playing video",
	},
}

func registerCommand(client *bot.Client, pushCommand bool) (err error) {
	if !pushCommand {
		println("Skipping register commands...")
		return nil
	}
	println("Registering commands...")
	_, err = client.Rest.SetGlobalCommands(client.ApplicationID, commands)
	return err
}
