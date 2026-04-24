package core

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/json"
	"github.com/disgoorg/snowflake/v2"
	"github.com/rs/zerolog/log"
	"github.com/yokkkoso/ongaku/internal/database"
	"github.com/yokkkoso/ongaku/internal/utils"
	"github.com/yokkkoso/ongaku/internal/utils/array"
)

func NewDJ() *DJ {
	dj := &DJ{
		nodes: make([]*Node, 0),
	}

	return dj
}

type DJ struct {
	Client   *bot.Client
	Database *database.Database
	nodes    []*Node
}

var nodeMutex sync.Mutex

func (dj *DJ) AddNode(n *Node) {
	dj.nodes = append(dj.nodes, n)
}

func (dj *DJ) GetNodeByID(ID snowflake.ID) *Node {
	node, ok := array.Find(
		dj.nodes, func(n *Node) bool {
			return n.ID() == ID
		},
	)

	if ok {
		return node
	}

	return nil
}

func (dj *DJ) GetNode(guildID snowflake.ID, channelID snowflake.ID) *Node {
	for _, n := range dj.nodes {
		nodeVoiceState, ok := n.DiscordClient.Caches.VoiceState(guildID, n.ID())

		if !ok || nodeVoiceState.ChannelID == nil {
			continue
		}

		if *nodeVoiceState.ChannelID == channelID {
			return n
		}
	}

	return nil
}

func (dj *DJ) GetFreeNode(guildID snowflake.ID, channelID snowflake.ID) *Node {
	nodeMutex.Lock()
	defer nodeMutex.Unlock()

	for _, n := range dj.nodes {
		if n.IsPreTaken.Load() {
			continue
		}

		_, ok := n.DiscordClient.Caches.Guild(guildID)

		if !ok {
			continue
		}

		nodeVoiceState, ok := n.DiscordClient.Caches.VoiceState(guildID, n.ID())

		if ok && nodeVoiceState.ChannelID != nil && *nodeVoiceState.ChannelID != channelID {
			continue
		}

		n.IsPreTaken.Store(true)

		return n
	}

	return nil
}

func (dj *DJ) onReady(e *events.Ready) {
	user, _ := e.Client().Caches.SelfUser()

	log.Info().Msgf("DJ %s is ready", user.Tag())

	if err := e.Client().SetPresence(
		context.TODO(),
		gateway.WithOnlineStatus(discord.OnlineStatusOnline),
		gateway.WithListeningActivity(fmt.Sprintf("%d %s", len(dj.nodes), utils.GetDeclensionWord(len(dj.nodes), [3]string{"бота", "ботов", "ботов"}))),
	); err != nil {
		log.Err(err).Msg("Failed to set DJ presence")
	}
}

func (dj *DJ) InitDiscordClient(token string) {
	var err error
	if dj.Client, err = disgo.New(
		token,
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(
				gateway.IntentGuilds,
				gateway.IntentGuildMembers,
				gateway.IntentGuildVoiceStates,
			),
		),
		bot.WithCacheConfigOpts(
			cache.WithCaches(
				cache.FlagGuilds,
				cache.FlagChannels,
				cache.FlagMembers,
				cache.FlagVoiceStates,
			),
		),
		bot.WithRestConfigOpts(
			rest.WithDefaultAllowedMentions(discord.AllowedMentions{Parse: []discord.AllowedMentionType{}}),
		),
		bot.WithEventManagerConfigOpts(bot.WithAsyncEventsEnabled()),
		bot.WithEventListenerFunc(dj.onReady),
		bot.WithMemberChunkingFilter(bot.MemberChunkingFilterAll),
	); err != nil {
		log.Fatal().Err(err).Msg("Failed to start bot")
	}
}

func (dj *DJ) InitPreviousPlayers() {
	queues, _ := dj.Database.GetQueuesWithFirstTrack()

	for _, queue := range queues {
		node := dj.GetNodeByID(queue.NodeID)
		if node == nil {
			_, _ = dj.Database.DeleteQueue(queue.ID)
			continue
		}

		if len(queue.Tracks) <= 0 {
			_, _ = dj.Database.DeleteQueue(queue.ID)
			continue
		}

		track, _ := dj.Database.SkipTracks(queue.ID, 1)

		if track == nil {
			_, _ = dj.Database.DeleteQueue(queue.ID)
			continue
		}

		audioChannel, ok := dj.Client.Caches.GuildAudioChannel(queue.ChannelID)

		if !ok {
			_, _ = dj.Database.DeleteQueue(queue.ID)
			continue
		}

		members := dj.Client.Caches.AudioChannelMembers(audioChannel)

		if len(members) <= 0 {
			_, _ = dj.Database.DeleteQueue(queue.ID)
			continue
		}

		_ = node.DiscordClient.UpdateVoiceState(
			context.TODO(),
			queue.GuildID,
			&queue.ChannelID,
			false,
			true,
		)

		var userData database.TrackUserData
		_ = json.Unmarshal([]byte(track.UserData), &userData)

		_ = node.LavalinkClient.Player(queue.GuildID).Update(
			context.TODO(),
			lavalink.WithEncodedTrack(track.Encoded),
			lavalink.WithTrackUserData(userData),
			lavalink.WithVolume(75),
		)
	}
}

func (dj *DJ) InitDatabase() {
	var err error

	if dj.Database, err = database.InitDatabase(dj.Client); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}

	log.Info().Msg("Database connected")
}

func (dj *DJ) StartAndBlock() {
	if err := dj.Client.OpenGateway(context.TODO()); err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Discord gateway")
	}

	for _, node := range dj.nodes {
		if err := node.DiscordClient.OpenGateway(context.TODO()); err != nil {
			log.Err(err).Str("botID", node.DiscordClient.ApplicationID.String()).Msg("Failed to connect to Discord gateway")
		}
	}

	defer func() {
		log.Info().Msg("Shutting down...")
		dj.Client.Close(context.TODO())
		for _, node := range dj.nodes {
			node.DiscordClient.Close(context.TODO())
		}
	}()

	dj.InitPreviousPlayers()

	log.Info().Msg("Bot is running. Press CTRL+C to exit")
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM)
	<-s
}
