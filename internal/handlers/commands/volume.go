package commands

import (
	"context"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"gitlab.com/yokkkoso/musicbot/internal/core"
	"gitlab.com/yokkkoso/musicbot/internal/utils"
)

var volumeCommand = discord.SlashCommandCreate{
	Name:        "volume",
	Description: "Изменить громкость",
	Contexts:    []discord.InteractionContextType{discord.InteractionContextTypeGuild},
	Options: []discord.ApplicationCommandOption{
		discord.ApplicationCommandOptionInt{
			Name: "volume",
			NameLocalizations: map[discord.Locale]string{
				discord.LocaleRussian: "громкость",
			},
			Description: "Громкость в процентах",
			MaxValue:    new(125),
			MinValue:    new(0),
			Required:    true,
		},
	},
}

func HandleVolumeSlashCommand(dj *core.DJ) handler.SlashCommandHandler {
	return func(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
		voiceState, ok := dj.Client.Caches.VoiceState(*e.GuildID(), e.User().ID)
		if !ok || voiceState.ChannelID == nil {
			return e.CreateMessage(
				discord.NewMessageCreate().
					WithEphemeral(true).
					AddEmbeds(
						utils.NewBaseEmbed().
							WithTitlef("Изменить громкость").
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
							WithTitlef("Изменить громкость").
							WithThumbnail(e.User().EffectiveAvatarURL()).
							WithDescriptionf(
								"%s, в Вашем голосовом канале нет бота. Используйте команды `/play` или `/search`",
								discord.UserMention(e.User().ID),
							),
					),
			)
		}

		volume := data.Int("volume")

		if err := node.LavalinkClient.ExistingPlayer(*e.GuildID()).Update(context.TODO(), lavalink.WithVolume(volume)); err != nil {
			return e.CreateMessage(
				discord.NewMessageCreate().
					WithEphemeral(true).
					AddEmbeds(
						utils.NewBaseEmbed().
							WithTitlef("Изменить громкость").
							WithThumbnail(e.User().EffectiveAvatarURL()).
							WithDescriptionf(
								"%s, не удалось изменить громкость",
								discord.UserMention(e.User().ID),
							),
					),
			)
		}

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
