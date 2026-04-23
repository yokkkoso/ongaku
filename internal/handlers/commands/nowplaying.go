package commands

import (
	"context"
	"fmt"
	"strconv"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/json"
	"github.com/disgoorg/snowflake/v2"
	"github.com/yokkkoso/musicbot/internal/core"
	"github.com/yokkkoso/musicbot/internal/database"
	"github.com/yokkkoso/musicbot/internal/utils"
	"github.com/yokkkoso/musicbot/internal/utils/progress_bar"
)

var nowPlayingCommand = discord.SlashCommandCreate{
	Name:        "nowplaying",
	Description: "Посмотреть текущую песню",
	Contexts:    []discord.InteractionContextType{discord.InteractionContextTypeGuild},
}

type nowPlayingData struct {
	Embeds     []discord.Embed
	Components []discord.LayoutComponent
}

func HandleNowPlayingSlashCommand(dj *core.DJ) handler.SlashCommandHandler {
	return func(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
		voiceState, ok := dj.Client.Caches.VoiceState(*e.GuildID(), e.User().ID)
		if !ok || voiceState.ChannelID == nil {
			return e.CreateMessage(
				discord.NewMessageCreate().
					WithEphemeral(true).
					AddEmbeds(
						utils.NewBaseEmbed().
							WithTitlef("Посмотреть текущую песню").
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
							WithTitlef("Посмотреть текущую песню").
							WithThumbnail(e.User().EffectiveAvatarURL()).
							WithDescriptionf(
								"%s, в Вашем голосовом канале нет бота. Используйте команды `/play` или `/search`",
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

func HandlePlayerPauseButton(dj *core.DJ) handler.ButtonComponentHandler {
	return func(data discord.ButtonInteractionData, e *handler.ComponentEvent) error {
		voiceState, ok := dj.Client.Caches.VoiceState(*e.GuildID(), e.User().ID)
		if !ok || voiceState.ChannelID == nil {
			return e.UpdateMessage(
				discord.NewMessageUpdate().
					WithComponents(
						discord.NewContainer(
							discord.NewSection(
								discord.NewTextDisplay("### Поставить на паузу песню"),
								discord.NewTextDisplayf("%s, Вы должны быть в голосовом канале для использования этой команды", discord.UserMention(e.User().ID)),
							).WithAccessory(discord.NewThumbnail(e.User().EffectiveAvatarURL())),
						),
					),
			)
		}

		node := dj.GetNode(*e.GuildID(), *voiceState.ChannelID)

		if node == nil {
			return e.UpdateMessage(
				discord.NewMessageUpdate().
					WithComponents(
						discord.NewContainer(
							discord.NewSection(
								discord.NewTextDisplay("### Поставить на паузу песню"),
								discord.NewTextDisplayf("%s, в Вашем голосовом канале нет бота. Используйте команды `/play` или `/search`", discord.UserMention(e.User().ID)),
							).WithAccessory(discord.NewThumbnail(e.User().EffectiveAvatarURL())),
						),
					),
			)
		}

		player := node.LavalinkClient.ExistingPlayer(*e.GuildID())

		_ = player.Update(context.TODO(), lavalink.WithPaused(!player.Paused()))

		messageData := generateNowPlayingMessageData(node, *e.GuildID())

		if messageData == nil {
			return nil
		}

		return e.UpdateMessage(
			discord.NewMessageUpdate().
				WithComponents(messageData.Components...),
		)
	}
}

func HandlePlayerNextButton(dj *core.DJ) handler.ButtonComponentHandler {
	return func(data discord.ButtonInteractionData, e *handler.ComponentEvent) error {
		voiceState, ok := dj.Client.Caches.VoiceState(*e.GuildID(), e.User().ID)
		if !ok || voiceState.ChannelID == nil {
			return e.UpdateMessage(
				discord.NewMessageUpdate().
					WithComponents(
						discord.NewContainer(
							discord.NewSection(
								discord.NewTextDisplay("### Пропустить песню"),
								discord.NewTextDisplayf("%s, Вы должны быть в голосовом канале для использования этой команды", discord.UserMention(e.User().ID)),
							).WithAccessory(discord.NewThumbnail(e.User().EffectiveAvatarURL())),
						),
					),
			)
		}

		node := dj.GetNode(*e.GuildID(), *voiceState.ChannelID)

		if node == nil {
			return e.UpdateMessage(
				discord.NewMessageUpdate().
					WithComponents(
						discord.NewContainer(
							discord.NewSection(
								discord.NewTextDisplay("### Пропустить песню"),
								discord.NewTextDisplayf("%s, в Вашем голосовом канале нет бота. Используйте команды `/play` или `/search`", discord.UserMention(e.User().ID)),
							).WithAccessory(discord.NewThumbnail(e.User().EffectiveAvatarURL())),
						),
					),
			)
		}

		queue, _ := dj.Database.GetQueue(*e.GuildID(), node.ID())
		track, err := dj.Database.SkipTracks(queue.ID, 1)

		if err != nil {
			return e.CreateMessage(
				discord.NewMessageCreate().
					WithEphemeral(true).
					AddEmbeds(
						utils.NewBaseEmbed().
							WithTitlef("Пропустить песню").
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

		_ = node.LavalinkClient.ExistingPlayer(*e.GuildID()).Update(
			context.TODO(),
			lavalink.WithEncodedTrack(track.Encoded),
			lavalink.WithTrackUserData(userData),
		)

		messageData := generateNowPlayingMessageData(node, *e.GuildID())

		if messageData == nil {
			return nil
		}

		return e.UpdateMessage(
			discord.NewMessageUpdate().
				WithComponents(messageData.Components...),
		)
	}
}

func HandlePlayerRepeatButton(dj *core.DJ) handler.ButtonComponentHandler {
	return func(data discord.ButtonInteractionData, e *handler.ComponentEvent) error {
		voiceState, ok := dj.Client.Caches.VoiceState(*e.GuildID(), e.User().ID)
		if !ok || voiceState.ChannelID == nil {
			return e.UpdateMessage(
				discord.NewMessageUpdate().
					WithComponents(
						discord.NewContainer(
							discord.NewSection(
								discord.NewTextDisplay("### Изменить режим повторения"),
								discord.NewTextDisplayf("%s, Вы должны быть в голосовом канале для использования этой команды", discord.UserMention(e.User().ID)),
							).WithAccessory(discord.NewThumbnail(e.User().EffectiveAvatarURL())),
						),
					),
			)
		}

		node := dj.GetNode(*e.GuildID(), *voiceState.ChannelID)

		if node == nil {
			return e.UpdateMessage(
				discord.NewMessageUpdate().
					WithComponents(
						discord.NewContainer(
							discord.NewSection(
								discord.NewTextDisplay("### Изменить режим повторения"),
								discord.NewTextDisplayf("%s, в Вашем голосовом канале нет бота. Используйте команды `/play` или `/search`", discord.UserMention(e.User().ID)),
							).WithAccessory(discord.NewThumbnail(e.User().EffectiveAvatarURL())),
						),
					),
			)
		}

		_, _ = dj.Database.UpdateQueueType(node.ID(), *e.GuildID())

		messageData := generateNowPlayingMessageData(node, *e.GuildID())

		if messageData == nil {
			return nil
		}

		return e.UpdateMessage(
			discord.NewMessageUpdate().
				WithComponents(messageData.Components...),
		)
	}
}

func HandlePlayerRefreshButton(dj *core.DJ) handler.ButtonComponentHandler {
	return func(data discord.ButtonInteractionData, e *handler.ComponentEvent) error {
		voiceState, ok := dj.Client.Caches.VoiceState(*e.GuildID(), e.User().ID)
		if !ok || voiceState.ChannelID == nil {
			return e.UpdateMessage(
				discord.NewMessageUpdate().
					WithComponents(
						discord.NewContainer(
							discord.NewSection(
								discord.NewTextDisplay("### Посмотреть текущую песню"),
								discord.NewTextDisplayf("%s, Вы должны быть в голосовом канале для использования этой команды", discord.UserMention(e.User().ID)),
							).WithAccessory(discord.NewThumbnail(e.User().EffectiveAvatarURL())),
						),
					),
			)
		}

		node := dj.GetNode(*e.GuildID(), *voiceState.ChannelID)

		if node == nil {
			return e.UpdateMessage(
				discord.NewMessageUpdate().
					WithComponents(
						discord.NewContainer(
							discord.NewSection(
								discord.NewTextDisplay("### Посмотреть текущую песню"),
								discord.NewTextDisplayf("%s, в Вашем голосовом канале нет бота. Используйте команды `/play` или `/search`", discord.UserMention(e.User().ID)),
							).WithAccessory(discord.NewThumbnail(e.User().EffectiveAvatarURL())),
						),
					),
			)
		}

		messageData := generateNowPlayingMessageData(node, *e.GuildID())

		if messageData == nil {
			return nil
		}

		return e.UpdateMessage(
			discord.NewMessageUpdate().
				WithComponents(messageData.Components...),
		)
	}
}

func HandlePlayerStopButton(dj *core.DJ) handler.ButtonComponentHandler {
	return func(data discord.ButtonInteractionData, e *handler.ComponentEvent) error {
		voiceState, ok := dj.Client.Caches.VoiceState(*e.GuildID(), e.User().ID)
		if !ok || voiceState.ChannelID == nil {
			return e.UpdateMessage(
				discord.NewMessageUpdate().
					WithComponents(
						discord.NewContainer(
							discord.NewSection(
								discord.NewTextDisplay("### Очистить очередь и отключить бота"),
								discord.NewTextDisplayf("%s, Вы должны быть в голосовом канале для использования этой команды", discord.UserMention(e.User().ID)),
							).WithAccessory(discord.NewThumbnail(e.User().EffectiveAvatarURL())),
						),
					),
			)
		}

		node := dj.GetNode(*e.GuildID(), *voiceState.ChannelID)

		if node == nil {
			return e.UpdateMessage(
				discord.NewMessageUpdate().
					WithComponents(
						discord.NewContainer(
							discord.NewSection(
								discord.NewTextDisplay("### Очистить очередь и отключить бота"),
								discord.NewTextDisplayf("%s, в Вашем голосовом канале нет бота. Используйте команды `/play` или `/search`", discord.UserMention(e.User().ID)),
							).WithAccessory(discord.NewThumbnail(e.User().EffectiveAvatarURL())),
						),
					),
			)
		}

		_ = node.DiscordClient.UpdateVoiceState(context.TODO(), *e.GuildID(), nil, false, true)

		return e.UpdateMessage(
			discord.NewMessageUpdate().
				WithComponents(
					discord.NewContainer(
						discord.NewSection(
							discord.NewTextDisplay("### Очистить очередь и отключить бота"),
							discord.NewTextDisplayf("%s, бот успешно отключен", discord.UserMention(e.User().ID)),
						).WithAccessory(discord.NewThumbnail(e.User().EffectiveAvatarURL())),
					),
				),
		)
	}
}

func HandlePlayerVolumeButton(dj *core.DJ) handler.ButtonComponentHandler {
	return func(data discord.ButtonInteractionData, e *handler.ComponentEvent) error {
		voiceState, ok := dj.Client.Caches.VoiceState(*e.GuildID(), e.User().ID)
		if !ok || voiceState.ChannelID == nil {
			return e.UpdateMessage(
				discord.NewMessageUpdate().
					WithComponents(
						discord.NewContainer(
							discord.NewSection(
								discord.NewTextDisplay("### Изменить громкость"),
								discord.NewTextDisplayf("%s, Вы должны быть в голосовом канале для использования этой команды", discord.UserMention(e.User().ID)),
							).WithAccessory(discord.NewThumbnail(e.User().EffectiveAvatarURL())),
						),
					),
			)
		}

		node := dj.GetNode(*e.GuildID(), *voiceState.ChannelID)

		if node == nil {
			return e.UpdateMessage(
				discord.NewMessageUpdate().
					WithComponents(
						discord.NewContainer(
							discord.NewSection(
								discord.NewTextDisplay("### Изменить громкость"),
								discord.NewTextDisplayf("%s, в Вашем голосовом канале нет бота. Используйте команды `/play` или `/search`", discord.UserMention(e.User().ID)),
							).WithAccessory(discord.NewThumbnail(e.User().EffectiveAvatarURL())),
						),
					),
			)
		}

		return e.Modal(
			discord.NewModalCreate(
				"/player/volume/setModal",
				"Изменить громкость",
				discord.NewLabel(
					"Новая громкость в процентах",
					discord.NewShortTextInput("volume").
						WithMinLength(1).
						WithMaxLength(3).
						WithPlaceholder("Значение от 0 до 125").
						WithRequired(true),
				),
			),
		)
	}
}

func HandlePlayerVolumeModal(dj *core.DJ) handler.ModalHandler {
	return func(e *handler.ModalEvent) error {
		voiceState, ok := dj.Client.Caches.VoiceState(*e.GuildID(), e.User().ID)
		if !ok || voiceState.ChannelID == nil {
			return e.UpdateMessage(
				discord.NewMessageUpdate().
					WithComponents(
						discord.NewContainer(
							discord.NewSection(
								discord.NewTextDisplay("### Изменить громкость"),
								discord.NewTextDisplayf("%s, Вы должны быть в голосовом канале для использования этой команды", discord.UserMention(e.User().ID)),
							).WithAccessory(discord.NewThumbnail(e.User().EffectiveAvatarURL())),
						),
					),
			)
		}

		node := dj.GetNode(*e.GuildID(), *voiceState.ChannelID)

		if node == nil {
			return e.UpdateMessage(
				discord.NewMessageUpdate().
					WithComponents(
						discord.NewContainer(
							discord.NewSection(
								discord.NewTextDisplay("### Изменить громкость"),
								discord.NewTextDisplayf("%s, в Вашем голосовом канале нет бота. Используйте команды `/play` или `/search`", discord.UserMention(e.User().ID)),
							).WithAccessory(discord.NewThumbnail(e.User().EffectiveAvatarURL())),
						),
					),
			)
		}

		volume64, err := strconv.ParseInt(e.Data.Text("volume"), 10, 64)
		if err != nil || volume64 < 0 || volume64 > 125 {
			return e.CreateMessage(
				discord.NewMessageCreate().
					WithEphemeral(true).
					AddEmbeds(
						utils.NewBaseEmbed().
							WithTitle("Изменить громкость").
							WithThumbnail(e.User().EffectiveAvatarURL()).
							WithDescriptionf(
								"%s, **громкость** должна быть **числом** от **0** до **125**",
								discord.UserMention(e.User().ID),
							),
					),
			)
		}

		if err := node.LavalinkClient.ExistingPlayer(*e.GuildID()).Update(context.TODO(), lavalink.WithVolume(int(volume64))); err != nil {
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

func generateNowPlayingMessageData(
	node *core.Node,
	guildID snowflake.ID,
) *nowPlayingData {
	player := node.LavalinkClient.ExistingPlayer(guildID)

	nowPlaying := player.Track()

	if nowPlaying == nil {
		return nil
	}

	userData := database.TrackUserData{}
	_ = nowPlaying.UserData.Unmarshal(&userData)

	queue, _ := node.DJ.Database.GetQueue(guildID, node.ID())

	container := discord.NewContainer()

	mainSection := discord.NewSection().AddComponents(
		discord.NewTextDisplayf("**[%s](%s)**", nowPlaying.Info.Title, *nowPlaying.Info.URI),
		discord.NewTextDisplayf("**Исполнитель**: %s", nowPlaying.Info.Author),
	)

	if nowPlaying.Info.ArtworkURL != nil {
		mainSection = mainSection.WithAccessory(discord.NewThumbnail(*nowPlaying.Info.ArtworkURL))
	} else {
		mainSection = mainSection.WithAccessory(discord.NewThumbnail("https://i.imgur.com/b15x8hx.png"))
	}

	container = container.AddComponents(
		mainSection,
		discord.NewSmallSeparator(),
		discord.NewTextDisplayf("**Бот**: %s", discord.UserMention(node.DiscordClient.ApplicationID)),
		discord.NewTextDisplayf("**Добавил**: %s", userData.OrderedByTag),
		discord.NewSmallSeparator(),
		discord.NewTextDisplay(
			fmt.Sprintf(
				"**Громкость**: %d%%", player.Volume(),
			),
		),
		discord.NewTextDisplay(
			fmt.Sprintf(
				"`%s`/`%s`",
				utils.FormatDuration(player.Position()),
				utils.FormatDuration(nowPlaying.Info.Length),
			),
		),
		discord.NewTextDisplay(
			progress_bar.ProgressBar(
				player.Position().Milliseconds(),
				nowPlaying.Info.Length.Milliseconds(),
			),
		),
		discord.NewLargeSeparator(),
	)

	if len(queue.Tracks) > 0 {
		queueSection := discord.NewSection().WithAccessory(
			discord.NewPrimaryButton("", "/player/next").WithEmoji(discord.ComponentEmoji{Name: "⏭"}),
		)

		queueTrack := queue.Tracks[0]
		lavalinkNode := node.LavalinkClient.Node(queue.LavalinkNodeName)
		track, _ := lavalinkNode.DecodeTrack(context.TODO(), queueTrack.Encoded)

		queueSection = queueSection.AddComponents(
			discord.NewTextDisplay(
				fmt.Sprintf(
					"**Следующая песня в очереди**: [%s](%s)",
					track.Info.Title,
					*track.Info.URI,
				),
			),
			discord.NewTextDisplay(
				fmt.Sprintf(
					"**Песен в очереди**: %d",
					len(queue.Tracks),
				),
			),
		)

		container = container.AddComponents(
			queueSection,
			discord.NewLargeSeparator(),
		)
	}

	container = container.AddComponents(
		discord.NewActionRow(
			discord.NewPrimaryButton(
				"",
				"/player/pause",
			).WithEmoji(discord.ComponentEmoji{Name: utils.Ternary(player.Paused(), "▶️", "⏸️")}),
			discord.NewPrimaryButton(
				"",
				"/player/volume/set",
			).WithEmoji(discord.ComponentEmoji{Name: "🔊"}),
			utils.Ternary(
				queue.Type == database.QueueTypeNormal,
				discord.NewSecondaryButton(
					"",
					"/player/repeat",
				).WithEmoji(discord.ComponentEmoji{Name: queue.Type.Emoji()}),
				discord.NewSuccessButton(
					"",
					"/player/repeat",
				).WithEmoji(discord.ComponentEmoji{Name: queue.Type.Emoji()}),
			),
			discord.NewDangerButton(
				"",
				"/player/stop",
			).WithEmoji(discord.ComponentEmoji{Name: "⏹️"}),
			discord.NewSecondaryButton(
				"Обновить",
				"/player/refresh",
			).WithEmoji(discord.ComponentEmoji{Name: "🔄"}),
		),
	)

	return &nowPlayingData{
		Components: []discord.LayoutComponent{container},
	}
}
