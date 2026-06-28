package ui

import (
	"fmt"

	"plexamp-tui/internal/plex"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type tracksFetchedMsg struct {
	tracks     []plex.PlexTrack
	context    trackBrowseContext
	requestKey string
	err        error
}

type trackPlaybackMsg struct {
	success   bool
	selected  string
	requestID int
	ratingKey string
	err       error
}

type trackItem struct {
	title     string
	filter    string
	ratingKey string
}

func (i trackItem) Title() string       { return i.title }
func (i trackItem) Description() string { return "" }
func (i trackItem) FilterValue() string { return i.filter }

func (m *model) initAlbumTrackBrowse(albumTitle, albumRatingKey string) {
	m.panelMode = panelModePlexAlbumTracks
	m.status = fmt.Sprintf("Loading tracks for %s...", albumTitle)
	m.currentAlbumKey = albumRatingKey
	m.currentAlbumName = albumTitle

	items := []list.Item{trackItem{title: "Loading tracks..."}}
	m.trackList = newBrowseList(fmt.Sprintf("Tracks - %s", albumTitle), items, m.width, m.height)
}

func (m *model) initPlaylistTrackBrowse(playlistTitle, playlistRatingKey string) {
	m.panelMode = panelModePlexPlaylistTracks
	m.status = fmt.Sprintf("Loading tracks for %s...", playlistTitle)
	m.currentPlaylistKey = playlistRatingKey
	m.currentPlaylistName = playlistTitle

	items := []list.Item{trackItem{title: "Loading tracks..."}}
	m.trackList = newBrowseList(fmt.Sprintf("Tracks - %s", playlistTitle), items, m.width, m.height)
}

func (m *model) fetchAlbumTracksCmd(albumRatingKey string) tea.Cmd {
	m.debug("Fetching album tracks...")
	resizeBrowseListForFetch(&m.trackList, m.width, m.height)

	if m.config == nil {
		return func() tea.Msg {
			return tracksFetchedMsg{
				context:    trackBrowseContextAlbum,
				requestKey: albumRatingKey,
				err:        fmt.Errorf("no config available"),
			}
		}
	}

	token := m.deps.plexClient.GetPlexToken()
	if token == "" {
		return func() tea.Msg {
			return tracksFetchedMsg{
				context:    trackBrowseContextAlbum,
				requestKey: albumRatingKey,
				err:        fmt.Errorf("no Plex token found - run with --auth flag"),
			}
		}
	}

	serverAddr := m.config.PlexServerAddr
	return func() tea.Msg {
		tracks, err := m.deps.plexClient.FetchAlbumTracks(serverAddr, albumRatingKey, token)
		return tracksFetchedMsg{
			tracks:     tracks,
			context:    trackBrowseContextAlbum,
			requestKey: albumRatingKey,
			err:        err,
		}
	}
}

func (m *model) fetchPlaylistTracksCmd(playlistRatingKey string) tea.Cmd {
	m.debug("Fetching playlist tracks...")
	resizeBrowseListForFetch(&m.trackList, m.width, m.height)

	if m.config == nil {
		return func() tea.Msg {
			return tracksFetchedMsg{
				context:    trackBrowseContextPlaylist,
				requestKey: playlistRatingKey,
				err:        fmt.Errorf("no config available"),
			}
		}
	}

	token := m.deps.plexClient.GetPlexToken()
	if token == "" {
		return func() tea.Msg {
			return tracksFetchedMsg{
				context:    trackBrowseContextPlaylist,
				requestKey: playlistRatingKey,
				err:        fmt.Errorf("no Plex token found - run with --auth flag"),
			}
		}
	}

	serverAddr := m.config.PlexServerAddr
	return func() tea.Msg {
		tracks, err := m.deps.plexClient.FetchPlaylistTracks(serverAddr, playlistRatingKey, token)
		return tracksFetchedMsg{
			tracks:     tracks,
			context:    trackBrowseContextPlaylist,
			requestKey: playlistRatingKey,
			err:        err,
		}
	}
}

func (m *model) playTrackCmd(ratingKey string, requestID int) tea.Cmd {
	if m.selected == "" {
		return func() tea.Msg {
			return trackPlaybackMsg{
				success:   false,
				requestID: requestID,
				ratingKey: ratingKey,
				err:       fmt.Errorf("no server selected"),
			}
		}
	}

	if m.config == nil {
		return func() tea.Msg {
			return trackPlaybackMsg{
				success:   false,
				requestID: requestID,
				ratingKey: ratingKey,
				err:       fmt.Errorf("no config available"),
			}
		}
	}
	if m.deps.plexClient == nil {
		return func() tea.Msg {
			return trackPlaybackMsg{
				success:   false,
				requestID: requestID,
				ratingKey: ratingKey,
				err:       fmt.Errorf("no Plex client available"),
			}
		}
	}

	serverIP := m.selected
	serverID := m.config.ServerID
	shuffle := m.shuffle
	plexClient := m.deps.plexClient

	return func() tea.Msg {
		err := plexClient.PlayMetadata(serverIP, serverID, ratingKey, shuffle)
		if err != nil {
			return trackPlaybackMsg{
				success:   false,
				selected:  serverIP,
				requestID: requestID,
				ratingKey: ratingKey,
				err:       err,
			}
		}
		return trackPlaybackMsg{
			success:   true,
			selected:  serverIP,
			requestID: requestID,
			ratingKey: ratingKey,
		}
	}
}

func (m *model) handleTrackBrowseUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.debug("handleTrackBrowseUpdate received message: %T", msg)

	if m.trackList.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.trackList, cmd = m.trackList.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		switch key {
		case "esc", "q":
			m.panelMode = m.trackReturnMode
			m.status = ""
			return m, nil

		case "enter":
			if selected, ok := m.trackList.SelectedItem().(trackItem); ok {
				if selected.ratingKey == "" {
					m.debug("Ignoring track playback for item without rating key")
					return m, nil
				}
				m.debug("Playing track: %s (ratingKey: %s)", selected.title, selected.ratingKey)
				m.lastCommand = fmt.Sprintf("Playing %s", selected.title)
				requestID := m.nextTrackPlaybackRequestID()
				m.beginPlaybackPendingForTrack("Loading track...", selected.ratingKey)
				return m, m.playTrackCmd(selected.ratingKey, requestID)
			}
			return m, nil

		default:
			if cmd, handled := m.handleControl(key); handled {
				return m, cmd
			}
		}

	case tracksFetchedMsg:
		m.debug(
			"tracksFetchedMsg received with %d tracks, context=%s, requestKey=%s, error=%v",
			len(msg.tracks), msg.context, msg.requestKey, msg.err,
		)

		switch msg.context {
		case trackBrowseContextAlbum:
			// Ignore stale/mismatched fetches so late responses cannot overwrite the active browse list.
			if m.panelMode != panelModePlexAlbumTracks || msg.requestKey != m.currentAlbumKey {
				m.debug(
					"Ignoring stale album track response (requestKey=%s, currentAlbumKey=%s, panelMode=%s)",
					msg.requestKey, m.currentAlbumKey, m.panelMode,
				)
				return m, nil
			}
		case trackBrowseContextPlaylist:
			// Ignore stale/mismatched fetches so late responses cannot overwrite the active browse list.
			if m.panelMode != panelModePlexPlaylistTracks || msg.requestKey != m.currentPlaylistKey {
				m.debug(
					"Ignoring stale playlist track response (requestKey=%s, currentPlaylistKey=%s, panelMode=%s)",
					msg.requestKey, m.currentPlaylistKey, m.panelMode,
				)
				return m, nil
			}
		default:
			m.debug("Ignoring track response with unknown context: %s", msg.context)
			return m, nil
		}

		if msg.err != nil {
			errMsg := fmt.Sprintf("Error fetching tracks: %v", msg.err)
			m.status = errMsg
			m.debug("%s", errMsg)
			return m, nil
		}

		var items []list.Item
		for _, track := range msg.tracks {
			display := track.Title
			filter := track.Title
			if msg.context == trackBrowseContextAlbum && track.Index > 0 {
				display = fmt.Sprintf("%02d. %s", track.Index, track.Title)
				filter = fmt.Sprintf("%02d %s", track.Index, track.Title)
			} else if track.GrandparentTitle != "" {
				display = fmt.Sprintf("%s - %s", track.GrandparentTitle, track.Title)
				if track.ParentTitle != "" {
					display = fmt.Sprintf("%s (%s)", display, track.ParentTitle)
				}
				filter = fmt.Sprintf("%s %s %s", track.Title, track.GrandparentTitle, track.ParentTitle)
			}

			items = append(items, trackItem{
				title:     display,
				filter:    filter,
				ratingKey: track.RatingKey,
			})
		}

		replaceBrowseListItems(&m.trackList, items)

		m.status = fmt.Sprintf("Loaded %d tracks", len(msg.tracks))
		return m, tea.Batch(tea.ClearScreen, func() tea.Msg { return nil })

	}

	var listCmd tea.Cmd
	m.trackList, listCmd = m.trackList.Update(msg)
	return m, listCmd
}
