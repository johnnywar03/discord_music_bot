# Requirement
- Linux (tested on ubuntu)
- golang
- ffmpeg
- yt-dlp

# env.json
> This json file is config file of the discord bot
```JSON
{
    "APIKey": "",
    "commandChannelId": "",
    "pushCommand": true
}
```
# How to install
## golang
```bash
snap install go
```
## ffmpeg
```bash
snap install ffmpeg
```
## yt-dlp
```bash
snap install bash
```
## This repo
```bash
git clone github.com/johnnywar03/discord_music_bot
cd discord_music_bot
go build
./main
```
