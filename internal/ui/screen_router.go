package ui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) routeScreenKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch m.panelMode {
	case panelModeEdit:
		model, cmd := m.handleEditUpdate(msg)
		return model, cmd, true
	case panelModePlayback:
		cmd, handled := m.handlePlaybackKey(msg)
		return *m, cmd, handled
	case panelModePlexArtists:
		_, cmd := m.handleArtistBrowseUpdate(msg)
		return *m, cmd, true
	case panelModePlexAlbums:
		_, cmd := m.handleAlbumBrowseUpdate(msg)
		return *m, cmd, true
	case panelModePlexArtistAlbums:
		_, cmd := m.handleArtistAlbumBrowseUpdate(msg)
		return *m, cmd, true
	case panelModePlexAlbumTracks, panelModePlexPlaylistTracks:
		_, cmd := m.handleTrackBrowseUpdate(msg)
		return *m, cmd, true
	case panelModePlexPlaylists:
		_, cmd := m.handlePlaylistBrowseUpdate(msg)
		return *m, cmd, true
	case panelModePlexServers:
		_, cmd := m.handleServerBrowseUpdate(msg)
		return *m, cmd, true
	case panelModePlexPlayers:
		_, cmd := m.handlePlayerBrowseUpdate(msg)
		return *m, cmd, true
	default:
		return *m, nil, false
	}
}

func (m *model) handlePlaybackKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	if m.playbackList.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.playbackList, cmd = m.playbackList.Update(msg)
		return cmd, true
	}

	switch msg.String() {
	case "a":
		m.initEditMode(editModePlayback, -1)
		return nil, true
	case "e":
		index := m.playbackList.Index()
		m.initEditMode(editModePlayback, index)
		return nil, true
	case "d":
		index := m.playbackList.Index()
		if err := m.deletePlaybackItem(index); err != nil {
			m.status = "Error deleting favorite: " + err.Error()
			m.lastCommand = "Delete Failed"
		}
		return nil, true
	case "r":
		if selected, ok := m.playbackList.SelectedItem().(item); ok {
			if pb, found := m.findFavoriteItem(selected); found && favoriteType(pb.Type) == favoriteTypeArtist {
				return m.triggerFavoriteRadioPlayback(pb), true
			}
		}
		return nil, false
	case "enter":
		if selected, ok := m.playbackList.SelectedItem().(item); ok {
			if pb, found := m.findFavoriteItem(selected); found {
				return m.openFavoriteItem(pb), true
			}
		}
		return nil, true
	case "P":
		if selected, ok := m.playbackList.SelectedItem().(item); ok {
			if pb, found := m.findFavoriteItem(selected); found {
				return m.triggerFavoritePlayback(pb), true
			}
		}
		return nil, true
	default:
		return nil, false
	}
}

func (m *model) routeScreenMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case artistsFetchedMsg:
		if m.panelMode == panelModePlexArtists {
			_, cmd := m.handleArtistBrowseUpdate(msg)
			return *m, cmd
		}
	case albumsFetchedMsg:
		if m.panelMode == panelModePlexAlbums {
			_, cmd := m.handleAlbumBrowseUpdate(msg)
			return *m, cmd
		}
	case artistAlbumsFetchedMsg:
		if m.panelMode == panelModePlexArtistAlbums {
			_, cmd := m.handleArtistAlbumBrowseUpdate(msg)
			return *m, cmd
		}
	case tracksFetchedMsg:
		if m.panelMode == panelModePlexAlbumTracks || m.panelMode == panelModePlexPlaylistTracks {
			_, cmd := m.handleTrackBrowseUpdate(msg)
			return *m, cmd
		}
	case playlistsFetchedMsg:
		if m.panelMode == panelModePlexPlaylists {
			_, cmd := m.handlePlaylistBrowseUpdate(msg)
			return *m, cmd
		}
	case serversFetchedMsg:
		if m.panelMode == panelModePlexServers {
			_, cmd := m.handleServerBrowseUpdate(msg)
			return *m, cmd
		}
	case playersFetchedMsg:
		if m.panelMode == panelModePlexPlayers {
			_, cmd := m.handlePlayerBrowseUpdate(msg)
			return *m, cmd
		}
	}
	return *m, nil
}

func (m *model) updateActiveList(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch m.panelMode {
	case panelModePlayback:
		m.playbackList, cmd = m.playbackList.Update(msg)
	case panelModePlexArtists:
		m.artistList, cmd = m.artistList.Update(msg)
	case panelModePlexArtistAlbums:
		m.artistAlbumList, cmd = m.artistAlbumList.Update(msg)
	case panelModePlexAlbums:
		m.albumList, cmd = m.albumList.Update(msg)
	case panelModePlexAlbumTracks, panelModePlexPlaylistTracks:
		m.trackList, cmd = m.trackList.Update(msg)
	case panelModePlexPlaylists:
		m.playlistList, cmd = m.playlistList.Update(msg)
	case panelModePlexServers:
		m.serverList, cmd = m.serverList.Update(msg)
	case panelModePlexPlayers:
		m.playerList, cmd = m.playerList.Update(msg)
	}
	return cmd
}

func (m model) activePanelView() string {
	switch m.panelMode {
	case panelModePlayback:
		return m.playbackList.View()
	case panelModePlexArtists:
		return m.artistList.View()
	case panelModePlexArtistAlbums:
		return m.artistAlbumList.View()
	case panelModePlexAlbums:
		return m.albumList.View()
	case panelModePlexAlbumTracks, panelModePlexPlaylistTracks:
		return m.trackList.View()
	case panelModePlexPlaylists:
		return m.playlistList.View()
	case panelModePlexServers:
		return m.serverList.View()
	case panelModePlexPlayers:
		return m.playerList.View()
	default:
		return ""
	}
}
