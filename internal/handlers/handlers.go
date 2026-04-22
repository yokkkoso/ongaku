package handlers

import (
	"strings"

	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgo/handler/middleware"
	"github.com/disgoorg/snowflake/v2"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"gitlab.com/yokkkoso/musicbot/internal/core"
	"gitlab.com/yokkkoso/musicbot/internal/handlers/commands"
	"gitlab.com/yokkkoso/musicbot/internal/utils"
)

func SyncCommands(dj *core.DJ) {
	if viper.GetBool("sync_commands") {
		if err := handler.SyncCommands(
			dj.Client,
			commands.Commands,
			utils.Ternary(viper.GetBool("is_dev"), []snowflake.ID{965039675171541002}, []snowflake.ID{}),
		); err != nil {
			log.Err(err).Msg("Failed to sync commands")
		}
	}
}

func InitHandlers(dj *core.DJ) {
	r := handler.New()
	r.Use(middleware.Go)

	r.Error(
		func(e *handler.InteractionEvent, err error) {
			if !strings.Contains(strings.ToLower(err.Error()), "unknown interaction") {
				log.Err(err).Msg("Failed to run interaction handler")
			}
		},
	)

	r.SlashCommand("/play", commands.HandlePlaySlashCommand(dj))
	r.SlashCommand("/queue", commands.HandleQueueSlashCommand(dj))
	r.SlashCommand("/skip", commands.HandleSkipSlashCommand(dj))
	r.SlashCommand("/volume", commands.HandleVolumeSlashCommand(dj))
	r.SlashCommand("/seek", commands.HandleSeekSlashCommand(dj))
	r.SlashCommand("/pause", commands.HandlePauseSlashCommand(dj))
	r.SlashCommand("/resume", commands.HandleResumeSlashCommand(dj))
	r.SlashCommand("/nowplaying", commands.HandleNowPlayingSlashCommand(dj))
	r.SlashCommand("/stop", commands.HandleStopSlashCommand(dj))
	r.SlashCommand("/repeat", commands.HandleRepeatSlashCommand(dj))
	r.SlashCommand("/clear", commands.HandleClearSlashCommand(dj))
	r.SlashCommand("/search", commands.HandleSearchSlashCommand(dj))

	r.ButtonComponent("/searchResult/{uri}", commands.HandleSearchResultButton(dj))

	r.ButtonComponent("/queueDelete/{identifier}", commands.HandleQueueRemoveElementButton(dj))
	r.ButtonComponent("/queueRefresh", commands.HandleQueueRefreshButton(dj))

	r.Route(
		"/player", func(r handler.Router) {
			r.ButtonComponent("/pause", commands.HandlePlayerPauseButton(dj))
			r.ButtonComponent("/next", commands.HandlePlayerNextButton(dj))
			r.ButtonComponent("/repeat", commands.HandlePlayerRepeatButton(dj))
			r.ButtonComponent("/refresh", commands.HandlePlayerRefreshButton(dj))
			r.ButtonComponent("/stop", commands.HandlePlayerStopButton(dj))
			r.ButtonComponent("/volume/set", commands.HandlePlayerVolumeButton(dj))
			r.Modal("/volume/setModal", commands.HandlePlayerVolumeModal(dj))
		},
	)

	r.Route(
		"/pagination/{expTime}/{executorID}/{pageAction}/{currentPage}/{totalPages}",
		func(r handler.Router) {
			r.Use(IsByExecutor, HasExpiredTime)

			r.ButtonComponent("/queue", commands.HandleQueuePaginationButton(dj))
		},
	)

	dj.Client.AddEventListeners(r)
}
