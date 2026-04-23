package commands

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/json"
	"github.com/disgoorg/snowflake/v2"
	"github.com/yokkkoso/musicbot/internal/core"
	"github.com/yokkkoso/musicbot/internal/database"
	"github.com/yokkkoso/musicbot/internal/utils"
	"github.com/yokkkoso/musicbot/internal/utils/array"
	"github.com/yokkkoso/musicbot/internal/utils/exptime"
	"github.com/yokkkoso/musicbot/internal/utils/pagination"
)

var queueCommand = discord.SlashCommandCreate{
	Name:        "queue",
	Description: "Посмотреть очередь песен",
	Contexts:    []discord.InteractionContextType{discord.InteractionContextTypeGuild},
}

type queuePageData struct {
	Components []discord.LayoutComponent
	TotalPages int
}

func HandleQueueSlashCommand(dj *core.DJ) handler.SlashCommandHandler {
	return func(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
		voiceState, ok := dj.Client.Caches.VoiceState(*e.GuildID(), e.User().ID)
		if !ok || voiceState.ChannelID == nil {
			return e.CreateMessage(
				discord.NewMessageCreate().
					WithEphemeral(true).
					AddEmbeds(
						utils.NewBaseEmbed().
							WithTitlef("Включить песню").
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
							WithTitlef("Включить песню").
							WithThumbnail(e.User().EffectiveAvatarURL()).
							WithDescriptionf(
								"%s, в Вашем голосовом канале нет бота. Используйте команды `/play` или `/search`",
								discord.UserMention(e.User().ID),
							),
					),
			)
		}

		pageData := generateQueuePage(node, *e.GuildID(), e.User().ID, exptime.GetExpTime(), 1)

		return e.CreateMessage(
			discord.NewMessageCreate().
				WithEphemeral(true).
				WithIsComponentsV2(true).
				WithComponents(pageData.Components...),
		)
	}
}

func HandleQueuePaginationButton(dj *core.DJ) handler.ButtonComponentHandler {
	return func(data discord.ButtonInteractionData, e *handler.ComponentEvent) error {
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

		currentPage, _ := strconv.Atoi(e.Vars["currentPage"])
		totalPages, _ := strconv.Atoi(e.Vars["totalPages"])

		pageAction := pagination.PageAction(e.Vars["pageAction"])

		switch pageAction {
		case pagination.FirstPageAction:
			{
				currentPage = 1
			}
		case pagination.PrevPageAction:
			{
				currentPage -= 1
			}
		case pagination.NextPageAction:
			{
				currentPage += 1
			}
		case pagination.LastPageAction:
			{
				currentPage = totalPages
			}
		}

		pageData := generateQueuePage(node, *e.GuildID(), e.User().ID, exptime.GetExpTime(), currentPage)

		return e.UpdateMessage(
			discord.NewMessageUpdate().
				WithComponents(pageData.Components...),
		)
	}
}

func HandleQueueRemoveElementButton(dj *core.DJ) handler.ButtonComponentHandler {
	return func(data discord.ButtonInteractionData, e *handler.ComponentEvent) error {
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

		_, _ = dj.Database.DeleteTrack(e.Vars["identifier"])

		pageData := generateQueuePage(node, *e.GuildID(), e.User().ID, exptime.GetExpTime(), 1)

		return e.UpdateMessage(
			discord.NewMessageUpdate().
				WithComponents(pageData.Components...),
		)
	}
}

func HandleQueueRefreshButton(dj *core.DJ) handler.ButtonComponentHandler {
	return func(data discord.ButtonInteractionData, e *handler.ComponentEvent) error {
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

		pageData := generateQueuePage(node, *e.GuildID(), e.User().ID, exptime.GetExpTime(), 1)

		return e.UpdateMessage(
			discord.NewMessageUpdate().
				WithComponents(pageData.Components...),
		)
	}
}

