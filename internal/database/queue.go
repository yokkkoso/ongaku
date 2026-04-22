package database

import (
	"time"

	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type QueueType string

const (
	QueueTypeNormal      QueueType = "normal"
	QueueTypeRepeatTrack QueueType = "repeat_track"
	QueueTypeRepeatQueue QueueType = "repeat_queue"
)

func (q QueueType) NextType() QueueType {
	switch q {
	case QueueTypeNormal:
		return QueueTypeRepeatQueue
	case QueueTypeRepeatQueue:
		return QueueTypeRepeatTrack
	case QueueTypeRepeatTrack:
		return QueueTypeNormal
	default:
		return QueueTypeNormal
	}
}

func (q QueueType) Emoji() string {
	switch q {
	case QueueTypeNormal:
		return "🔁"
	case QueueTypeRepeatQueue:
		return "🔁"
	case QueueTypeRepeatTrack:
		return "🔂"
	default:
		return "🔁"
	}
}

type TrackUserData struct {
	OrderedByID      snowflake.ID `json:"ordered_by_id"`
	OrderedByTag     string       `json:"ordered_by_tag"`
	InteractionToken string       `json:"interaction_token"`
}

type Queue struct {
	ID               uint `gorm:"primaryKey;autoIncrement"`
	NodeID           snowflake.ID
	GuildID          snowflake.ID
	ChannelID        snowflake.ID
	LavalinkNodeName string
	Type             QueueType
	Tracks           []QueueTrack
}

type QueueTrack struct {
	ID        uint `gorm:"primaryKey;autoIncrement"`
	QueueID   uint
	Encoded   string
	UserData  string
	CreatedAt time.Time
}

func (db *Database) SkipTracks(queueID uint, amount int) (*QueueTrack, error) {
	var tracks []QueueTrack

	err := db.DB.Where("queue_id = ?", queueID).
		Order("created_at ASC").
		Find(&tracks).Error

	if err != nil {
		return nil, err
	}

	if len(tracks) == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	if amount > len(tracks) {
		amount = len(tracks)
	}

	var idsToDelete []uint
	for i := 0; i < amount; i++ {
		idsToDelete = append(idsToDelete, tracks[i].ID)
	}

	if len(idsToDelete) > 0 {
		err = db.DB.Delete(&QueueTrack{}, idsToDelete).Error
		if err != nil {
			return nil, err
		}
	}

	if amount <= len(tracks) {
		return &tracks[amount-1], nil
	}

	return nil, gorm.ErrRecordNotFound
}

func (db *Database) FindOrCreateQueue(lavalinkNodeName string, nodeID, guildID, channelID snowflake.ID) (Queue, error) {
	var queue Queue

	err := db.DB.Where(
		Queue{
			NodeID:           nodeID,
			GuildID:          guildID,
			ChannelID:        channelID,
			LavalinkNodeName: lavalinkNodeName,
		},
	).Preload("Tracks").FirstOrCreate(
		&queue, Queue{
			NodeID:           nodeID,
			GuildID:          guildID,
			ChannelID:        channelID,
			LavalinkNodeName: lavalinkNodeName,
			Type:             QueueTypeNormal,
		},
	).Error

	return queue, err
}

func (db *Database) UpdateChannel(nodeID, guildID, channelID snowflake.ID) error {
	var queue Queue

	err := db.DB.Where(
		Queue{
			NodeID:  nodeID,
			GuildID: guildID,
		},
	).First(&queue).Error

	if err != nil {
		return err
	}

	err = db.DB.Model(&queue).Update("channel_id", channelID).Error
	if err != nil {
		return err
	}

	return nil
}

func (db *Database) UpdateQueueType(nodeID, guildID snowflake.ID) (QueueType, error) {
	var queue Queue

	err := db.DB.Where(
		Queue{
			NodeID:  nodeID,
			GuildID: guildID,
		},
	).First(&queue).Error

	if err != nil {
		return QueueTypeNormal, err
	}

	nextType := queue.Type.NextType()

	err = db.DB.Model(&queue).Update("type", nextType).Error
	if err != nil {
		return QueueTypeNormal, err
	}

	return nextType, nil
}

func (db *Database) CreateTrack(queueID uint, track lavalink.Track) (QueueTrack, error) {
	queueTrack := QueueTrack{
		QueueID:  queueID,
		Encoded:  track.Encoded,
		UserData: string(track.UserData),
	}

	err := db.DB.Create(&queueTrack).Error

	return queueTrack, err
}

func (db *Database) GetQueue(guildID, nodeID snowflake.ID) (queue Queue, err error) {
	err = db.DB.
		Where("guild_id = ? and node_id = ?", guildID, nodeID).
		Preload(
			"Tracks",
			func(db *gorm.DB) *gorm.DB {
				return db.Order("created_at ASC")
			},
		).First(&queue).Error

	return
}

func (db *Database) GetQueuesWithFirstTrack() (queues []Queue, err error) {
	err = db.DB.
		Preload(
			"Tracks",
			func(db *gorm.DB) *gorm.DB {
				return db.Order("created_at ASC").Limit(1)
			},
		).
		Find(&queues).Error

	return
}

func (db *Database) DeleteQueue(queueID uint) (queue Queue, err error) {
	err = db.DB.
		Clauses(clause.Returning{}).
		Delete(&queue, queueID).Error
	return
}

func (db *Database) DeleteTrack(trackID string) (track QueueTrack, err error) {
	err = db.DB.
		Clauses(clause.Returning{}).
		Delete(&track, trackID).Error
	return
}

func (db *Database) DeleteQueueByNodeAndGuild(nodeID, guildID snowflake.ID) error {
	return db.DB.Where("node_id = ? AND guild_id = ?", nodeID, guildID).Delete(&Queue{}).Error
}
