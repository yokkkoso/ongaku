package commands

import (
	"context"
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/lavasearch-plugin"
	"github.com/disgoorg/lavasrc-plugin"
	"github.com/rs/zerolog/log"
	"github.com/yokkkoso/ongaku/internal/core"
	"github.com/yokkkoso/ongaku/internal/database"
	"github.com/yokkkoso/ongaku/internal/utils"
	"github.com/yokkkoso/ongaku/internal/utils/uri_cache"
)

var searchCommand = discord.SlashCommandCreate{
	Name:        "search",
	Description: "Найти песню или альбом",
	Contexts:    []discord.InteractionContextType{discord.InteractionContextTypeGuild},
	Options: []discord.ApplicationCommandOption{
		discord.ApplicationCommandOptionString{
			Name: "query",
			NameLocalizations: map[discord.Locale]string{
				discord.LocaleRussian: "запрос",
			},
			Description: "Название песни или альбома",
			Required:    true,
		},
		discord.ApplicationCommandOptionString{
			Name:        "source",
			Description: "Источник",
			Required:    false,
			Choices: []discord.ApplicationCommandOptionChoiceString{
				{
					Name:  "Spotify (По умолчанию)",
					Value: "spsearch",
				},
				{
					Name:  "Yandex Music",
					Value: "ymsearch",
				},
				{
					Name:  "YouTube",
					Value: string(lavalink.SearchTypeYouTube),
				},
			},
		},
	},
}

func HandleSearchSlashCommand(dj *core.DJ) handler.SlashCommandHandler {
	return func(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
		rawIdentifier := data.String("query")
		var identifier string

		if source, ok := data.OptString("source"); ok {
			identifier = lavalink.SearchType(source).Apply(rawIdentifier)
		} else if !searchPattern.MatchString(rawIdentifier) {
			identifier = lavalink.SearchType("spsearch").Apply(rawIdentifier)
		} else {
			identifier = rawIdentifier
		}

		voiceState, ok := dj.Client.Caches.VoiceState(*e.GuildID(), e.User().ID)
		if !ok || voiceState.ChannelID == nil {
			return e.CreateMessage(
				discord.NewMessageCreate().
					WithEphemeral(true).
					AddEmbeds(
						utils.NewBaseEmbed().
							WithTitlef("Найти песню или альбом").
							WithThumbnail(e.User().EffectiveAvatarURL()).
							WithDescriptionf(
								"%s, Вы должны быть в голосовом канале для использования этой команды",
								discord.UserMention(e.User().ID),
							),
					),
			)
		}

		node := dj.GetFreeNode(*e.GuildID(), *voiceState.ChannelID)

		if node == nil {
			return e.CreateMessage(
				discord.NewMessageCreate().
					WithEphemeral(true).
					AddEmbeds(
						utils.NewBaseEmbed().
							WithTitlef("Найти песню или альбом").
							WithThumbnail(e.User().EffectiveAvatarURL()).
							WithDescriptionf(
								"%s, все боты сейчас заняты, подождите немного и повторите попытку!",
								discord.UserMention(e.User().ID),
							),
					),
			)
		}

		if err := e.DeferCreateMessage(true); err != nil {
			return err
		}

		node.IsPreTaken.Store(false)

		result, err := lavasearch.LoadSearch(
			context.TODO(),
			node.LavalinkClient.BestNode().Rest(),
			identifier,
			[]lavasearch.SearchType{lavasearch.SearchTypeAlbum, lavasearch.SearchTypeTrack},
		)

		if err != nil || (len(result.Tracks) <= 0 && len(result.Albums) <= 0) {
			_, err = e.UpdateInteractionResponse(
				discord.NewMessageUpdate().
					AddEmbeds(
						utils.NewBaseEmbed().
							WithTitlef("Найти песню или альбом").
							WithThumbnail(e.User().EffectiveAvatarURL()).
							WithDescriptionf(
								"%s, по запросу `%s` ничего не найдено",
								discord.UserMention(e.User().ID),
								rawIdentifier,
							),
					),
			)

			return err
		}

		var container = discord.NewContainer(
			discord.NewTextDisplayf("**Результаты поиска по запросу:** %s", rawIdentifier),
			discord.NewTextDisplay("Нажмите на \"✅\" у нужного результата"),
		)

		for _, track := range result.Tracks {
			if len(container.Components) >= 6 {
				break
			}

			container = container.AddComponents(
				discord.NewSection(
					discord.NewTextDisplay(
						fmt.Sprintf(
							"🎶 [%s — %s](%s) | %s\n",
							track.Info.Author,
							track.Info.Title,
							*track.Info.URI,
							utils.TracksDuration(track),
						),
					),
				).WithAccessory(
					discord.NewPrimaryButton(
						"",
						fmt.Sprintf("/searchResult/%s", uri_cache.CacheURI(*track.Info.URI)),
					).WithEmoji(discord.ComponentEmoji{Name: "✅"}),
				),
			)
		}

		for _, album := range result.Albums {
			if len(container.Components) >= 12 {
				break
			}

			var albumInfo lavasrc.PlaylistInfo
			_ = album.PluginInfo.Unmarshal(&albumInfo)

			container = container.AddComponents(
				discord.NewSection(
					discord.NewTextDisplay(
						fmt.Sprintf(
							"💿 [%s — %s](%s) | %d %s\n",
							albumInfo.Author,
							album.Info.Name,
							albumInfo.URL,
							albumInfo.TotalTracks,
							utils.GetDeclensionWord(albumInfo.TotalTracks, [3]string{"трек", "трека", "треков"}),
						),
					),
				).WithAccessory(
					discord.NewPrimaryButton(
						"",
						fmt.Sprintf("/searchResult/%s", uri_cache.CacheURI(albumInfo.URL)),
					).WithEmoji(discord.ComponentEmoji{Name: "✅"}),
				),
			)
		}

		_, err = e.UpdateInteractionResponse(
			discord.NewMessageUpdate().
				WithIsComponentsV2(true).
				WithComponents(container),
		)

		return err
	}
}

