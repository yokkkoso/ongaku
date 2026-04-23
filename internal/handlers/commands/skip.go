package commands

import (
	"context"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/json"
	"github.com/yokkkoso/musicbot/internal/core"
	"github.com/yokkkoso/musicbot/internal/database"
	"github.com/yokkkoso/musicbot/internal/utils"
)

var skipCommand = discord.SlashCommandCreate{
	Name:        "skip",
	Description: "Пропустить песни",
	Contexts:    []discord.InteractionContextType{discord.InteractionContextTypeGuild},
	Options: []discord.ApplicationCommandOption{
		discord.ApplicationCommandOptionInt{
			Name: "amount",
			NameLocalizations: map[discord.Locale]string{
				discord.LocaleRussian: "количество",
			},
			Description: "Количество песен для пропуска",
			Required:    false,
		},
	},
}

func HandleSkipSlashCommand(dj *core.DJ) handler.SlashCommandHandler {
	return func(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
		voiceState, ok := dj.Client.Caches.VoiceState(*e.GuildID(), e.User().ID)
		if !ok || voiceState.ChannelID == nil {
			return e.CreateMessage(
				discord.NewMessageCreate().
					WithEphemeral(true).
					AddEmbeds(
						utils.NewBaseEmbed().
							WithTitlef("Пропустить песни").
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
							WithTitlef("Пропустить песни").
							WithThumbnail(e.User().EffectiveAvatarURL()).
							WithDescriptionf(
								"%s, в Вашем голосовом канале нет бота. Используйте команды `/play` или `/search`",
								discord.UserMention(e.User().ID),
							),
					),
			)
		}

		amount, ok := data.OptInt("amount")
		if !ok {
			amount = 1
		}

		queue, _ := dj.Database.GetQueue(*e.GuildID(), node.ID())
		track, err := dj.Database.SkipTracks(queue.ID, amount)

		if err != nil {
			return e.CreateMessage(
				discord.NewMessageCreate().
					WithEphemeral(true).
					AddEmbeds(
						utils.NewBaseEmbed().
							WithTitlef("Пропустить песни").
							WithThumbnail(e.User().EffectiveAvatarURL()).
							WithDescriptionf(
								"%s, в очереди нет больше песен",
								discord.UserMention(e.User().ID),
							),
					),
			)
		}

		var userData database.TrackUserData
		_ = json.Unmarshal([]byte(track.UserData), &userData)

		if err := node.LavalinkClient.ExistingPlayer(*e.GuildID()).Update(
			context.TODO(),
			lavalink.WithEncodedTrack(track.Encoded),
			lavalink.WithTrackUserData(userData),
		); err != nil {
			return e.CreateMessage(
				discord.NewMessageCreate().
					WithEphemeral(true).
					AddEmbeds(
						utils.NewBaseEmbed().
							WithTitlef("Пропустить песни").
							WithThumbnail(e.User().EffectiveAvatarURL()).
							WithDescriptionf(
								"%s, не удалось пропустить песню",
								discord.UserMention(e.User().ID),
							),
					),
			)
		}

		messageData := generateNowPlayingMessageData(node, *e.GuildID())

		return e.CreateMessage(
			discord.NewMessageCreate().
				WithEphemeral(true).
				WithIsComponentsV2(true).
				WithComponents(messageData.Components...),
		)
	}
}
