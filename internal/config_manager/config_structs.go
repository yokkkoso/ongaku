package config_manager

type Config struct {
	DJToken       string               `mapstructure:"dj_token"`
	Color         int                  `mapstructure:"color"`
	SyncCommands  bool                 `mapstructure:"sync_commands"`
	IsDev         bool                 `mapstructure:"is_dev"`
	LogChannelID  string               `mapstructure:"log_channel_id"`
	Database      DatabaseConfig       `mapstructure:"database"`
	DiscordNodes  []DiscordNodeConfig  `mapstructure:"discord_nodes"`
	LavalinkNodes []LavalinkNodeConfig `mapstructure:"lavalink_nodes"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
}

type DiscordNodeConfig struct {
	Token string `mapstructure:"token"`
}

type LavalinkNodeConfig struct {
	Name     string `mapstructure:"name"`
	Address  string `mapstructure:"address"`
	Password string `mapstructure:"password"`
	Secure   bool   `mapstructure:"secure"`
}
