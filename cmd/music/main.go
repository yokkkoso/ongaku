package main

import (
	"github.com/disgoorg/snowflake/v2"
	"github.com/yokkkoso/musicbot/internal/config_manager"
	"github.com/yokkkoso/musicbot/internal/core"
	"github.com/yokkkoso/musicbot/internal/handlers"
	"github.com/yokkkoso/musicbot/internal/utils/logger"
)

func main() {
	snowflake.AllowUnquoted = true

	logger.SetupLogger()

	config := config_manager.GetConfigManager().Get()

	dj := core.NewDJ()

	dj.InitDiscordClient(config.DJToken)
	dj.InitDatabase()

	for _, discordNodeConfig := range config.DiscordNodes {
		node := core.NewNode(dj)
		err := node.SetUpClients(discordNodeConfig, config.LavalinkNodes)

		if err != nil {
			continue
		}

		dj.AddNode(node)
	}

	handlers.SyncCommands(dj)
	handlers.InitHandlers(dj)

	dj.StartAndBlock()
}
