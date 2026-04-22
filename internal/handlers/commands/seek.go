package commands

import (
	"context"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"gitlab.com/yokkkoso/musicbot/internal/core"
	"gitlab.com/yokkkoso/musicbot/internal/utils"
)

var seekCommand = discord.SlashCommandCreate{
	Name:        "seek",
	Description: "Промотать вперед песню",
	Contexts:    []discord.InteractionContextType{discord.InteractionContextTypeGuild},
	Options: []discord.ApplicationCommandOption{
		discord.ApplicationCommandOptionString{
			Name: "time",
			NameLocalizations: map[discord.Locale]string{
				discord.LocaleRussian: "время",
			},
			Description: "На сколько нужно проматать песню (например, 1m21s)",
			Required:    true,
		},
	},
}

func HandleSeekSlashCommand(dj *core.DJ) handler.SlashCommandHandler {
	return func(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
		voiceState, ok := dj.Client.Caches.VoiceState(*e.GuildID(), e.User().ID)
		if !ok || voiceState.ChannelID == nil {
			return e.CreateMessage(
				discord.NewMessageCreate().
					WithEphemeral(true).
					AddEmbeds(
						utils.NewBaseEmbed().
							WithTitlef("Промотать вперед песню").
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
							WithTitlef("Промотать вперед песню").
							WithThumbnail(e.User().EffectiveAvatarURL()).
							WithDescriptionf(
								"%s, в Вашем голосовом канале нет бота. Используйте команды `/play` или `/search`",
								discord.UserMention(e.User().ID),
							),
					),
			)
		}

		duration, err := time.ParseDuration(data.String("time"))
		if err != nil {
			return e.CreateMessage(
				discord.NewMessageCreate().
					WithEphemeral(true).
					AddEmbeds(
						utils.NewBaseEmbed().
							WithTitlef("Промотать вперед песню").
							WithThumbnail(e.User().EffectiveAvatarURL()).
							WithDescriptionf(
								"%s, Вы должны ввести время в специальном формате, например, 1m21s, что перемотает песню на 1 минуту и 21 секунду вперед",
								discord.UserMention(e.User().ID),
							),
					),
			)
		}

		player := node.LavalinkClient.Player(*e.GuildID())

		nowPlaying := player.Track()

		if nowPlaying == nil {
			return e.CreateMessage(
				discord.NewMessageCreate().
					WithEphemeral(true).
					AddEmbeds(
						utils.NewBaseEmbed().
							WithTitlef("Промотать вперед песню").
							WithThumbnail(e.User().EffectiveAvatarURL()).
							WithDescriptionf(
								"%s, сейчас ничего не играет",
								discord.UserMention(e.User().ID),
							),
					),
			)
		}

		var position lavalink.Duration

		if nowPlaying.Info.Length.Milliseconds() < duration.Milliseconds() {
			position = nowPlaying.Info.Length
		} else {
			position = lavalink.Duration(player.Position().Milliseconds() + duration.Milliseconds())
		}

		if err := player.Update(context.TODO(), lavalink.WithPosition(position)); err != nil {
			return e.CreateMessage(
				discord.NewMessageCreate().
					WithEphemeral(true).
					AddEmbeds(
						utils.NewBaseEmbed().
							WithTitlef("Промотать вперед песню").
							WithThumbnail(e.User().EffectiveAvatarURL()).
							WithDescriptionf(
								"%s, не удалось промотать вперед песню",
								discord.UserMention(e.User().ID),
							),
					),
			)
		}

		return e.CreateMessage(
			discord.NewMessageCreate().
				WithEphemeral(true).
				AddEmbeds(
					utils.NewBaseEmbed().
						WithTitlef("Промотать вперед песню").
						WithThumbnail(e.User().EffectiveAvatarURL()).
						WithDescriptionf(
							"%s, песня промотана на **%s**",
							discord.UserMention(e.User().ID),
							utils.FormatDuration(position),
						),
				),
		)
	}
}
