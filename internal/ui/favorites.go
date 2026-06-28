package ui

import (
	"fmt"

	"plexamp-tui/internal/config"

	tea "github.com/charmbracelet/bubbletea"
)

// =====================
// Playback Trigger
// =====================

func (m *model) triggerFavoriteRadioPlayback(item config.FavoriteItem) tea.Cmd {
	m.debug("Triggering radio playback for %s", item.Name)
	if m.selected == "" {
		return func() tea.Msg {
			return playbackTriggeredMsg{success: false, err: fmt.Errorf("no server selected")}
		}
	}

	if m.config == nil {
		return func() tea.Msg {
			return playbackTriggeredMsg{success: false, err: fmt.Errorf("no config available")}
		}
	}

	m.status = fmt.Sprintf("Starting radio for %s...", item.Name)
	m.lastCommand = fmt.Sprintf("Playing radio for %s", item.Name)
	return m.playArtistRadioCmd(item.MetadataKey)
}

func (m *model) triggerFavoritePlayback(item config.FavoriteItem) tea.Cmd {
	m.debug("Triggering playback for %s", item.Name)
	if m.selected == "" {
		return func() tea.Msg {
			return playbackTriggeredMsg{success: false, err: fmt.Errorf("no server selected")}
		}
	}

	if m.config == nil {
		return func() tea.Msg {
			return playbackTriggeredMsg{success: false, err: fmt.Errorf("no config available")}
		}
	}

	m.status = fmt.Sprintf("Starting playback for %s...", item.Name)
	m.lastCommand = fmt.Sprintf("Playing %s", item.Name)
	switch favoriteType(item.Type) {
	case favoriteTypeArtist:
		m.debug("Playing artist: %s", item.Name)
		return m.playArtistCmd(item.MetadataKey)
	case favoriteTypeAlbum:
		m.debug("Playing album: %s", item.Name)
		return m.playAlbumCmd(item.MetadataKey)
	case favoriteTypePlaylist:
		m.debug("Playing playlist: %s", item.Name)
		return m.playPlaylistCmd(item.MetadataKey)
	default:
		m.debug("Unknown type: %s", item.Type)
		return func() tea.Msg {
			return playbackTriggeredMsg{success: false, err: fmt.Errorf("unknown type: %s", item.Type)}
		}
	}
}

func (m *model) findFavoriteItem(selected item) (config.FavoriteItem, bool) {
	return m.favoritesController().findSelected(selected)
}

func (m *model) openFavoriteItem(item config.FavoriteItem) tea.Cmd {
	if item.MetadataKey == "" {
		m.debug("Cannot open favorite %s (%s): missing metadata key", item.Name, item.Type)
		m.status = fmt.Sprintf("Cannot open %s: missing metadata key", item.Name)
		return nil
	}

	switch favoriteType(item.Type) {
	case favoriteTypeArtist:
		m.lastCommand = fmt.Sprintf("Viewing %s", item.Name)
		m.initArtistAlbumBrowse(artistItem{
			title:     item.Name,
			ratingKey: item.MetadataKey,
		})
		return m.fetchArtistAlbumsCmd(item.MetadataKey)
	case favoriteTypeAlbum:
		m.lastCommand = fmt.Sprintf("Viewing %s", item.Name)
		m.trackReturnMode = panelModePlayback
		m.initAlbumTrackBrowse(item.Name, item.MetadataKey)
		return m.fetchAlbumTracksCmd(item.MetadataKey)
	case favoriteTypePlaylist:
		m.lastCommand = fmt.Sprintf("Viewing %s", item.Name)
		m.trackReturnMode = panelModePlayback
		m.initPlaylistTrackBrowse(item.Name, item.MetadataKey)
		return m.fetchPlaylistTracksCmd(item.MetadataKey)
	default:
		m.status = fmt.Sprintf("Unknown favorite type: %s", item.Type)
		return nil
	}
}

func (m *model) addRemoveFavorite(name string, k string, t favoriteType) (tea.Model, tea.Cmd) {
	m.debug("Toggling favorite for %s", name)
	added, err := m.favoritesController().toggle(name, k, t)
	if err != nil {
		m.status = "Error toggling favorite: " + err.Error()
		m.lastCommand = "Favorite Toggle Failed"
		return m, nil
	}
	if added {
		m.debug("Added favorite: %s", name)
	} else {
		m.debug("Removed favorite: %s", name)
	}
	return m, nil
}
