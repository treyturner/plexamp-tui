package ui

import (
	"plexamp-tui/internal/config"

	"github.com/charmbracelet/bubbles/list"
)

type modelOption func(*model)

func testModel(options ...modelOption) model {
	m := model{
		panelMode: panelModePlayback,
	}
	for _, option := range options {
		option(&m)
	}
	return m
}

func withStatus(status string) modelOption {
	return func(m *model) {
		m.status = status
	}
}

func withSelectedPlayer(address string, cfg *config.Config) modelOption {
	return func(m *model) {
		m.selected = address
		m.config = cfg
	}
}

func withPlaybackFavorites(favorites ...config.FavoriteItem) modelOption {
	return func(m *model) {
		m.playbackConfig = &config.Favorites{Items: favorites}
		m.playbackList = list.New(favoriteListItems(favorites), list.NewDefaultDelegate(), 0, 0)
	}
}

func testFavorite(name string, itemType favoriteType, metadataKey string) config.FavoriteItem {
	return config.FavoriteItem{Name: name, Type: string(itemType), MetadataKey: metadataKey}
}
