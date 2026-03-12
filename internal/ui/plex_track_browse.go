package ui

import (
	"fmt"

	"plexamp-tui/internal/plex"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type tracksFetchedMsg struct {
	tracks     []plex.PlexTrack
	context    string
	requestKey string
	err        error
}

type trackPlaybackMsg struct {
	success   bool
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
	m.panelMode = "plex-album-tracks"
	m.status = fmt.Sprintf("Loading tracks for %s...", albumTitle)
	m.currentAlbumKey = albumRatingKey
	m.currentAlbumName = albumTitle

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false

	items := []list.Item{trackItem{title: "Loading tracks..."}}
	m.trackList = list.New(items, delegate, 0, 0)
	m.trackList.Title = fmt.Sprintf("Tracks - %s", albumTitle)
	m.trackList.SetShowFilter(true)
	m.trackList.SetFilteringEnabled(true)
	m.trackList.Styles.Title = titleStyle
	m.trackList.Styles.PaginationStyle = paginationStyle
	m.trackList.Styles.HelpStyle = helpStyle

	if m.width > 0 && m.height > 0 {
		m.trackList.SetSize(m.width/2-4, m.height-4)
	}
}

func (m *model) initPlaylistTrackBrowse(playlistTitle, playlistRatingKey string) {
	m.panelMode = "plex-playlist-tracks"
	m.status = fmt.Sprintf("Loading tracks for %s...", playlistTitle)
	m.currentPlaylistKey = playlistRatingKey
	m.currentPlaylistName = playlistTitle

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false

	items := []list.Item{trackItem{title: "Loading tracks..."}}
	m.trackList = list.New(items, delegate, 0, 0)
	m.trackList.Title = fmt.Sprintf("Tracks - %s", playlistTitle)
	m.trackList.SetShowFilter(true)
	m.trackList.SetFilteringEnabled(true)
	m.trackList.Styles.Title = titleStyle
	m.trackList.Styles.PaginationStyle = paginationStyle
	m.trackList.Styles.HelpStyle = helpStyle

	if m.width > 0 && m.height > 0 {
		m.trackList.SetSize(m.width/2-4, m.height-4)
	}
}

func (m *model) fetchAlbumTracksCmd(albumRatingKey string) tea.Cmd {
	log.Debug("Fetching album tracks...")
	footerHeight := 3
	availableHeight := m.height - footerHeight - 5
	m.trackList.SetSize(m.width/2-4, availableHeight)

	if m.config == nil {
		return func() tea.Msg {
			return tracksFetchedMsg{
				context:    "album",
				requestKey: albumRatingKey,
				err:        fmt.Errorf("no config available"),
			}
		}
	}

	token := plexClient.GetPlexToken()
	if token == "" {
		return func() tea.Msg {
			return tracksFetchedMsg{
				context:    "album",
				requestKey: albumRatingKey,
				err:        fmt.Errorf("no Plex token found - run with --auth flag"),
			}
		}
	}

	serverAddr := m.config.PlexServerAddr
	return func() tea.Msg {
		tracks, err := plexClient.FetchAlbumTracks(serverAddr, albumRatingKey, token)
		return tracksFetchedMsg{
			tracks:     tracks,
			context:    "album",
			requestKey: albumRatingKey,
			err:        err,
		}
	}
}

func (m *model) fetchPlaylistTracksCmd(playlistRatingKey string) tea.Cmd {
	log.Debug("Fetching playlist tracks...")
	footerHeight := 3
	availableHeight := m.height - footerHeight - 5
	m.trackList.SetSize(m.width/2-4, availableHeight)

	if m.config == nil {
		return func() tea.Msg {
			return tracksFetchedMsg{
				context:    "playlist",
				requestKey: playlistRatingKey,
				err:        fmt.Errorf("no config available"),
			}
		}
	}

	token := plexClient.GetPlexToken()
	if token == "" {
		return func() tea.Msg {
			return tracksFetchedMsg{
				context:    "playlist",
				requestKey: playlistRatingKey,
				err:        fmt.Errorf("no Plex token found - run with --auth flag"),
			}
		}
	}

	serverAddr := m.config.PlexServerAddr
	return func() tea.Msg {
		tracks, err := plexClient.FetchPlaylistTracks(serverAddr, playlistRatingKey, token)
		return tracksFetchedMsg{
			tracks:     tracks,
			context:    "playlist",
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

	serverIP := m.selected
	serverID := m.config.ServerID
	shuffle := m.shuffle

	return func() tea.Msg {
		err := PlayMetadata(serverIP, serverID, ratingKey, shuffle)
		if err != nil {
			return trackPlaybackMsg{
				success:   false,
				requestID: requestID,
				ratingKey: ratingKey,
				err:       err,
			}
		}
		return trackPlaybackMsg{
			success:   true,
			requestID: requestID,
			ratingKey: ratingKey,
		}
	}
}

func (m *model) handleTrackBrowseUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Debug(fmt.Sprintf("handleTrackBrowseUpdate received message: %T", msg))

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
					log.Debug("Ignoring track playback for item without rating key")
					return m, nil
				}
				log.Debug(fmt.Sprintf("Playing track: %s (ratingKey: %s)", selected.title, selected.ratingKey))
				m.lastCommand = fmt.Sprintf("Playing %s", selected.title)
				m.trackPlaybackReqID++
				requestID := m.trackPlaybackReqID
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
		log.Debug(fmt.Sprintf(
			"tracksFetchedMsg received with %d tracks, context=%s, requestKey=%s, error=%v",
			len(msg.tracks), msg.context, msg.requestKey, msg.err),
		)

		switch msg.context {
		case "album":
			// Ignore stale/mismatched fetches so late responses cannot overwrite the active browse list.
			if m.panelMode != "plex-album-tracks" || msg.requestKey != m.currentAlbumKey {
				log.Debug(fmt.Sprintf(
					"Ignoring stale album track response (requestKey=%s, currentAlbumKey=%s, panelMode=%s)",
					msg.requestKey, m.currentAlbumKey, m.panelMode),
				)
				return m, nil
			}
		case "playlist":
			// Ignore stale/mismatched fetches so late responses cannot overwrite the active browse list.
			if m.panelMode != "plex-playlist-tracks" || msg.requestKey != m.currentPlaylistKey {
				log.Debug(fmt.Sprintf(
					"Ignoring stale playlist track response (requestKey=%s, currentPlaylistKey=%s, panelMode=%s)",
					msg.requestKey, m.currentPlaylistKey, m.panelMode),
				)
				return m, nil
			}
		default:
			log.Debug(fmt.Sprintf("Ignoring track response with unknown context: %s", msg.context))
			return m, nil
		}

		if msg.err != nil {
			errMsg := fmt.Sprintf("Error fetching tracks: %v", msg.err)
			m.status = errMsg
			log.Debug(errMsg)
			return m, nil
		}

		var items []list.Item
		for _, track := range msg.tracks {
			display := track.Title
			filter := track.Title
			if msg.context == "album" && track.Index > 0 {
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

		filterState := m.trackList.FilterState()
		filterValue := m.trackList.FilterValue()

		m.trackList.SetItems(items)
		m.trackList.ResetSelected()

		if filterState == list.Filtering {
			m.trackList.ResetFilter()
			m.trackList.FilterInput.SetValue(filterValue)
		}

		m.status = fmt.Sprintf("Loaded %d tracks", len(msg.tracks))
		return m, tea.Batch(tea.ClearScreen, func() tea.Msg { return nil })

	}

	var listCmd tea.Cmd
	m.trackList, listCmd = m.trackList.Update(msg)
	return m, listCmd
}
