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
	log.Debug(fmt.Sprintf("Triggering radio playback for %s", item.Name))
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
	log.Debug(fmt.Sprintf("Triggering playback for %s", item.Name))
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
	switch item.Type {
	case "artist":
		log.Debug(fmt.Sprintf("Playing artist: %s", item.Name))
		return m.playArtistCmd(item.MetadataKey)
	case "album":
		log.Debug(fmt.Sprintf("Playing album: %s", item.Name))
		return m.playAlbumCmd(item.MetadataKey)
	case "playlist":
		log.Debug(fmt.Sprintf("Playing playlist: %s", item.Name))
		return m.playPlaylistCmd(item.MetadataKey)
	default:
		log.Debug(fmt.Sprintf("Unknown type: %s", item.Type))
		return func() tea.Msg {
			return playbackTriggeredMsg{success: false, err: fmt.Errorf("unknown type: %s", item.Type)}
		}
	}
}

func (m *model) findFavoriteItem(selected item) (config.FavoriteItem, bool) {
	if m.playbackConfig == nil {
		return config.FavoriteItem{}, false
	}

	// Primary match: metadata key + type, fallback by name+type for legacy entries.
	for _, pb := range m.playbackConfig.Items {
		if pb.MetadataKey != "" && pb.MetadataKey == selected.MetadataKey && pb.Type == selected.Type {
			return pb, true
		}
	}
	for _, pb := range m.playbackConfig.Items {
		if pb.Name == string(selected.Name) && pb.Type == selected.Type {
			return pb, true
		}
	}
	return config.FavoriteItem{}, false
}

func (m *model) openFavoriteItem(item config.FavoriteItem) tea.Cmd {
	switch item.Type {
	case "artist":
		m.lastCommand = fmt.Sprintf("Viewing %s", item.Name)
		m.initArtistAlbumBrowse(artistItem{
			title:     item.Name,
			ratingKey: item.MetadataKey,
		})
		return m.fetchArtistAlbumsCmd(item.MetadataKey)
	case "album":
		m.lastCommand = fmt.Sprintf("Viewing %s", item.Name)
		m.trackReturnMode = "playback"
		m.initAlbumTrackBrowse(item.Name, item.MetadataKey)
		return m.fetchAlbumTracksCmd(item.MetadataKey)
	case "playlist":
		m.lastCommand = fmt.Sprintf("Viewing %s", item.Name)
		m.trackReturnMode = "playback"
		m.initPlaylistTrackBrowse(item.Name, item.MetadataKey)
		return m.fetchPlaylistTracksCmd(item.MetadataKey)
	default:
		m.status = fmt.Sprintf("Unknown favorite type: %s", item.Type)
		return nil
	}
}

func (m *model) addRemoveFavorite(name string, k string, t string) (tea.Model, tea.Cmd) {
	log.Debug(fmt.Sprintf("Toggling favorite for %s", name))
	favSet := m.getCurrentFavSet()
	if _, exists := favSet[k]; exists {
		log.Debug(fmt.Sprintf("Removing favorite: %s", name))
		// Delete selected playback item
		index := m.playbackList.Index()
		m.deletePlaybackItem(index)
		return m, nil
	}
	log.Debug(fmt.Sprintf("Adding favorite: %s", name))
	m.savePlaybackItem(name, k, t)
	return m, nil
}

func (m *model) getCurrentFavSet() map[string]struct{} {
	favSet := make(map[string]struct{})
	for _, pItem := range m.playbackList.Items() {
		pItem := pItem.(item)
		favSet[pItem.GetMetadataKey()] = struct{}{}
	}
	return favSet
}
