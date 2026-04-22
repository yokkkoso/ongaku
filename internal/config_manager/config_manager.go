package config_manager

import (
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type Manager struct {
	cfg *Config
	mu  sync.RWMutex
}

func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.cfg
}

func newConfigManager() (*Manager, error) {
	viper.SetConfigFile("configs/config.toml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	manager := &Manager{
		cfg: &cfg,
	}

	viper.WatchConfig()
	viper.OnConfigChange(
		func(e fsnotify.Event) {
			var newCfg Config

			if err := viper.Unmarshal(&newCfg); err != nil {
				return
			}

			manager.mu.Lock()
			manager.cfg = &newCfg
			manager.mu.Unlock()
		},
	)

	return manager, nil
}

var (
	configManager *Manager
	once          sync.Once
)

func GetConfigManager() *Manager {
	once.Do(
		func() {
			var err error
			configManager, err = newConfigManager()
			if err != nil {
				panic(err)
			}
		},
	)

	return configManager
}