func HandleSearchResultButton(dj *core.DJ) handler.ButtonComponentHandler {
	return func(data discord.ButtonInteractionData, e *handler.ComponentEvent) error {
		voiceState, ok := dj.Client.Caches.VoiceState(*e.GuildID(), e.User().ID)
		if !ok || voiceState.ChannelID == nil {
			return e.UpdateMessage(
				discord.NewMessageUpdate().
					WithComponents(
						discord.NewContainer(
							discord.NewSection(
								discord.NewTextDisplay("### Найти песню"),
								discord.NewTextDisplayf("%s, Вы должны быть в голосовом канале для использования этой команды", discord.UserMention(e.User().ID)),
							).WithAccessory(discord.NewThumbnail(e.User().EffectiveAvatarURL())),
						),
					),
			)
		}

		node := dj.GetFreeNode(*e.GuildID(), *voiceState.ChannelID)

		if node == nil {
			return e.UpdateMessage(
				discord.NewMessageUpdate().
					WithComponents(
						discord.NewContainer(
							discord.NewSection(
								discord.NewTextDisplay("### Найти песню"),
								discord.NewTextDisplayf("%s, все боты сейчас заняты, подождите немного и повторите попытку!", discord.UserMention(e.User().ID)),
							).WithAccessory(discord.NewThumbnail(e.User().EffectiveAvatarURL())),
						),
					),
			)
		}

		shortHash := e.Vars["uri"]
		uri := uri_cache.GetCachedURI(shortHash)
		if uri == nil {
			return e.UpdateMessage(
				discord.NewMessageUpdate().
					WithComponents(
						discord.NewContainer(
							discord.NewSection(
								discord.NewTextDisplay("### Найти песню"),
								discord.NewTextDisplayf("%s, срок действия кнопки истек. Пожалуйста, выполните поиск снова", discord.UserMention(e.User().ID)),
							).WithAccessory(discord.NewThumbnail(e.User().EffectiveAvatarURL())),
						),
					),
			)
		}

		lavalinkNode := node.LavalinkClient.BestNode()
		player := node.LavalinkClient.Player(*e.GuildID())
		queue, _ := dj.Database.FindOrCreateQueue(lavalinkNode.Config().Name, node.ID(), *e.GuildID(), *voiceState.ChannelID)
		var toPlay *lavalink.Track

		lavalinkNode.LoadTracksHandler(
			context.TODO(),
			uri.Value(),
			disgolink.NewResultHandler(
				// Single track
				func(track lavalink.Track) {
					track, _ = track.WithUserData(
						database.TrackUserData{
							OrderedByID:      e.User().ID,
							OrderedByTag:     e.User().Tag(),
							InteractionToken: e.Token(),
						},
					)

					_ = e.UpdateMessage(
						discord.NewMessageUpdate().
							WithComponents(
								discord.NewContainer(
									discord.NewSection(
										discord.NewTextDisplay("### Включить песню"),
										discord.NewTextDisplayf(
											"%s, трек [%s — %s](%s) добавлен в очередь",
											discord.UserMention(e.User().ID),
											track.Info.Author,
											track.Info.Title,
											*track.Info.URI,
										),
									).WithAccessory(discord.NewThumbnail(e.User().EffectiveAvatarURL())),
								),
							),
					)

					if player.Track() == nil {
						toPlay = &track
					} else {
						if _, err := dj.Database.CreateTrack(queue.ID, track); err != nil {
							log.Err(err).Uint("queue", queue.ID).Msg("Failed to create track")
						}
					}
				},

				// Playlist
				func(playlist lavalink.Playlist) {
					_ = e.UpdateMessage(
						discord.NewMessageUpdate().
							WithComponents(
								discord.NewContainer(
									discord.NewSection(
										discord.NewTextDisplay("### Включить песню"),
										discord.NewTextDisplayf(
											"%s, плейлист `%s` с `%d` %s добавлен в очередь",
											discord.UserMention(e.User().ID),
											playlist.Info.Name,
											len(playlist.Tracks),
											utils.GetDeclensionWord(len(playlist.Tracks), [3]string{"треком", "треками", "треками"}),
										),
									).WithAccessory(discord.NewThumbnail(e.User().EffectiveAvatarURL())),
								),
							),
					)

					for i := range playlist.Tracks {
						playlist.Tracks[i], _ = playlist.Tracks[i].WithUserData(
							database.TrackUserData{
								OrderedByID:      e.User().ID,
								OrderedByTag:     e.User().Tag(),
								InteractionToken: e.Token(),
							},
						)
					}

					if player.Track() == nil {
						toPlay = &playlist.Tracks[0]

						for _, track := range playlist.Tracks[1:] {
							if _, err := dj.Database.CreateTrack(queue.ID, track); err != nil {
								log.Err(err).Uint("queue", queue.ID).Msg("Failed to create track")
							}
						}
					} else {
						for _, track := range playlist.Tracks {
							if _, err := dj.Database.CreateTrack(queue.ID, track); err != nil {
								log.Err(err).Uint("queue", queue.ID).Msg("Failed to create track")
							}
						}
					}
				},

				// Search track
				func(tracks []lavalink.Track) {
					track, _ := tracks[0].WithUserData(
						database.TrackUserData{
							OrderedByID:      e.User().ID,
							OrderedByTag:     e.User().Tag(),
							InteractionToken: e.Token(),
						},
					)

					_ = e.UpdateMessage(
						discord.NewMessageUpdate().
							WithComponents(
								discord.NewContainer(
									discord.NewSection(
										discord.NewTextDisplay("### Включить песню"),
										discord.NewTextDisplayf(
											"%s, трек [%s — %s](%s) добавлен в очередь",
											discord.UserMention(e.User().ID),
											track.Info.Author,
											track.Info.Title,
											*track.Info.URI,
										),
									).WithAccessory(discord.NewThumbnail(e.User().EffectiveAvatarURL())),
								),
							),
					)

					if player.Track() == nil {
						toPlay = &track
					} else {
						if _, err := dj.Database.CreateTrack(queue.ID, track); err != nil {
							log.Err(err).Uint("queue", queue.ID).Msg("Failed to create track")
						}
					}
				},

				func() {
					_ = e.UpdateMessage(
						discord.NewMessageUpdate().
							WithComponents(
								discord.NewContainer(
									discord.NewSection(
										discord.NewTextDisplay("### Включить песню"),
										discord.NewTextDisplayf(
											"%s, по Вашему запросу ничего не найдено",
											discord.UserMention(e.User().ID),
										),
									).WithAccessory(discord.NewThumbnail(e.User().EffectiveAvatarURL())),
								),
							),
					)
				},

				func(err error) {
					_ = e.UpdateMessage(
						discord.NewMessageUpdate().
							WithComponents(
								discord.NewContainer(
									discord.NewSection(
										discord.NewTextDisplay("### Включить песню"),
										discord.NewTextDisplayf(
											"%s, по Вашему запросу ничего не найдено",
											discord.UserMention(e.User().ID),
										),
									).WithAccessory(discord.NewThumbnail(e.User().EffectiveAvatarURL())),
								),
							),
					)
				},
			),
		)

		if toPlay == nil {
			node.IsPreTaken.Store(false)
			return nil
		}

		if err := node.DiscordClient.UpdateVoiceState(
			context.TODO(),
			*e.GuildID(),
			voiceState.ChannelID,
			false,
			true,
		); err != nil {
			node.IsPreTaken.Store(false)
			return err
		}

		node.IsPreTaken.Store(false)

		return player.Update(context.TODO(), lavalink.WithTrack(*toPlay), lavalink.WithVolume(75))
	}
}
