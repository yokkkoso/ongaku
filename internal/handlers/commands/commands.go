package commands

import "github.com/disgoorg/disgo/discord"

var Commands = []discord.ApplicationCommandCreate{
	playCommand,
	queueCommand,
	skipCommand,
	volumeCommand,
	seekCommand,
	pauseCommand,
	resumeCommand,
	nowPlayingCommand,
	repeatCommand,
	stopCommand,
	clearCommand,
	searchCommand,
}
