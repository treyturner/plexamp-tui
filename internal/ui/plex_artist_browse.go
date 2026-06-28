package ui

import (
	"fmt"
	"strings"

	"plexamp-tui/internal/plex"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// =====================
// Artist Browse Messages
// =====================

type artistsFetchedMsg struct {
	artists []plex.PlexArtist
	err     error
}

// =====================
// Artist Browse Functions
// =====================

// fetchArtistsCmd fetches artists from the Plex server
func (m *model) fetchArtistsCmd() tea.Cmd {
	m.debug("Fetching artists...")
	resizeBrowseListForFetch(&m.artistList, m.width, m.height)
	if m.config == nil {
		return func() tea.Msg {
			return artistsFetchedMsg{err: fmt.Errorf("no config available")}
		}
	}

	token := m.deps.plexClient.GetPlexToken()
	if token == "" {
		return func() tea.Msg {
			return artistsFetchedMsg{err: fmt.Errorf("no Plex token found - run with --auth flag")}
		}
	}

	serverAddr := m.config.PlexServerAddr
	libraryID := m.config.PlexLibraryID

	return func() tea.Msg {
		artists, err := m.deps.plexClient.FetchArtists(serverAddr, libraryID, token)
		return artistsFetchedMsg{artists: artists, err: err}
	}
}

// playArtistCmd starts playback for an artist (using artist's tracks)
func (m *model) playArtistCmd(ratingKey string) tea.Cmd {
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
	if m.deps.plexClient == nil {
		return func() tea.Msg {
			return playbackTriggeredMsg{success: false, err: fmt.Errorf("no Plex client available")}
		}
	}

	serverIP := m.selected
	serverID := m.config.ServerID
	shuffle := m.shuffle
	requestID := m.nextPlaybackRequestID()
	plexClient := m.deps.plexClient

	return func() tea.Msg {
		err := plexClient.PlayMetadata(serverIP, serverID, ratingKey, shuffle)
		if err != nil {
			return playbackTriggeredMsg{success: false, selected: serverIP, requestID: requestID, err: err}
		}
		return playbackTriggeredMsg{success: true, selected: serverIP, requestID: requestID}
	}
}

// playArtistRadioCmd starts playback for an artist's radio station
func (m *model) playArtistRadioCmd(ratingKey string) tea.Cmd {
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
	if m.deps.plexClient == nil {
		return func() tea.Msg {
			return playbackTriggeredMsg{success: false, err: fmt.Errorf("no Plex client available")}
		}
	}

	serverIP := m.selected
	serverID := m.config.ServerID
	shuffle := m.shuffle
	requestID := m.nextPlaybackRequestID()
	plexClient := m.deps.plexClient

	return func() tea.Msg {
		err := plexClient.PlayArtistRadio(serverIP, serverID, ratingKey, shuffle)
		if err != nil {
			return playbackTriggeredMsg{success: false, selected: serverIP, requestID: requestID, err: err}
		}
		return playbackTriggeredMsg{success: true, selected: serverIP, requestID: requestID}
	}
}

// initArtistBrowse initializes the artist browse panel
func (m *model) initArtistBrowse() {
	m.debug("Initializing artist browse")
	m.panelMode = panelModePlexArtists
	m.status = "Loading artists..."
	// Log the current model state
	m.debug("initArtistBrowse - panelMode: %s, status: %s", m.panelMode, m.status)

	items := []list.Item{artistItem{title: "Loading artists..."}}
	m.artistList = newBrowseList("Plex Artists", items, m.width, m.height)
	m.artistList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("f"),
				key.WithHelp("f", "favs"),
			),
		}
	}
	m.artistList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("f"),
				key.WithHelp("f", "Add/Remove from Favorites"),
			),
			key.NewBinding(
				key.WithKeys("P"),
				key.WithHelp("P", "Play Artist"),
			),
			key.NewBinding(
				key.WithKeys("r"),
				key.WithHelp("r", "Play Radio"),
			),
			key.NewBinding(
				key.WithKeys("R"),
				key.WithHelp("R", "Refresh Artists"),
			),
		}
	}

	m.debug("Initialized artist list with size: %dx%d", m.width/2-4, m.height-4)
}

