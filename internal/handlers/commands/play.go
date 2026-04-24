package commands

import (
	"context"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/rs/zerolog/log"
	"github.com/yokkkoso/ongaku/internal/core"
	"github.com/yokkkoso/ongaku/internal/database"
	"github.com/yokkkoso/ongaku/internal/utils"
)

var searchPattern = regexp.MustCompile(`^(.{2})search:(.+)`)

var supportedDomains = []string{
	"spotify.link",
	"spotify.com",
	"yandex.ru",
	"yandex.by",
	"yandex.kz",
	"yandex.com",
	"soundcloud.com",
	"youtube.com",
	"youtu.be",
}

func isSupportedDomain(hostname string) bool {
	parts := strings.Split(hostname, ".")
	if len(parts) < 2 {
		return false
	}
	return slices.Contains(supportedDomains, parts[len(parts)-2]+"."+parts[len(parts)-1])
}

var playCommand = discord.SlashCommandCreate{
	Name:        "play",
	Description: "Включить песню",
	Contexts:    []discord.InteractionContextType{discord.InteractionContextTypeGuild},
	Options: []discord.ApplicationCommandOption{
		discord.ApplicationCommandOptionString{
			Name: "song",
			NameLocalizations: map[discord.Locale]string{
				discord.LocaleRussian: "песня",
			},
			Description: "Ссылка на песню или название",
			Required:    true,
		},
		discord.ApplicationCommandOptionString{
			Name:        "source",
			Description: "Источник песни",
			Required:    false,
			Choices: []discord.ApplicationCommandOptionChoiceString{
				{
					Name:  "Spotify (По умолчанию)",
					Value: "spsearch",
				},
				{
					Name:  "SoundCloud",
					Value: string(lavalink.SearchTypeSoundCloud),
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

func HandlePlaySlashCommand(dj *core.DJ) handler.SlashCommandHandler {
	return func(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
		rawIdentifier := data.String("song")
		var identifier string

		u, err := url.Parse(rawIdentifier)
		isURL := err == nil && (u.Scheme == "http" || u.Scheme == "https")

		switch {
		case isURL && isSupportedDomain(u.Hostname()):
			identifier = rawIdentifier
		case isURL:
			source, ok := data.OptString("source")
			if !ok {
				source = "spsearch"
			}
			identifier = lavalink.SearchType(source).Apply(rawIdentifier)
		default:
			if source, ok := data.OptString("source"); ok {
				identifier = lavalink.SearchType(source).Apply(rawIdentifier)
			} else if !searchPattern.MatchString(rawIdentifier) {
				identifier = lavalink.SearchTypeSoundCloud.Apply(rawIdentifier)
			} else {
				identifier = rawIdentifier
			}
		}

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

		if err := e.DeferCreateMessage(true); err != nil {
			return err
		}

		node := dj.GetFreeNode(*e.GuildID(), *voiceState.ChannelID)

		if node == nil {
			_, _ = e.UpdateInteractionResponse(
				discord.NewMessageUpdate().
					AddEmbeds(
						utils.NewBaseEmbed().
							WithTitlef("Включить песню").
							WithThumbnail(e.User().EffectiveAvatarURL()).
							WithDescriptionf(
								"%s, все боты сейчас заняты, подождите немного и повторите попытку!",
								discord.UserMention(e.User().ID),
							),
					),
			)

			return nil
		}

		lavalinkNode := node.LavalinkClient.BestNode()
		player := node.LavalinkClient.Player(*e.GuildID())
		queue, _ := dj.Database.FindOrCreateQueue(lavalinkNode.Config().Name, node.ID(), *e.GuildID(), *voiceState.ChannelID)
		var toPlay *lavalink.Track

		lavalinkNode.LoadTracksHandler(
			context.TODO(),
			identifier,
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

					_, _ = e.UpdateInteractionResponse(
						discord.NewMessageUpdate().
							AddEmbeds(
								utils.NewBaseEmbed().
									WithTitlef("Включить песню").
									WithThumbnail(e.User().EffectiveAvatarURL()).
									WithDescriptionf(
										"%s, трек [%s — %s](%s) добавлен в очередь",
										discord.UserMention(e.User().ID),
										track.Info.Author,
										track.Info.Title,
										*track.Info.URI,
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
					if len(playlist.Tracks) <= 0 {
						_, _ = e.UpdateInteractionResponse(
							discord.NewMessageUpdate().
								AddEmbeds(
									utils.NewBaseEmbed().
										WithTitlef("Включить песню").
										WithThumbnail(e.User().EffectiveAvatarURL()).
										WithDescriptionf(
											"%s, по запросу `%s` ничего не найдено",
											discord.UserMention(e.User().ID),
											rawIdentifier,
										),
								),
						)

						return
					}
					_, _ = e.UpdateInteractionResponse(
						discord.NewMessageUpdate().
							AddEmbeds(
								utils.NewBaseEmbed().
									WithTitlef("Включить песню").
									WithThumbnail(e.User().EffectiveAvatarURL()).
									WithDescriptionf(
										"%s, плейлист `%s` с `%d` %s добавлен в очередь",
										discord.UserMention(e.User().ID),
										playlist.Info.Name,
										len(playlist.Tracks),
										utils.GetDeclensionWord(len(playlist.Tracks), [3]string{"треком", "треками", "треками"}),
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
					if len(tracks) <= 0 {
						_, _ = e.UpdateInteractionResponse(
							discord.NewMessageUpdate().
								AddEmbeds(
									utils.NewBaseEmbed().
										WithTitlef("Включить песню").
										WithThumbnail(e.User().EffectiveAvatarURL()).
										WithDescriptionf(
											"%s, по запросу `%s` ничего не найдено",
											discord.UserMention(e.User().ID),
											rawIdentifier,
										),
								),
						)

						return
					}

					track, _ := tracks[0].WithUserData(
						database.TrackUserData{
							OrderedByID:      e.User().ID,
							OrderedByTag:     e.User().Tag(),
							InteractionToken: e.Token(),
						},
					)

					_, _ = e.UpdateInteractionResponse(
						discord.NewMessageUpdate().
							AddEmbeds(
								utils.NewBaseEmbed().
									WithTitlef("Включить песню").
									WithThumbnail(e.User().EffectiveAvatarURL()).
									WithDescriptionf(
										"%s, трек [%s — %s](%s) добавлен в очередь",
										discord.UserMention(e.User().ID),
										track.Info.Author,
										track.Info.Title,
										*track.Info.URI,
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
					_, _ = e.UpdateInteractionResponse(
						discord.NewMessageUpdate().
							AddEmbeds(
								utils.NewBaseEmbed().
									WithTitlef("Включить песню").
									WithThumbnail(e.User().EffectiveAvatarURL()).
									WithDescriptionf(
										"%s, по запросу `%s` ничего не найдено",
										discord.UserMention(e.User().ID),
										rawIdentifier,
									),
							),
					)
				},

				func(err error) {
					_, _ = e.UpdateInteractionResponse(
						discord.NewMessageUpdate().
							AddEmbeds(
								utils.NewBaseEmbed().
									WithTitlef("Включить песню").
									WithThumbnail(e.User().EffectiveAvatarURL()).
									WithDescriptionf(
										"%s, по запросу `%s` ничего не найдено",
										discord.UserMention(e.User().ID),
										rawIdentifier,
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
