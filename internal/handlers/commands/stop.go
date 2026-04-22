package commands

import (
	"context"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"gitlab.com/yokkkoso/musicbot/internal/core"
	"gitlab.com/yokkkoso/musicbot/internal/utils"
)

var stopCommand = discord.SlashCommandCreate{
	Name:        "stop",
	Description: "Очистить очередь и отключить бота",
	Contexts:    []discord.InteractionContextType{discord.InteractionContextTypeGuild},
}

func HandleStopSlashCommand(dj *core.DJ) handler.SlashCommandHandler {
	return func(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
		voiceState, ok := dj.Client.Caches.VoiceState(*e.GuildID(), e.User().ID)
		if !ok || voiceState.ChannelID == nil {
			return e.CreateMessage(
				discord.NewMessageCreate().
					WithEphemeral(true).
					AddEmbeds(
						utils.NewBaseEmbed().
							WithTitlef("Очистить очередь и отключить бота").
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
							WithTitlef("Очистить очередь и отключить бота").
							WithThumbnail(e.User().EffectiveAvatarURL()).
							WithDescriptionf(
								"%s, в Вашем голосовом канале нет бота. Используйте команды `/play` или `/search`",
								discord.UserMention(e.User().ID),
							),
					),
			)
		}

		_ = node.DiscordClient.UpdateVoiceState(context.TODO(), *e.GuildID(), nil, false, true)

		return e.CreateMessage(
			discord.NewMessageCreate().
				WithEphemeral(true).
				AddEmbeds(
					utils.NewBaseEmbed().
						WithTitlef("Очистить очередь и отключить бота").
						WithThumbnail(e.User().EffectiveAvatarURL()).
						WithDescriptionf(
							"%s, бот успешно отключен",
							discord.UserMention(e.User().ID),
						),
				),
		)
	}
}