// handleArtistBrowseUpdate handles updates when in artist browse mode
// It updates the model in place and returns the updated model and a command
func (m *model) handleArtistBrowseUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.debug("handleArtistBrowseUpdate received message: %T", msg)

	// If we're in filtering mode, let the list handle the input
	if m.artistList.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.artistList, cmd = m.artistList.Update(msg)
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

		case "enter":
			// View selected artist's albums
			if selected, ok := m.artistList.SelectedItem().(artistItem); ok {
				m.debug("Viewing artist albums: %s (ratingKey: %s)", selected.title, selected.ratingKey)
				m.lastCommand = fmt.Sprintf("Viewing %s", selected.title)
				m.initArtistAlbumBrowse(selected)
				return m, m.fetchArtistAlbumsCmd(selected.ratingKey)
			}
			return m, nil

		case "P":
			if selected, ok := m.artistList.SelectedItem().(artistItem); ok {
				m.debug("Playing artist: %s (ratingKey: %s)", selected.title, selected.ratingKey)
				m.lastCommand = fmt.Sprintf("Playing %s", selected.title)
				m.status = fmt.Sprintf("Starting playback for %s...", selected.title)
				return m, m.playArtistCmd(selected.ratingKey)
			}
			return m, nil

		case "f":
			// add or remove selected artist from favorites (playback list)
			if selected, ok := m.artistList.SelectedItem().(artistItem); ok {
				if selected.ratingKey == "" {
					m.debug("Ignoring artist favorite toggle for item without rating key")
					return m, nil
				}
				m.debug("Toggling favorite for artist: %s (ratingKey: %s)", selected.title, selected.ratingKey)
				m.lastCommand = fmt.Sprintf("Toggling favorite for %s", selected.title)
				_, cmd := m.addRemoveFavorite(selected.title, selected.ratingKey, favoriteTypeArtist)
				selected.ToggleFavorite()
				// Update the item in the list
				m.artistList.SetItem(m.artistList.Index(), selected)
				return m, cmd
			}

		case "r": // Shift+R for artist radio
			// Play selected artist's radio station
			if selected, ok := m.artistList.SelectedItem().(artistItem); ok {
				m.debug("Playing artist radio: %s (ratingKey: %s)", selected.title, selected.ratingKey)
				m.lastCommand = fmt.Sprintf("Playing %s Radio", selected.title)
				return m, m.playArtistRadioCmd(selected.ratingKey)
			}
			return m, nil

		case "R":
			// Refresh artist list
			m.status = "Refreshing artists..."
			return m, m.fetchArtistsCmd()

		default:

			// Otherwise try the common controls
			if cmd, handled := m.handleControl(key); handled {
				return m, cmd
			}
		}

	case artistsFetchedMsg:
		m.debug("artistsFetchedMsg received with %d artists, error: %v", len(msg.artists), msg.err)
		if msg.err != nil {
			errMsg := fmt.Sprintf("Error fetching artists: %v", msg.err)
			m.status = errMsg
			m.debug("%s", errMsg)
			return m, nil
		}

		favSet := m.favoritesController().metadataKeySet()

		// Convert artists to list items
		var items []list.Item
		for i, artist := range msg.artists {
			if i < 5 { // Only log first 5 artists to avoid log spam
				m.debug("Adding artist %d: %s (ratingKey: %s)", i+1, artist.Title, artist.RatingKey)
			}

			fav := false
			if _, exists := favSet[artist.RatingKey]; exists {
				fav = true
			}
			title := artist.Title
			if fav {
				title = fmt.Sprintf("%s ★", artist.Title)
			}
			items = append(items, artistItem{
				title:     title,
				ratingKey: artist.RatingKey,
			})
		}

		m.debug("Creating new list with %d items", len(items))
		replaceBrowseListItems(&m.artistList, items)
		m.status = fmt.Sprintf("Loaded %d artists", len(msg.artists))
		m.debug("Updated model with new artist list. List has %d items", m.artistList.VisibleItems())

		// Force a redraw
		return m, tea.Batch(tea.ClearScreen, func() tea.Msg { return nil })

	}

	// Update the artist list and get the command
	var listCmd tea.Cmd
	m.artistList, listCmd = m.artistList.Update(msg)
	// Return the current model (as a pointer) and the command
	return m, listCmd
}

// =====================
// Artist Item Type
// =====================

type artistItem struct {
	title     string
	ratingKey string
}

func (i artistItem) Title() string       { return i.title }
func (i artistItem) Description() string { return "" } // No description needed
// FilterValue implements list.Item
func (i artistItem) FilterValue() string {
	// Return the title in lowercase for case-insensitive matching
	return i.title
}

func (a *artistItem) ToggleFavorite() {
	// If title already has a star, remove it
	if strings.HasSuffix(a.title, " ★") {
		a.title = strings.TrimSuffix(a.title, " ★")
	} else {
		a.title = fmt.Sprintf("%s ★", a.title)
	}
}

// Custom styles for the list
var (
	titleStyle      = lipgloss.NewStyle().MarginLeft(2)
	helpStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Margin(1, 0, 0, 2)
	paginationStyle = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
)
