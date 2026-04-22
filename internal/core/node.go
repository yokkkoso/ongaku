package core

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/godave/golibdave"
	"github.com/disgoorg/json"
	"github.com/disgoorg/snowflake/v2"
	"github.com/rs/zerolog/log"
	"gitlab.com/yokkkoso/musicbot/internal/config_manager"
	"gitlab.com/yokkkoso/musicbot/internal/database"
	"gitlab.com/yokkkoso/musicbot/internal/utils"
)

type Node struct {
	DJ             *DJ
	DiscordClient  *bot.Client
	LavalinkClient disgolink.Client
	IsPreTaken     atomic.Bool
	leaveTimers    sync.Map
}

func NewNode(dj *DJ) *Node {
	return &Node{
		DJ: dj,
	}
}

func (n *Node) onVoiceStateUpdate(event *events.GuildVoiceStateUpdate) {
	if event.VoiceState.UserID == n.ID() {
		n.LavalinkClient.OnVoiceStateUpdate(
			context.TODO(),
			event.VoiceState.GuildID,
			event.VoiceState.ChannelID,
			event.VoiceState.SessionID,
		)

		if event.VoiceState.ChannelID != nil &&
			event.OldVoiceState.ChannelID != nil &&
			*event.VoiceState.ChannelID != *event.OldVoiceState.ChannelID {
			_ = n.DJ.Database.UpdateChannel(n.ID(), event.VoiceState.GuildID, *event.VoiceState.ChannelID)
		}

		if event.VoiceState.ChannelID == nil {
			_ = n.DJ.Database.DeleteQueueByNodeAndGuild(n.ID(), event.VoiceState.GuildID)

			if timer, exists := n.leaveTimers.LoadAndDelete(event.VoiceState.GuildID); exists {
				timer.(*time.Timer).Stop()
			}
		}
		return
	}

	botVoiceState, ok := n.DJ.Client.Caches.VoiceState(event.VoiceState.GuildID, n.ID())
	if !ok || botVoiceState.ChannelID == nil {
		return
	}

	if !utils.PtrEqual(event.VoiceState.ChannelID, botVoiceState.ChannelID) &&
		!utils.PtrEqual(event.OldVoiceState.ChannelID, botVoiceState.ChannelID) {
		return
	}

	var listeningUsersInChannel []snowflake.ID

	for state := range n.DJ.Client.Caches.VoiceStates(event.VoiceState.GuildID) {
		if !state.SelfDeaf && !state.GuildDeaf && state.ChannelID != nil && *state.ChannelID == *botVoiceState.ChannelID {
			listeningUsersInChannel = append(listeningUsersInChannel, state.UserID)
		}
	}

	if len(listeningUsersInChannel) <= 1 {
		if _, exists := n.leaveTimers.Load(event.VoiceState.GuildID); !exists {
			timer := time.AfterFunc(
				5*time.Minute,
				func() {
					_ = n.DiscordClient.UpdateVoiceState(context.TODO(), event.VoiceState.GuildID, nil, false, false)
					_ = n.DJ.Database.DeleteQueueByNodeAndGuild(n.ID(), event.VoiceState.GuildID)

					n.leaveTimers.Delete(event.VoiceState.GuildID)
				},
			)

			n.leaveTimers.Store(event.VoiceState.GuildID, timer)
		}
	} else {
		if timer, exists := n.leaveTimers.LoadAndDelete(event.VoiceState.GuildID); exists {
			timer.(*time.Timer).Stop()
			n.leaveTimers.Delete(event.VoiceState.GuildID)
		}
	}
}

func (n *Node) onVoiceServerUpdate(event *events.VoiceServerUpdate) {
	if event.Endpoint != nil {
		n.LavalinkClient.OnVoiceServerUpdate(context.TODO(), event.GuildID, event.Token, *event.Endpoint)
	}
}

func (n *Node) onReady(e *events.Ready) {
	user, _ := e.Client().Caches.SelfUser()

	log.Info().Msgf("Node %s is ready", user.Tag())

	if err := e.Client().SetPresence(
		context.TODO(),
		gateway.WithOnlineStatus(discord.OnlineStatusIdle),
		gateway.WithCustomActivity("🎶"),
	); err != nil {
		log.Err(err).Str("node", n.ID().String()).Msg("Failed to set node presence")
	}
}

func (n *Node) SetUpClients(
	nodeConfig config_manager.DiscordNodeConfig,
	lavalinkNodeConfigs []config_manager.LavalinkNodeConfig,
) error {
	var err error
	if n.DiscordClient, err = disgo.New(
		nodeConfig.Token,
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(gateway.IntentGuilds, gateway.IntentGuildVoiceStates),
		),
		bot.WithCacheConfigOpts(
			cache.WithCaches(cache.FlagVoiceStates, cache.FlagGuilds),
		),
		bot.WithEventListenerFunc(n.onVoiceStateUpdate),
		bot.WithEventListenerFunc(n.onVoiceServerUpdate),
		bot.WithEventListenerFunc(n.onReady),
		bot.WithVoiceManagerConfigOpts(
			voice.WithDaveSessionCreateFunc(golibdave.NewSession),
		),
	); err != nil {
		log.Err(err).Msg("Failed to start node bot")
		return err
	}

	n.LavalinkClient = disgolink.New(
		n.ID(),
		disgolink.WithListenerFunc(n.onTrackEnd),
		disgolink.WithListenerFunc(n.onTrackException),
		disgolink.WithListenerFunc(n.onTrackStuck),
	)

	for _, nodeConfig := range lavalinkNodeConfigs {
		_, err := n.LavalinkClient.AddNode(
			context.TODO(),
			disgolink.NodeConfig{
				Name:     nodeConfig.Name,
				Address:  nodeConfig.Address,
				Password: nodeConfig.Password,
				Secure:   nodeConfig.Secure,
			},
		)

		if err != nil {
			log.Err(err).Msg("Failed to add lavalink node")
		}
	}

	return nil
}

