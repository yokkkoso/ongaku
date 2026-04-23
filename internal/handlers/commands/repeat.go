package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/yokkkoso/musicbot/internal/core"
	"github.com/yokkkoso/musicbot/internal/database"
	"github.com/yokkkoso/musicbot/internal/utils"
)

var repeatCommand = discord.SlashCommandCreate{
	Name:        "repeat",
	Description: "Изменить режим повторения",
	Contexts:    []discord.InteractionContextType{discord.InteractionContextTypeGuild},
	Options: []discord.ApplicationCommandOption{
		discord.ApplicationCommandOptionString{
			Name: "mode",
			NameLocalizations: map[discord.Locale]string{
				discord.LocaleRussian: "режим",
			},
			Description: "Режим повторения",
			Choices: []discord.ApplicationCommandOptionChoiceString{
				{
					Name:  "Выключить",
					Value: string(database.QueueTypeNormal),
				},
				{
					Name:  "Повторять всю очередь",
					Value: string(database.QueueTypeRepeatQueue),
				},
				{
					Name:  "Повторять одну песню",
					Value: string(database.QueueTypeRepeatTrack),
				},
			},
			Required: true,
		},
	},
}

func HandleRepeatSlashCommand(dj *core.DJ) handler.SlashCommandHandler {
	return func(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
		voiceState, ok := dj.Client.Caches.VoiceState(*e.GuildID(), e.User().ID)
		if !ok || voiceState.ChannelID == nil {
			return e.CreateMessage(
				discord.NewMessageCreate().
					WithEphemeral(true).
					AddEmbeds(
						utils.NewBaseEmbed().
							WithTitlef("Изменить режим повторения").
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
							WithTitlef("Изменить режим повторения").
							WithThumbnail(e.User().EffectiveAvatarURL()).
							WithDescriptionf(
								"%s, в Вашем голосовом канале нет бота. Используйте команды `/play` или `/search`",
								discord.UserMention(e.User().ID),
							),
					),
			)
		}

		_, _ = dj.Database.UpdateQueueType(node.ID(), *e.GuildID())

		messageData := generateNowPlayingMessageData(node, *e.GuildID())

		if messageData == nil {
			return nil
		}

		return e.CreateMessage(
			discord.NewMessageCreate().
				WithEphemeral(true).
				WithIsComponentsV2(true).
				WithComponents(messageData.Components...),
		)
	}
}
