package ui

import (
	"fmt"
	"strings"

	"plexamp-tui/internal/plex"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// albumItem represents an album in the list
type albumItem struct {
	title     string
	artist    string
	year      string
	ratingKey string
}

// Title returns the album title
func (i albumItem) Title() string {
	if strings.HasSuffix(i.title, " ★") {
		return fmt.Sprintf("%s - %s (%s) ★", strings.TrimSuffix(i.title, " ★"), i.artist, i.year)
	}
	return fmt.Sprintf("%s - %s (%s)", i.title, i.artist, i.year)
}

// Description returns the album description (empty for now)
func (i albumItem) Description() string { return "" }

// FilterValue implements list.Item
func (i albumItem) FilterValue() string {
	return i.title + " " + i.artist
}

func (a *albumItem) ToggleFavorite() {
	// If title already has a star, remove it
	if strings.HasSuffix(a.title, " ★") {
		a.title = strings.TrimSuffix(a.title, " ★")
	} else {
		a.title = fmt.Sprintf("%s ★", a.title)
	}
}

// fetchAlbumsCmd fetches albums from the Plex server
func (m *model) fetchAlbumsCmd() tea.Cmd {
	m.debug("Fetching albums...")
	resizeBrowseListForFetch(&m.albumList, m.width, m.height)
	if m.config == nil {
		return func() tea.Msg {
			return albumsFetchedMsg{err: fmt.Errorf("no config available")}
		}
	}

	token := m.deps.plexClient.GetPlexToken()
	if token == "" {
		return func() tea.Msg {
			return albumsFetchedMsg{err: fmt.Errorf("no Plex token found - run with --auth flag")}
		}
	}

	serverAddr := m.config.PlexServerAddr
	libraryID := m.config.PlexLibraryID

	return func() tea.Msg {
		albums, err := m.deps.plexClient.FetchAlbums(serverAddr, libraryID, token)
		return albumsFetchedMsg{albums: albums, err: err}
	}
}

// initAlbumBrowse creates a new album browser
func (m *model) initAlbumBrowse() {
	m.panelMode = panelModePlexAlbums
	m.status = "Loading albums..."

	items := []list.Item{albumItem{title: "Loading albums..."}}
	m.albumList = newBrowseList("Plex Albums", items, m.width, m.height)
	m.albumList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("f"),
				key.WithHelp("f", "favs"),
			),
		}
	}
	m.albumList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("f"),
				key.WithHelp("f", "Add/Remove from Favorites"),
			),
			key.NewBinding(
				key.WithKeys("P"),
				key.WithHelp("P", "Play Album"),
			),
			key.NewBinding(
				key.WithKeys("R"),
				key.WithHelp("R", "Refresh Albums"),
			),
		}
	}
}

func (m *model) playAlbumCmd(ratingKey string) tea.Cmd {
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
		err := PlayMetadata(serverIP, serverID, ratingKey, shuffle, deps)
		if err != nil {
			return playbackTriggeredMsg{success: false, selected: serverIP, requestID: requestID, err: err}
		}
		return playbackTriggeredMsg{success: true, selected: serverIP, requestID: requestID}
	}
}

func (m *model) handleAlbumBrowseUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.debug("handleAlbumBrowseUpdate received message: %T", msg)

	// If we're in filtering mode, let the list handle the input
	if m.albumList.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.albumList, cmd = m.albumList.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		switch key {
		case "esc", "q":
			// Return to playback panel
			m.panelMode = panelModePlayback
			m.status = ""
			return m, nil

		case "f":
			// add or remove selected artist from favorites (playback list)
			if selected, ok := m.albumList.SelectedItem().(albumItem); ok {
				if selected.ratingKey == "" {
					m.debug("Ignoring album favorite toggle for item without rating key")
					return m, nil
				}
				m.debug("Toggling favorite for album: %s (ratingKey: %s)", selected.title, selected.ratingKey)
				m.lastCommand = fmt.Sprintf("Toggling favorite for %s", selected.title)

				_, cmd := m.addRemoveFavorite(selected.title, selected.ratingKey, favoriteTypeAlbum)
				selected.ToggleFavorite()

				// Update the item in the list
				m.albumList.SetItem(m.albumList.Index(), selected)

				return m, cmd
			}

		case "enter":
			// View selected album's tracks
			if selected, ok := m.albumList.SelectedItem().(albumItem); ok {
				m.debug("Viewing album tracks: %s (ratingKey: %s)", selected.title, selected.ratingKey)
				m.lastCommand = fmt.Sprintf("Viewing %s", selected.title)
				m.trackReturnMode = panelModePlexAlbums
				m.initAlbumTrackBrowse(selected.title, selected.ratingKey)
				return m, m.fetchAlbumTracksCmd(selected.ratingKey)
			}
			return m, nil

		case "P":
			if selected, ok := m.albumList.SelectedItem().(albumItem); ok {
				m.debug("Playing album: %s (ratingKey: %s)", selected.title, selected.ratingKey)
				m.lastCommand = fmt.Sprintf("Playing %s", selected.title)
				return m, m.playAlbumCmd(selected.ratingKey)
			}
			return m, nil

		case "R":
			// Refresh album list
			m.status = "Refreshing albums..."
			m.lastCommand = "Refreshing album list"
			return m, m.fetchAlbumsCmd()

		default:

			// Otherwise try the common controls
			if cmd, handled := m.handleControl(key); handled {
				return m, cmd
			}
		}

	case albumsFetchedMsg:
		m.debug("albumsFetchedMsg received with %d albums, error: %v", len(msg.albums), msg.err)
		if msg.err != nil {
			errMsg := fmt.Sprintf("Error fetching albums: %v", msg.err)
			m.status = errMsg
			m.debug("%s", errMsg)
			return m, nil
		}

		favSet := m.favoritesController().metadataKeySet()
		// Convert albums to list items
		var items []list.Item
		for i, album := range msg.albums {
			if i < 5 { // Only log first 5 albums to avoid log spam
				m.debug("Adding album %d: %s (ratingKey: %s)", i+1, album.Title, album.RatingKey)
			}

			fav := false
			if _, exists := favSet[album.RatingKey]; exists {
				fav = true
			}
			title := album.Title
			if fav {
				title = fmt.Sprintf("%s ★", album.Title)
			}

			items = append(items, albumItem{
				title:     title,
				artist:    album.ParentTitle,
				year:      album.Year,
				ratingKey: album.RatingKey,
			})
		}

		m.debug("Creating new list with %d items", len(items))
		replaceBrowseListItems(&m.albumList, items)
		m.status = fmt.Sprintf("Loaded %d albums", len(msg.albums))
		m.debug("Updated model with new album list. List has %d items", m.albumList.VisibleItems())

		resizeBrowseListForFetch(&m.albumList, m.width, m.height)

		// Force a redraw
		return m, tea.Batch(tea.ClearScreen, func() tea.Msg { return nil })

	}

	// Update the artist list and get the command
	var listCmd tea.Cmd
	m.albumList, listCmd = m.albumList.Update(msg)
	// Return the current model (as a pointer) and the command
	return m, listCmd
}

// View renders the album browser
func (m *model) ViewAlbum() string {
	return m.albumList.View() + "\n" + m.status
}

// albumsFetchedMsg is a message containing fetched albums
type albumsFetchedMsg struct {
	albums []plex.PlexAlbum
	err    error
}