func (n *Node) ID() snowflake.ID {
	return n.DiscordClient.ApplicationID
}

func (n *Node) onTrackEnd(player disgolink.Player, event lavalink.TrackEndEvent) {
	if !event.Reason.MayStartNext() {
		return
	}

	const maxAttempts = 3
	for attempt := 0; attempt < maxAttempts; attempt++ {
		queue, err := n.DJ.Database.GetQueue(event.GuildID(), n.ID())
		if err != nil {
			return
		}

		var (
			nextTrackEncoded  string
			nextTrackUserData database.TrackUserData
		)

		switch queue.Type {
		case database.QueueTypeNormal:
			nextTrack, err := n.DJ.Database.SkipTracks(queue.ID, 1)
			if err != nil {
				_ = n.DiscordClient.UpdateVoiceState(context.TODO(), player.GuildID(), nil, false, true)
				return
			}

			nextTrackEncoded = nextTrack.Encoded
			_ = json.Unmarshal([]byte(nextTrack.UserData), &nextTrackUserData)

		case database.QueueTypeRepeatTrack:
			nextTrackEncoded = event.Track.Encoded
			_ = event.Track.UserData.Unmarshal(&nextTrackUserData)

		case database.QueueTypeRepeatQueue:
			if _, err := n.DJ.Database.CreateTrack(queue.ID, event.Track); err != nil {
				log.Err(err).Uint("queue", queue.ID).Msg("Failed to create track")
				return
			}

			nextTrack, err := n.DJ.Database.SkipTracks(queue.ID, 1)
			if err != nil {
				_ = n.DiscordClient.UpdateVoiceState(context.TODO(), player.GuildID(), nil, false, true)
				return
			}

			nextTrackEncoded = nextTrack.Encoded
			_ = json.Unmarshal([]byte(nextTrack.UserData), &nextTrackUserData)
		}

		if err := player.Update(context.TODO(), lavalink.WithEncodedTrack(nextTrackEncoded), lavalink.WithTrackUserData(nextTrackUserData)); err != nil {
			log.Err(err).Msg("Failed to play track")
			continue
		}
		return
	}
}

func (n *Node) onTrackException(player disgolink.Player, event lavalink.TrackExceptionEvent) {
	if strings.Contains(event.Exception.Message, "This video is unavailable") ||
		strings.Contains(event.Exception.Message, "This video is private") ||
		strings.Contains(event.Exception.Message, "This video is not available") ||
		strings.Contains(event.Exception.Message, "Something broke when playing the track") {
		userData := database.TrackUserData{}
		err := event.Track.UserData.Unmarshal(&userData)

		if err != nil {
			return
		}

		_, _ = n.DJ.Client.Rest.UpdateInteractionResponse(
			n.DJ.Client.ApplicationID,
			userData.InteractionToken,
			discord.NewMessageUpdate().
				AddEmbeds(
					utils.NewBaseEmbed().
						WithDescriptionf(
							"%s, не удалость включить [%s](%s). Возможно песня недоступна или имеет ограничение по возрасту, попробуйте другой источник",
							discord.UserMention(userData.OrderedByID),
							event.Track.Info.Title,
							*event.Track.Info.URI,
						),
				),
		)

		return
	}

	log.Error().Str("node", n.ID().String()).Any("event", event).Msg("Track exception")

	const maxAttempts = 3
	for attempt := 0; attempt < maxAttempts; attempt++ {
		queue, err := n.DJ.Database.GetQueue(event.GuildID(), n.ID())
		if err != nil {
			return
		}

		var (
			nextTrackEncoded  string
			nextTrackUserData database.TrackUserData
		)

		switch queue.Type {
		case database.QueueTypeNormal:
			nextTrack, err := n.DJ.Database.SkipTracks(queue.ID, 1)
			if err != nil {
				_ = n.DiscordClient.UpdateVoiceState(context.TODO(), player.GuildID(), nil, false, true)
				return
			}

			nextTrackEncoded = nextTrack.Encoded
			_ = json.Unmarshal([]byte(nextTrack.UserData), &nextTrackUserData)

		case database.QueueTypeRepeatTrack:
			nextTrackEncoded = event.Track.Encoded
			_ = event.Track.UserData.Unmarshal(&nextTrackUserData)

		case database.QueueTypeRepeatQueue:
			if _, err := n.DJ.Database.CreateTrack(queue.ID, event.Track); err != nil {
				log.Err(err).Uint("queue", queue.ID).Msg("Failed to create track")
				return
			}

			nextTrack, err := n.DJ.Database.SkipTracks(queue.ID, 1)
			if err != nil {
				_ = n.DiscordClient.UpdateVoiceState(context.TODO(), player.GuildID(), nil, false, true)
				return
			}

			nextTrackEncoded = nextTrack.Encoded
			_ = json.Unmarshal([]byte(nextTrack.UserData), &nextTrackUserData)
		}

		if err := player.Update(context.TODO(), lavalink.WithEncodedTrack(nextTrackEncoded), lavalink.WithTrackUserData(nextTrackUserData)); err != nil {
			log.Err(err).Msg("Failed to play track")
			continue
		}
		return
	}
}

func (n *Node) onTrackStuck(_ disgolink.Player, event lavalink.TrackStuckEvent) {
	log.Warn().Str("node", n.ID().String()).Any("event", event).Msg("Track stuck")
}
