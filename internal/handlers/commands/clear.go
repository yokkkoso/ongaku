package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/yokkkoso/ongaku/internal/core"
	"github.com/yokkkoso/ongaku/internal/utils"
)

var clearCommand = discord.SlashCommandCreate{
	Name:        "clear",
	Description: "Очистить очередь песен",
	Contexts:    []discord.InteractionContextType{discord.InteractionContextTypeGuild},
}

func HandleClearSlashCommand(dj *core.DJ) handler.SlashCommandHandler {
	return func(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
		voiceState, ok := dj.Client.Caches.VoiceState(*e.GuildID(), e.User().ID)
		if !ok || voiceState.ChannelID == nil {
			return e.CreateMessage(
				discord.NewMessageCreate().
					WithEphemeral(true).
					AddEmbeds(
						utils.NewBaseEmbed().
							WithTitlef("Очистить очередь песен").
							WithThumbnail(e.User().EffectiveAvatarURL()).
							WithDescriptionf(
								"%s, Вы должны быть в голосовом канале для использования этой команды",
								discord.UserMention(e.User().ID),
							),
					),
			)
		}

		node := dj.GetNode(*e.GuildID(), *voiceState.ChannelID)

		if node == nil {
			return e.CreateMessage(
				discord.NewMessageCreate().
					WithEphemeral(true).
					AddEmbeds(
						utils.NewBaseEmbed().
							WithTitlef("Очистить очередь песен").
							WithThumbnail(e.User().EffectiveAvatarURL()).
							WithDescriptionf(
								"%s, в Вашем голосовом канале нет бота. Используйте команды `/play` или `/search`",
								discord.UserMention(e.User().ID),
							),
					),
			)
		}

		_ = dj.Database.DeleteQueueByNodeAndGuild(node.ID(), *e.GuildID())

		return e.CreateMessage(
			discord.NewMessageCreate().
				WithEphemeral(true).
				AddEmbeds(
					utils.NewBaseEmbed().
						WithTitlef("Очистить очередь песен").
						WithThumbnail(e.User().EffectiveAvatarURL()).
						WithDescriptionf(
							"%s, очередь успешно очищена",
							discord.UserMention(e.User().ID),
						),
				),
		)
	}
}
