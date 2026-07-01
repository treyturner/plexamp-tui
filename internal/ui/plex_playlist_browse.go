package ui

import (
	"fmt"
	"strings"

	"plexamp-tui/internal/plex"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// playlistItem represents a playlist in the list
type playlistItem struct {
	title     string
	artist    string
	year      string
	ratingKey string
}

// playlistsFetchedMsg is a message containing fetched playlists
type playlistsFetchedMsg struct {
	playlists []plex.PlexPlaylist
	err       error
}

// Title returns the playlist title
func (i playlistItem) Title() string {
	if strings.HasSuffix(i.title, " ★") {
		return fmt.Sprintf("%s - %s (%s) ★", strings.TrimSuffix(i.title, " ★"), i.artist, i.year)
	}
	return fmt.Sprintf("%s - %s (%s)", i.title, i.artist, i.year)
}

// Description returns the playlist description (empty for now)
func (i playlistItem) Description() string { return "" }

// FilterValue implements list.Item
func (i playlistItem) FilterValue() string {
	return i.title + " " + i.artist
}

func (p *playlistItem) ToggleFavorite() {
	// If title already has a star, remove it
	if strings.HasSuffix(p.title, " ★") {
		p.title = strings.TrimSuffix(p.title, " ★")
	} else {
		p.title = fmt.Sprintf("%s ★", p.title)
	}
}

// fetchPlaylistsCmd fetches playlists from the Plex server
func (m *model) fetchPlaylistsCmd() tea.Cmd {
	m.debug("Fetching playlists...")
	// ✅ Reapply sizing
	footerHeight := 3 // or dynamically measure your footer
	availableHeight := m.height - footerHeight - 5
	m.playlistList.SetSize(m.width/2-4, availableHeight)
	if m.config == nil {
		return func() tea.Msg {
			return playlistsFetchedMsg{err: fmt.Errorf("no config available")}
		}
	}

	token := m.deps.plexClient.GetPlexToken()
	if token == "" {
		return func() tea.Msg {
			return playlistsFetchedMsg{err: fmt.Errorf("no Plex token found - run with --auth flag")}
		}
	}

	serverAddr := m.config.PlexServerAddr

	return func() tea.Msg {
		playlists, err := m.deps.plexClient.FetchPlaylists(serverAddr, token)
		return playlistsFetchedMsg{playlists: playlists, err: err}
	}
}

// initPlaylistBrowse creates a new playlist browser
func (m *model) initPlaylistBrowse() {
	m.panelMode = "plex-playlists"
	m.status = "Loading playlists..."

	// Create a new default delegate with custom styling
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false

	items := []list.Item{playlistItem{title: "Loading playlists..."}}

	// Create the list with empty items for now
	m.playlistList = list.New(items, delegate, 0, 0)
	m.playlistList.Title = "Plex Playlists"
	m.playlistList.SetShowFilter(true)
	m.playlistList.SetFilteringEnabled(true)
	m.playlistList.Styles.Title = titleStyle
	m.playlistList.Styles.PaginationStyle = paginationStyle
	m.playlistList.Styles.HelpStyle = helpStyle

	m.playlistList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("f"),
				key.WithHelp("f", "favs"),
			),
		}
	}
	m.playlistList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("f"),
				key.WithHelp("f", "Add/Remove from Favorites"),
			),
			key.NewBinding(
				key.WithKeys("P"),
				key.WithHelp("P", "Play Playlist"),
			),
			key.NewBinding(
				key.WithKeys("R"),
				key.WithHelp("R", "Refresh Playlists"),
			),
		}
	}
	if m.width > 0 && m.height > 0 {
		m.playlistList.SetSize(m.width/2-4, m.height-4)
	}
}

func (m *model) playPlaylistCmd(ratingKey string) tea.Cmd {
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

	serverIP := m.selected
	serverID := m.config.ServerID
	shuffle := m.shuffle
	requestID := m.nextPlaybackRequestID()
	deps := m.deps

	return func() tea.Msg {
		err := PlayPlaylist(serverIP, serverID, ratingKey, shuffle, deps)
		if err != nil {
			return playbackTriggeredMsg{success: false, selected: serverIP, requestID: requestID, err: err}
		}
		return playbackTriggeredMsg{success: true, selected: serverIP, requestID: requestID}
	}
}

func (m *model) handlePlaylistBrowseUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.debug("handlePlaylistBrowseUpdate received message: %T", msg)

	// If we're in filtering mode, let the list handle the input
	if m.playlistList.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.playlistList, cmd = m.playlistList.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		switch key {
		case "esc", "q":
			// Return to playback panel
			m.panelMode = "playback"
			m.status = ""
			return m, nil

		case "enter":
			// View selected playlist's tracks
			if selected, ok := m.playlistList.SelectedItem().(playlistItem); ok {
				m.debug("Viewing playlist tracks: %s (ratingKey: %s)", selected.title, selected.ratingKey)
				m.lastCommand = fmt.Sprintf("Viewing %s", selected.title)
				m.trackReturnMode = "plex-playlists"
				m.initPlaylistTrackBrowse(selected.title, selected.ratingKey)
				return m, m.fetchPlaylistTracksCmd(selected.ratingKey)
			}
			return m, nil

		case "P":
			if selected, ok := m.playlistList.SelectedItem().(playlistItem); ok {
				m.debug("Playing playlist: %s (ratingKey: %s)", selected.title, selected.ratingKey)
				m.lastCommand = fmt.Sprintf("Playing %s", selected.title)
				return m, m.playPlaylistCmd(selected.ratingKey)
			}
			return m, nil

		case "f":
			// add or remove selected artist from favorites (playback list)
			if selected, ok := m.playlistList.SelectedItem().(playlistItem); ok {
				if selected.ratingKey == "" {
					m.debug("Ignoring playlist favorite toggle for item without rating key")
					return m, nil
				}
				m.debug("Toggling favorite for playlist: %s (ratingKey: %s)", selected.title, selected.ratingKey)
				m.lastCommand = fmt.Sprintf("Toggling favorite for %s", selected.title)
				_, cmd := m.addRemoveFavorite(selected.title, selected.ratingKey, "playlist")
				selected.ToggleFavorite()
				// Update the item in the list
				m.playlistList.SetItem(m.playlistList.Index(), selected)
				return m, cmd
			}

		case "R":
			// Refresh album list
			m.status = "Refreshing playlists..."
			return m, m.fetchPlaylistsCmd()

		default:

			// Otherwise try the common controls
			if cmd, handled := m.handleControl(key); handled {
				return m, cmd
			}
		}

	case playlistsFetchedMsg:
		m.debug("playlistsFetchedMsg received with %d playlists, error: %v", len(msg.playlists), msg.err)
		if msg.err != nil {
			errMsg := fmt.Sprintf("Error fetching playlists: %v", msg.err)
			m.status = errMsg
			m.debug("%s", errMsg)
			return m, nil
		}

		favSet := make(map[string]struct{})
		for _, pItem := range m.playbackList.Items() {
			pItem := pItem.(item)
			favSet[pItem.GetMetadataKey()] = struct{}{}
		}

		// Convert playlists to list items
		var items []list.Item
		for i, playlist := range msg.playlists {
			if i < 5 { // Only log first 5 playlists to avoid log spam
				m.debug("Adding playlist %d: %s (ratingKey: %s)", i+1, playlist.Title, playlist.RatingKey)
			}

			fav := false
			if _, exists := favSet[playlist.RatingKey]; exists {
				fav = true
			}
			title := playlist.Title
			if fav {
				title = fmt.Sprintf("%s ★", playlist.Title)
			}

			items = append(items, playlistItem{
				title:     title,
				ratingKey: playlist.RatingKey,
			})
		}

		m.debug("Creating new list with %d items", len(items))
		// Create a new list with the fetched items
		// Preserve the current filter state
		filterState := m.playlistList.FilterState()
		filterValue := m.playlistList.FilterValue()

		// Create a new default delegate with custom styling
		delegate := list.NewDefaultDelegate()
		delegate.ShowDescription = false // Don't show description

		// Create new list with existing items
		m.playlistList.SetItems(items)
		m.playlistList.ResetSelected()

		// Restore filter state if there was one
		if filterState == list.Filtering {
			m.playlistList.ResetFilter()
			m.playlistList.FilterInput.SetValue(filterValue)
		}
		m.status = fmt.Sprintf("Loaded %d playlists", len(msg.playlists))
		m.debug("Updated model with new playlist list. List has %d items", m.playlistList.VisibleItems())

		// Force a redraw
		return m, tea.Batch(tea.ClearScreen, func() tea.Msg { return nil })

	}

	// Update the artist list and get the command
	var listCmd tea.Cmd
	m.playlistList, listCmd = m.playlistList.Update(msg)
	// Return the current model (as a pointer) and the command
	return m, listCmd
}

// View renders the playlist browser
func (m *model) ViewPlaylist() string {
	return m.playlistList.View() + "\n" + m.status
}
