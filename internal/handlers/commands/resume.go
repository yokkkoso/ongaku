package commands

import (
	"context"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/yokkkoso/musicbot/internal/core"
	"github.com/yokkkoso/musicbot/internal/utils"
)

var resumeCommand = discord.SlashCommandCreate{
	Name:        "resume",
	Description: "Снять песню с паузы",
	Contexts:    []discord.InteractionContextType{discord.InteractionContextTypeGuild},
}

func HandleResumeSlashCommand(dj *core.DJ) handler.SlashCommandHandler {
	return func(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
		voiceState, ok := dj.Client.Caches.VoiceState(*e.GuildID(), e.User().ID)
		if !ok || voiceState.ChannelID == nil {
			return e.CreateMessage(
				discord.NewMessageCreate().
					WithEphemeral(true).
					AddEmbeds(
						utils.NewBaseEmbed().
							WithTitlef("Снять песню с паузы").
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
							WithTitlef("Снять песню с паузы").
							WithThumbnail(e.User().EffectiveAvatarURL()).
							WithDescriptionf(
								"%s, в Вашем голосовом канале нет бота. Используйте команды `/play` или `/search`",
								discord.UserMention(e.User().ID),
							),
					),
			)
		}

		player := node.LavalinkClient.ExistingPlayer(*e.GuildID())

		if !player.Paused() {
			return e.CreateMessage(
				discord.NewMessageCreate().
					WithEphemeral(true).
					AddEmbeds(
						utils.NewBaseEmbed().
							WithTitlef("Снять песню с паузы").
							WithThumbnail(e.User().EffectiveAvatarURL()).
							WithDescriptionf(
								"%s, песня **уже** воспроизводится",
								discord.UserMention(e.User().ID),
							),
					),
			)
		}

		if err := player.Update(context.TODO(), lavalink.WithPaused(false)); err != nil {
			return e.CreateMessage(
				discord.NewMessageCreate().
					WithEphemeral(true).
					AddEmbeds(
						utils.NewBaseEmbed().
							WithTitlef("Снять песню с паузы").
							WithThumbnail(e.User().EffectiveAvatarURL()).
							WithDescriptionf(
								"%s, не удалось снять песню с паузы",
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
