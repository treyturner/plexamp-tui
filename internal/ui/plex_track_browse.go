package ui

import (
	"fmt"

	"plexamp-tui/internal/plex"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type tracksFetchedMsg struct {
	tracks  []plex.PlexTrack
	context string
	err     error
}

type trackPlaybackMsg struct {
	success bool
	err     error
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
			return tracksFetchedMsg{context: "album", err: fmt.Errorf("no config available")}
		}
	}

	token := plexClient.GetPlexToken()
	if token == "" {
		return func() tea.Msg {
			return tracksFetchedMsg{context: "album", err: fmt.Errorf("no Plex token found - run with --auth flag")}
		}
	}

	serverAddr := m.config.PlexServerAddr
	return func() tea.Msg {
		tracks, err := plexClient.FetchAlbumTracks(serverAddr, albumRatingKey, token)
		return tracksFetchedMsg{tracks: tracks, context: "album", err: err}
	}
}

func (m *model) fetchPlaylistTracksCmd(playlistRatingKey string) tea.Cmd {
	log.Debug("Fetching playlist tracks...")
	footerHeight := 3
	availableHeight := m.height - footerHeight - 5
	m.trackList.SetSize(m.width/2-4, availableHeight)

	if m.config == nil {
		return func() tea.Msg {
			return tracksFetchedMsg{context: "playlist", err: fmt.Errorf("no config available")}
		}
	}

	token := plexClient.GetPlexToken()
	if token == "" {
		return func() tea.Msg {
			return tracksFetchedMsg{context: "playlist", err: fmt.Errorf("no Plex token found - run with --auth flag")}
		}
	}

	serverAddr := m.config.PlexServerAddr
	return func() tea.Msg {
		tracks, err := plexClient.FetchPlaylistTracks(serverAddr, playlistRatingKey, token)
		return tracksFetchedMsg{tracks: tracks, context: "playlist", err: err}
	}
}

func (m *model) playTrackCmd(ratingKey string) tea.Cmd {
	if m.selected == "" {
		return func() tea.Msg {
			return trackPlaybackMsg{success: false, err: fmt.Errorf("no server selected")}
		}
	}

	if m.config == nil {
		return func() tea.Msg {
			return trackPlaybackMsg{success: false, err: fmt.Errorf("no config available")}
		}
	}

	serverIP := m.selected
	serverID := m.config.ServerID
	shuffle := m.shuffle

	return func() tea.Msg {
		err := PlayMetadata(serverIP, serverID, ratingKey, shuffle)
		if err != nil {
			return trackPlaybackMsg{success: false, err: err}
		}
		return trackPlaybackMsg{success: true}
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
				log.Debug(fmt.Sprintf("Playing track: %s (ratingKey: %s)", selected.title, selected.ratingKey))
				m.lastCommand = fmt.Sprintf("Playing %s", selected.title)
				return m, m.playTrackCmd(selected.ratingKey)
			}
			return m, nil

		default:
			if cmd, handled := m.handleControl(key); handled {
				return m, cmd
			}
		}

	case tracksFetchedMsg:
		log.Debug(fmt.Sprintf("tracksFetchedMsg received with %d tracks, error: %v", len(msg.tracks), msg.err))
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

	case trackPlaybackMsg:
		if msg.success {
			m.lastCommand = "Track Playback Started"
			m.status = "Playback triggered successfully"
		} else {
			m.lastCommand = "Playback Failed"
			m.status = fmt.Sprintf("Playback error: %v", msg.err)
		}
		return m, nil
	}

	var listCmd tea.Cmd
	m.trackList, listCmd = m.trackList.Update(msg)
	return m, listCmd
}