func generateQueuePage(
	node *core.Node,
	guildID snowflake.ID,
	executorID snowflake.ID,
	expTime string,
	page int,
) queuePageData {
	queue, _ := node.DJ.Database.GetQueue(guildID, node.ID())

	totalPages := int(math.Ceil(float64(len(queue.Tracks)) / 8))

	encodedTracks := make([]string, 0, len(queue.Tracks))

	for _, track := range queue.Tracks {
		encodedTracks = append(encodedTracks, track.Encoded)
	}

	var nowPlaying = node.LavalinkClient.ExistingPlayer(guildID).Track()

	var container = discord.NewContainer()

	var duration string
	nowPlayingUserData := database.TrackUserData{}

	lavalinkNode := node.LavalinkClient.Node(queue.LavalinkNodeName)

	tracks, _ := lavalinkNode.DecodeTracks(context.TODO(), encodedTracks)

	if nowPlaying != nil {
		allTracks := make([]lavalink.Track, 0, len(queue.Tracks)+1)
		allTracks = append(allTracks, *nowPlaying)
		allTracks = append(allTracks, tracks...)
		duration = utils.TracksDuration(allTracks...)

		_ = nowPlaying.UserData.Unmarshal(&nowPlayingUserData)
	} else {
		duration = utils.TracksDuration(tracks...)
	}

	container = container.AddComponents(
		discord.NewTextDisplayf("**Бот**: %s", discord.UserMention(node.DiscordClient.ApplicationID)),
		discord.NewTextDisplayf("**Длительность очереди:** %d | %s", len(queue.Tracks)+1, duration),
	)

	if nowPlaying != nil {
		container = container.AddComponents(
			discord.NewTextDisplay(
				fmt.Sprintf(
					"**Сейчас играет:** [%s — %s](%s) | %s | `%s`",
					nowPlaying.Info.Author,
					nowPlaying.Info.Title,
					*nowPlaying.Info.URI,
					utils.TracksDuration(*nowPlaying),
					nowPlayingUserData.OrderedByTag,
				),
			),
			discord.NewLargeSeparator(),
		)
	}

	for index := 0; index < len(queue.Tracks) && index < 8; index++ {
		pageIndex := index + 8*(page-1)

		if pageIndex < len(queue.Tracks) {
			track := queue.Tracks[pageIndex]

			decodedTrack, ok := array.Find(
				tracks,
				func(etrack lavalink.Track) bool {
					return track.Encoded == etrack.Encoded
				},
			)

			if !ok {
				continue
			}

			userData := database.TrackUserData{}
			_ = json.Unmarshal([]byte(track.UserData), &userData)

			container = container.AddComponents(
				discord.NewSection(
					discord.NewTextDisplay(
						fmt.Sprintf(
							"%d) [%s — %s](%s) | %s | `%s`\n",
							pageIndex+1,
							decodedTrack.Info.Author,
							decodedTrack.Info.Title,
							*decodedTrack.Info.URI,
							utils.TracksDuration(decodedTrack),
							userData.OrderedByTag,
						),
					),
				).WithAccessory(
					discord.NewDangerButton(
						"",
						fmt.Sprintf("/queueDelete/%d", track.ID),
					).WithEmoji(discord.ComponentEmoji{Name: "🗑️"}),
				),
			)
		}
	}

	container = container.AddComponents(
		discord.NewActionRow(
			discord.NewPrimaryButton(
				"Обновить очередь",
				"/queueRefresh",
			).WithEmoji(discord.ComponentEmoji{Name: "🔄"}),
		),
	)

	components := []discord.LayoutComponent{container}

	if totalPages > 1 {
		components = append(
			components,
			discord.NewContainer(
				discord.NewTextDisplayf("Страница %d из %d", page, totalPages),
				pagination.GeneratePageButtons(
					"queue",
					expTime,
					executorID,
					page,
					totalPages,
				),
			),
		)
	}

	return queuePageData{
		Components: components,
		TotalPages: totalPages,
	}
}
