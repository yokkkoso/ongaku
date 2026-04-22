-- +goose Up
CREATE TYPE queue_type AS ENUM ('normal', 'repeat_track', 'repeat_queue');

CREATE TABLE queues
(
	id                 SERIAL PRIMARY KEY,
	node_id            BIGINT     NOT NULL,
	guild_id           BIGINT     NOT NULL,
	channel_id         BIGINT     NOT NULL,
	lavalink_node_name TEXT       NOT NULL,
	type               queue_type NOT NULL
);

CREATE TABLE queue_tracks
(
	id         SERIAL PRIMARY KEY,
	queue_id   INTEGER                  NOT NULL REFERENCES queues (id) ON DELETE CASCADE ON UPDATE NO ACTION,
	encoded    TEXT                     NOT NULL,
	user_data  TEXT                     NOT NULL,
	created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (queue_id) REFERENCES queues (id)
);

CREATE INDEX idx_queue_tracks_queue_id ON queue_tracks (queue_id);
CREATE INDEX idx_queues_guild_id ON queues (guild_id);
CREATE INDEX idx_queues_channel_id ON queues (channel_id);


-- +goose Down
DROP INDEX IF EXISTS idx_queues_channel_id;
DROP INDEX IF EXISTS idx_queues_guild_id;
DROP INDEX IF EXISTS idx_queue_tracks_queue_id;
DROP TABLE IF EXISTS queue_tracks;
DROP TABLE IF EXISTS queues;
DROP TYPE IF EXISTS queue_type;
