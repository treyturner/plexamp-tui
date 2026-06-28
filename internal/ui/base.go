// Package ui contains the main TUI model and Bubble Tea implementation for Plexamp control.
package ui

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"plexamp-tui/internal/config"
	"plexamp-tui/internal/logger"
	"plexamp-tui/internal/plex"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// =====================
// TUI Types
// =====================

type item struct {
	Name        string
	Type        string
	MetadataKey string
}

func (i item) Title() string          { return string(i.Name) }
func (i item) Description() string    { return string(i.Type) }
func (i item) FilterValue() string    { return string(i.Name) }
func (i item) GetMetadataKey() string { return i.MetadataKey }

type model struct {
	deps uiDeps

	playbackList            list.Model
	artistList              list.Model // Plex artist browse list
	artistAlbumList         list.Model // Plex artist album browse list
	albumList               list.Model // Plex album browse list
	trackList               list.Model // Plex track browse list
	playlistList            list.Model // Plex playlist browse list
	serverList              list.Model // Plex server browse list
	playerList              list.Model // Plex player browse list
	selected                string
	status                  string
	width                   int
	height                  int
	playback                playbackState
	lastCommand             string
	usingDefaultCfg         bool
	shuffle                 bool // Tracks shuffle state
	shuffleCommandID        int
	pendingShuffleCommandID int
	pendingShuffleBase      bool
	acknowledgedShuffle     bool
	acknowledgedShuffleID   int
	plexAuthenticated       bool // Plex authentication status
	playbackRequestID       int
	ackPlaybackRequestID    int
	trackPlaybackReqID      int
	ackTrackPlaybackReqID   int
	ackTrackPlaybackKey     string
	currentArtistKey        string
	currentArtistName       string
	artistAlbumReturnMode   panelMode
	currentAlbumKey         string
	currentAlbumName        string
	currentPlaylistKey      string
	currentPlaylistName     string
	trackReturnMode         panelMode

	panelMode      panelMode
	playbackConfig *config.Favorites
	config         *config.Config // Store config for server ID access

	// Edit mode fields
	editMode       editMode
	editIndex      int // Index of item being edited
	editInputs     []textinput.Model
	typeSelect     list.Model // Dropdown for type selection
	editFocusIndex int
}

type MediaContainer struct {
	Timelines []Timeline `xml:"Timeline"`
}

type Timeline struct {
	Type     string `xml:"type,attr"`
	State    string `xml:"state,attr"`
	Time     int    `xml:"time,attr"`
	Duration int    `xml:"duration,attr"`
	Volume   int    `xml:"volume,attr"`
	Track    Track  `xml:"Track"`
}

type Track struct {
	RatingKey        string `xml:"ratingKey,attr"`
	Title            string `xml:"title,attr"`
	ParentTitle      string `xml:"parentTitle,attr"`
	GrandparentTitle string `xml:"grandparentTitle,attr"`
}

type (
	trackMsg string
	errMsg   struct{ err error }
	pollMsg  struct{}
)

type trackMsgWithState struct {
	Selected  string
	TrackText string
	TrackKey  string
	IsPlaying bool
	Duration  int
	Position  int
	Volume    int
	RequestID int
}

type playbackTriggeredMsg struct {
	success   bool
	selected  string
	requestID int
	err       error
}

type playbackControlAction int

const (
	playbackControlToggle playbackControlAction = iota
	playbackControlNext
	playbackControlPrevious
	playbackControlVolume
	playbackControlShuffle
)

var errNoPlayerSelected = errors.New("no Plexamp instance selected")

type playbackControlMsg struct {
	action           playbackControlAction
	path             string
	selected         string
	isPlaying        bool
	toggleCommandID  int
	volume           int
	volumeCommandID  int
	shuffle          bool
	shuffleCommandID int
	poll             bool
	err              error
}

type UiManager struct {
	Model model
}

type uiDeps struct {
	log         *logger.Logger
	cfgManager  *config.Manager
	plexClient  *plex.PlexClient
	favsManager *config.FavoritesManager
}

func (d uiDeps) debug(format string, v ...interface{}) {
	if d.log == nil {
		return
	}
	d.log.Debug(format, v...)
}

func (m model) debug(format string, v ...interface{}) {
	m.deps.debug(format, v...)
}

func NewUiManager(logger *logger.Logger, config *config.Config, manager *config.Manager,
	favorites *config.Favorites, client *plex.PlexClient, favoritesMgr *config.FavoritesManager,
) *UiManager {
	// Create playback list
	var playbackItems []list.Item
	if favorites != nil {
		for _, pb := range favorites.Items {
			playbackItems = append(playbackItems, item{Name: pb.Name, Type: pb.Type, MetadataKey: pb.MetadataKey})
		}
	}
	playbackList := list.New(playbackItems, list.NewDefaultDelegate(), 0, 0)
	playbackList.Title = "Favorites"
	// Add keys to the short help (shown at the bottom of the list)
	playbackList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "open"),
			),
			key.NewBinding(
				key.WithKeys("P"),
				key.WithHelp("P", "play"),
			),
			key.NewBinding(
				key.WithKeys("a"),
				key.WithHelp("a", "add"),
			),
			key.NewBinding(
				key.WithKeys("e"),
				key.WithHelp("e", "edit"),
			),
			key.NewBinding(
				key.WithKeys("d"),
				key.WithHelp("d", "delete"),
			),
		}
	}

	// Add keys to the full help (shown when pressing '?')
	playbackList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "Open selected item"),
			),
			key.NewBinding(
				key.WithKeys("P"),
				key.WithHelp("P", "Play selected item"),
			),
			key.NewBinding(
				key.WithKeys("r"),
				key.WithHelp("r", "Play artist radio"),
			),
			key.NewBinding(
				key.WithKeys("a"),
				key.WithHelp("a", "Add new item"),
			),
			key.NewBinding(
				key.WithKeys("e"),
				key.WithHelp("e", "Edit selected item"),
			),
			key.NewBinding(
				key.WithKeys("d"),
				key.WithHelp("d", "Delete selected item"),
			),
		}
	}

	m := model{
		deps: uiDeps{
			log:         logger,
			cfgManager:  manager,
			plexClient:  client,
			favsManager: favoritesMgr,
		},
		playbackList:      playbackList,
		artistList:        list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
		artistAlbumList:   list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
		albumList:         list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
		trackList:         list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
		playlistList:      list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
		serverList:        list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
		playerList:        list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
		selected:          config.SelectedPlayer,
		usingDefaultCfg:   manager.UsingDefault,
		playbackConfig:    favorites,
		config:            config,
		panelMode:         panelModePlayback,
		shuffle:           true, // Default shuffle to ON
		plexAuthenticated: client.VerifyPlexAuthentication(),
	}

	return &UiManager{
		Model: m,
	}
}

// =====================
// Bubble Tea Methods
// =====================

func (m model) Init() tea.Cmd {
	return tea.Batch(m.pollTimeline(), tick())
}

func tick() tea.Cmd {
	return tea.Tick(time.Second*2, func(time.Time) tea.Msg {
		return pollMsg{}
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case playerSelectMsg:
		if msg.err != nil {
			m.status = "Error selecting player: " + msg.err.Error()
			return m, nil
		}
		if msg.success {
			if msg.player.address != m.selected {
				m.clearPendingPlaybackControls()
			}
			m.config.SelectedPlayer = msg.player.address
			m.config.SelectedPlayerName = msg.player.title
			m.selected = msg.player.address
			if err := m.deps.cfgManager.Save(m.config); err != nil {
				m.status = "Error saving player selection: " + err.Error()
				m.lastCommand = "Player Selection Save Failed"
				return m, nil
			}
			m.lastCommand = "Player Selected"
			m.status = ""
			m.panelMode = panelModePlayback // Return to playback view after selection
		}
		return m, nil

	case serverSelectMsg:
		if msg.err != nil {
			m.status = "Error selecting server: " + msg.err.Error()
			return m, nil
		}
		if msg.success {
			m.config.ServerID = msg.server.clientIdentifier
			serverAddr := msg.server.address + ":" + msg.server.port
			if msg.server.scheme != "" {
				serverAddr = msg.server.scheme + "://" + serverAddr
			}
			m.config.PlexServerAddr = serverAddr
			m.config.PlexServerName = msg.server.title
			m.config.PlexLibraries = msg.libraries

			found := false
			if len(msg.libraries) == 0 {
				m.debug("No libraries found on this server")
				m.panelMode = panelModePlayback
				m.lastCommand = "Server Selected Failed, No Libraries"
				m.status = "No libraries found on this server"
				return m, nil
			}

			// check if new library list has our configured library
			for _, lib := range msg.libraries {
				if lib.Title == m.config.PlexLibraryName {
					found = true
					break
				}
			}

			if !found {
				m.debug("Current Library not found on this server, using first library")
				m.config.PlexLibraryName = msg.libraries[0].Title
				m.config.PlexLibraryID = msg.libraries[0].Key
			}

			m.debug("Saving server config: %v", m.config)
			if err := m.deps.cfgManager.Save(m.config); err != nil {
				m.status = "Error saving server selection: " + err.Error()
				m.lastCommand = "Server Selection Save Failed"
				return m, nil
			}
			m.lastCommand = "Server Selected"
			m.status = ""
			m.panelMode = panelModePlayback // Return to playback view after selection
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

		// Reserve a few lines for the footer (and maybe the title)
		footerHeight := 3 // adjust if your footer grows taller
		titleHeight := 3
		availableHeight := msg.Height - footerHeight - titleHeight - 2

		m.playbackList.SetSize(msg.Width/2-4, availableHeight)
		m.artistList.SetSize(msg.Width/2-4, availableHeight)
		m.artistAlbumList.SetSize(msg.Width/2-4, availableHeight)
		m.albumList.SetSize(msg.Width/2-4, availableHeight)
		m.trackList.SetSize(msg.Width/2-4, availableHeight)
		m.playlistList.SetSize(msg.Width/2-4, availableHeight)
		m.serverList.SetSize(msg.Width/2-4, availableHeight)
		m.playerList.SetSize(msg.Width/2-4, availableHeight)

		return m, nil

	case tea.KeyMsg:
		// Handle edit mode separately
		if m.panelMode == panelModeEdit {
			return m.handleEditUpdate(msg)
		}

		// Handle artist browse mode
		if m.panelMode == panelModePlexArtists {
			modelPtr := &m
			_, cmd := modelPtr.handleArtistBrowseUpdate(msg)
			return m, cmd
		}

		// Handle album browse mode
		if m.panelMode == panelModePlexAlbums {
			modelPtr := &m
			_, cmd := modelPtr.handleAlbumBrowseUpdate(msg)
			return m, cmd
		}

		// Handle artist album browse mode
		if m.panelMode == panelModePlexArtistAlbums {
			modelPtr := &m
			_, cmd := modelPtr.handleArtistAlbumBrowseUpdate(msg)
			return m, cmd
		}

		// Handle album/playlist track browse mode
		if m.panelMode == panelModePlexAlbumTracks || m.panelMode == panelModePlexPlaylistTracks {
			modelPtr := &m
			_, cmd := modelPtr.handleTrackBrowseUpdate(msg)
			return m, cmd
		}

		// Handle playlist browse mode
		if m.panelMode == panelModePlexPlaylists {
			modelPtr := &m
			_, cmd := modelPtr.handlePlaylistBrowseUpdate(msg)
			return m, cmd
		}

		// Handle server browse mode
		if m.panelMode == panelModePlexServers {
			modelPtr := &m
			_, cmd := modelPtr.handleServerBrowseUpdate(msg)
			return m, cmd
		}

		// Handle player browse mode
		if m.panelMode == panelModePlexPlayers {
			modelPtr := &m
			_, cmd := modelPtr.handlePlayerBrowseUpdate(msg)
			return m, cmd
		}

		// Handle playback selection (when in playback/favorites mode)
		if m.panelMode == panelModePlayback {
			// Check if we're in filtering mode for the playback list
			if m.playbackList.FilterState() == list.Filtering {
				var cmd tea.Cmd
				m.playbackList, cmd = m.playbackList.Update(msg)
				return m, cmd
			}

			switch msg.String() {
			case "a":
				// Add new playback item
				m.initEditMode(editModePlayback, -1)
				return m, nil

			case "e":
				// Edit selected playback item
				index := m.playbackList.Index()
				m.initEditMode(editModePlayback, index)
				return m, nil

			case "d":
				// Delete selected playback item
				index := m.playbackList.Index()
				if err := m.deletePlaybackItem(index); err != nil {
					m.status = "Error deleting favorite: " + err.Error()
					m.lastCommand = "Delete Failed"
				}
				return m, nil

			case "r":
				// play station/radio if selection is an artist
				if selected, ok := m.playbackList.SelectedItem().(item); ok {
					if pb, found := m.findFavoriteItem(selected); found && favoriteType(pb.Type) == favoriteTypeArtist {
						return m, m.triggerFavoriteRadioPlayback(pb)
					}
				}

			case "enter":
				// Drill down into the selected favorite item.
				if selected, ok := m.playbackList.SelectedItem().(item); ok {
					if pb, found := m.findFavoriteItem(selected); found {
						return m, m.openFavoriteItem(pb)
					}
				}
				return m, nil

			case "P":
				// Direct playback for selected favorite item.
				if selected, ok := m.playbackList.SelectedItem().(item); ok {
					if pb, found := m.findFavoriteItem(selected); found {
						return m, m.triggerFavoritePlayback(pb)
					}
				}
				return m, nil

			}
		}

		// Main app key handlers (only processed when popup is NOT open)
		key := msg.String()

		switch key {
		case "ctrl+c", "q":
			return m, tea.Quit

		default:
			// Try the common controls
			if cmd, handled := m.handleControl(key); handled {
				return m, cmd
			}
		}

	case pollMsg:
		return m, tea.Batch(m.pollTimeline(), tick())

	case trackMsgWithState:
		if msg.Selected != "" && msg.Selected != m.selected {
			return m, nil
		}
		m.playback.applyTimeline(msg, time.Now(), m.debug)
		return m, nil

	case trackMsg:
		m.playback.currentTrack = string(msg)
		return m, nil

	case errMsg:
		m.status = fmt.Sprintf("Error: %v", msg.err)
		return m, nil

	case playbackControlMsg:
		if msg.selected != m.selected {
			return m, nil
		}

		m.recordPlaybackControlAck(msg)
		if m.isStalePlaybackControl(msg) {
			return m, nil
		}

		if msg.err != nil {
			m.rollbackFailedPlaybackControl(msg)
			if errors.Is(msg.err, errNoPlayerSelected) {
				m.status = "No Plexamp instance selected"
			} else {
				m.status = fmt.Sprintf("Error: %v", msg.err)
			}
			return m, nil
		}

		m.applyPlaybackControl(msg)

		m.status = fmt.Sprintf("[%s] Sent %s", msg.selected, msg.path)
		if msg.poll {
			return m, m.pollTimeline()
		}
		return m, nil

	case playbackTriggeredMsg:
		if msg.selected != "" && msg.selected != m.selected {
			return m, nil
		}
		if msg.success && msg.requestID != 0 && msg.requestID > m.ackPlaybackRequestID {
			m.ackPlaybackRequestID = msg.requestID
		}
		if msg.requestID != 0 && msg.requestID != m.playbackRequestID {
			return m, nil
		}

		if msg.success {
			// Invalidate in-flight track playback responses when generic playback starts.
			m.nextTrackPlaybackRequestID()
			m.lastCommand = "Playback Started"
			m.status = "Playback triggered successfully"
			return m, m.beginPlaybackRefresh("")
		} else {
			if msg.requestID != 0 && m.ackPlaybackRequestID != 0 {
				m.lastCommand = "Playback Started"
				m.status = "Playback triggered successfully"
				return m, m.beginPlaybackRefresh("")
			}
			m.lastCommand = "Playback Failed"
			m.status = fmt.Sprintf("Playback error: %v", msg.err)
		}
		return m, nil

	case artistsFetchedMsg:
		// Forward the message to the artist browse handler
		if m.panelMode == panelModePlexArtists {
			modelPtr := &m
			_, cmd := modelPtr.handleArtistBrowseUpdate(msg)
			return m, cmd
		}
		return m, nil

	case albumsFetchedMsg:
		// Forward the message to the album browse handler
		if m.panelMode == panelModePlexAlbums {
			modelPtr := &m
			_, cmd := modelPtr.handleAlbumBrowseUpdate(msg)
			return m, cmd
		}
		return m, nil

	case artistAlbumsFetchedMsg:
		if m.panelMode == panelModePlexArtistAlbums {
			modelPtr := &m
			_, cmd := modelPtr.handleArtistAlbumBrowseUpdate(msg)
			return m, cmd
		}
		return m, nil

	case tracksFetchedMsg:
		if m.panelMode == panelModePlexAlbumTracks || m.panelMode == panelModePlexPlaylistTracks {
			modelPtr := &m
			_, cmd := modelPtr.handleTrackBrowseUpdate(msg)
			return m, cmd
		}
		return m, nil

	case trackPlaybackMsg:
		if msg.selected != "" && msg.selected != m.selected {
			return m, nil
		}
		if msg.success && msg.requestID > m.ackTrackPlaybackReqID {
			m.ackTrackPlaybackReqID = msg.requestID
			m.ackTrackPlaybackKey = msg.ratingKey
		}
		if msg.requestID != m.trackPlaybackReqID {
			m.debug(
				"Ignoring stale trackPlaybackMsg (requestID=%d, current=%d)",
				msg.requestID, m.trackPlaybackReqID,
			)
			return m, nil
		}

		if msg.success {
			m.lastCommand = "Track Playback Started"
			m.status = "Playback triggered successfully"
			return m, m.beginPlaybackRefreshForTrack("", msg.ratingKey)
		}

		m.lastCommand = "Playback Failed"
		m.status = fmt.Sprintf("Playback error: %v", msg.err)
		if m.ackTrackPlaybackReqID != 0 {
			m.lastCommand = "Track Playback Started"
			m.status = "Playback triggered successfully"
			return m, m.beginPlaybackRefreshForTrack("", m.ackTrackPlaybackKey)
		}
		m.playback.clearAfterFailure()
		return m, nil

	case playlistsFetchedMsg:
		// Forward the message to the playlist browse handler
		if m.panelMode == panelModePlexPlaylists {
			modelPtr := &m
			_, cmd := modelPtr.handlePlaylistBrowseUpdate(msg)
			return m, cmd
		}
		return m, nil

	case serversFetchedMsg:
		// Forward the message to the server browse handler
		if m.panelMode == panelModePlexServers {
			modelPtr := &m
			_, cmd := modelPtr.handleServerBrowseUpdate(msg)
			return m, cmd
		}
		return m, nil

	case playersFetchedMsg:
		// Forward the message to the player browse handler
		if m.panelMode == panelModePlexPlayers {
			modelPtr := &m
			_, cmd := modelPtr.handlePlayerBrowseUpdate(msg)
			return m, cmd
		}
		return m, nil
	}

	// Update the appropriate list based on panel mode
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
	return m, cmd
}

func (m model) View() string {
	border := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00ffff")).Render("🎧 Plexamp Control")

	// Show edit panel if in edit mode
	if m.panelMode == panelModeEdit {
		editContent := m.editPanelView()
		editPanel := border.Width(m.width - 4).Render(editContent)
		return lipgloss.JoinVertical(lipgloss.Left, title, editPanel)
	}

	// Build left panel content
	var leftPanelContent string
	switch m.panelMode {
	case panelModePlayback:
		leftPanelContent = m.playbackList.View()
	case panelModePlexArtists:
		leftPanelContent = m.artistList.View()
	case panelModePlexArtistAlbums:
		leftPanelContent = m.artistAlbumList.View()
	case panelModePlexAlbums:
		leftPanelContent = m.albumList.View()
	case panelModePlexAlbumTracks, panelModePlexPlaylistTracks:
		leftPanelContent = m.trackList.View()
	case panelModePlexPlaylists:
		leftPanelContent = m.playlistList.View()
	case panelModePlexServers:
		leftPanelContent = m.serverList.View()
	case panelModePlexPlayers:
		leftPanelContent = m.playerList.View()
	}

	// Left panel
	leftPanel := border.Width(m.width/2 - 2).Render(leftPanelContent)

	// Right side has two stacked panels
	playbackPanel := border.Width(m.width/2 - 2).Render(m.playbackStatusView())
	controlsPanel := border.Width(m.width/2 - 2).Render(m.appControlsView())
	rightSide := lipgloss.JoinVertical(lipgloss.Left, playbackPanel, controlsPanel)

	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightSide)

	// Combine all elements with the footer at the bottom
	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinVertical(lipgloss.Left, title, content),
		"\n"+m.footerView(),
	)
}

// helper
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// =====================
// Plexamp control logic
// =====================

func playbackControlCmd(selected, path string, msg playbackControlMsg) tea.Cmd {
	msg.selected = selected
	msg.path = path

	return func() tea.Msg {
		if selected == "" {
			msg.err = errNoPlayerSelected
			return msg
		}

		url := fmt.Sprintf("http://%s:32500/player/%s", selected, path)
		resp, err := http.Get(url)
		if err != nil {
			msg.err = err
			return msg
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			msg.err = fmt.Errorf("server returned status %d", resp.StatusCode)
		}
		return msg
	}
}

func (m *model) recordPlaybackControlAck(msg playbackControlMsg) {
	if msg.err != nil {
		return
	}

	switch msg.action {
	case playbackControlToggle:
		if msg.toggleCommandID > m.playback.acknowledgedToggleID {
			m.playback.acknowledgedPlaying = msg.isPlaying
			m.playback.acknowledgedToggleID = msg.toggleCommandID
		}
	case playbackControlVolume:
		if msg.volumeCommandID > m.playback.acknowledgedVolumeID {
			m.playback.acknowledgedVolume = msg.volume
			m.playback.acknowledgedVolumeID = msg.volumeCommandID
		}
	case playbackControlShuffle:
		if msg.shuffleCommandID > m.acknowledgedShuffleID {
			m.acknowledgedShuffle = msg.shuffle
			m.acknowledgedShuffleID = msg.shuffleCommandID
		}
	}
}

func (m model) isStalePlaybackControl(msg playbackControlMsg) bool {
	switch msg.action {
	case playbackControlToggle:
		return stalePlaybackControlID(
			msg.toggleCommandID,
			m.playback.pendingToggleCommandID,
			m.playback.acknowledgedToggleID,
			msg.err,
		)
	case playbackControlVolume:
		return stalePlaybackControlID(
			msg.volumeCommandID,
			m.playback.pendingVolumeCommandID,
			m.playback.acknowledgedVolumeID,
			msg.err,
		)
	case playbackControlShuffle:
		return stalePlaybackControlID(
			msg.shuffleCommandID,
			m.pendingShuffleCommandID,
			m.acknowledgedShuffleID,
			msg.err,
		)
	default:
		return false
	}
}

func stalePlaybackControlID(id, pendingID, acknowledgedID int, err error) bool {
	if id == 0 || id == pendingID {
		return false
	}
	if pendingID == 0 && err == nil && id == acknowledgedID {
		return false
	}
	return true
}

func (m *model) rollbackFailedPlaybackControl(msg playbackControlMsg) {
	switch msg.action {
	case playbackControlToggle:
		if msg.toggleCommandID == 0 {
			return
		}
		if m.playback.acknowledgedToggleID != 0 {
			m.playback.isPlaying = m.playback.acknowledgedPlaying
		} else {
			m.playback.isPlaying = m.playback.pendingToggleBasePlaying
		}
		m.playback.pendingToggleCommandID = 0
	case playbackControlVolume:
		if msg.volumeCommandID == 0 {
			return
		}
		if m.playback.acknowledgedVolumeID != 0 {
			m.playback.volume = m.playback.acknowledgedVolume
		} else {
			m.playback.volume = m.playback.pendingVolumeBase
		}
		m.playback.pendingVolumeCommandID = 0
	case playbackControlShuffle:
		if msg.shuffleCommandID == 0 {
			return
		}
		if m.acknowledgedShuffleID != 0 {
			m.shuffle = m.acknowledgedShuffle
		} else {
			m.shuffle = m.pendingShuffleBase
		}
		m.pendingShuffleCommandID = 0
	}
}

func (m *model) applyPlaybackControl(msg playbackControlMsg) {
	switch msg.action {
	case playbackControlToggle:
		if msg.toggleCommandID != 0 {
			m.playback.pendingToggleCommandID = 0
		}
		m.playback.isPlaying = msg.isPlaying
		if msg.isPlaying {
			m.lastCommand = "Play"
		} else {
			m.lastCommand = "Pause"
		}
	case playbackControlNext:
		m.lastCommand = "Next"
	case playbackControlPrevious:
		m.playback.restartPrevious(time.Now())
		m.lastCommand = "Previous"
	case playbackControlVolume:
		if msg.volumeCommandID != 0 {
			m.playback.pendingVolumeCommandID = 0
		}
		m.playback.volume = msg.volume
		m.lastCommand = fmt.Sprintf("Volume %d%%", msg.volume)
	case playbackControlShuffle:
		if msg.shuffleCommandID != 0 {
			m.pendingShuffleCommandID = 0
		}
		m.shuffle = msg.shuffle
		if msg.shuffle {
			m.lastCommand = "Shuffle ON"
		} else {
			m.lastCommand = "Shuffle OFF"
		}
	}
}

func (m *model) clearPendingPlaybackControls() {
	m.playback.pendingToggleCommandID = 0
	m.playback.pendingVolumeCommandID = 0
	m.playback.acknowledgedToggleID = 0
	m.playback.acknowledgedVolumeID = 0
	m.pendingShuffleCommandID = 0
	m.acknowledgedShuffle = false
	m.acknowledgedShuffleID = 0
	m.playbackRequestID++
	m.ackPlaybackRequestID = 0
	m.trackPlaybackReqID++
	m.ackTrackPlaybackReqID = 0
	m.ackTrackPlaybackKey = ""
	m.playback.timelineRequestID++
}

func (m *model) nextPlaybackRequestID() int {
	m.playbackRequestID++
	m.ackPlaybackRequestID = 0
	return m.playbackRequestID
}

func (m *model) nextTrackPlaybackRequestID() int {
	m.trackPlaybackReqID++
	m.ackTrackPlaybackReqID = 0
	m.ackTrackPlaybackKey = ""
	return m.trackPlaybackReqID
}

func (m *model) pollTimeline() tea.Cmd {
	if m.selected == "" {
		return nil
	}
	reqID := m.playback.timelineRequestID
	selected := m.selected

	return func() tea.Msg {
		url := fmt.Sprintf("http://%s:32500/player/timeline/poll?wait=1&includeMetadata=1&commandID=1&type=music", selected)
		resp, err := http.Get(url)
		if err != nil {
			return trackMsgWithState{
				Selected:  selected,
				RequestID: reqID,
				TrackText: "",
				TrackKey:  "",
				IsPlaying: false,
				Duration:  0,
				Position:  0,
				Volume:    0,
			}
		}
		defer func() { _ = resp.Body.Close() }()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return trackMsgWithState{
				Selected:  selected,
				RequestID: reqID,
				TrackText: "",
				TrackKey:  "",
				IsPlaying: false,
				Duration:  0,
				Position:  0,
				Volume:    0,
			}
		}

		var mc MediaContainer
		if err := xml.Unmarshal(data, &mc); err != nil {
			return trackMsgWithState{
				Selected:  selected,
				RequestID: reqID,
				TrackText: "",
				TrackKey:  "",
				IsPlaying: false,
				Duration:  0,
				Position:  0,
				Volume:    0,
			}
		}

		var chosen *Timeline
		for i := range mc.Timelines {
			t := &mc.Timelines[i]
			if t.Type == "music" {
				chosen = t
				break
			}
		}
		if chosen == nil && len(mc.Timelines) > 0 {
			chosen = &mc.Timelines[0]
		}

		track := ""
		trackKey := ""
		isPlaying := false
		duration := 0
		position := 0
		volume := 0
		if chosen != nil {
			trackKey = chosen.Track.RatingKey
			if chosen.Track.Title != "" {
				track = fmt.Sprintf("%s - %s (%s)", chosen.Track.GrandparentTitle, chosen.Track.Title, chosen.Track.ParentTitle)
			}
			isPlaying = chosen.State == "playing"
			duration = chosen.Duration
			position = chosen.Time
			volume = chosen.Volume
		}

		return trackMsgWithState{
			Selected:  selected,
			TrackText: track,
			TrackKey:  trackKey,
			IsPlaying: isPlaying,
			Duration:  duration,
			Position:  position,
			Volume:    volume,
			RequestID: reqID,
		}
	}
}

// =====================
// Helpers
// =====================

func (m *model) beginPlaybackRefresh(pendingText string) tea.Cmd {
	return m.beginPlaybackRefreshForTrack(pendingText, "")
}

func (m *model) beginPlaybackRefreshForTrack(pendingText, trackKey string) tea.Cmd {
	m.playback.beginRefresh(pendingText, trackKey, time.Now())
	return m.pollTimeline()
}

func (m *model) beginPlaybackPendingForTrack(pendingText, trackKey string) {
	m.playback.beginPending(pendingText, trackKey, time.Now())
}

func (m model) currentPosition() int {
	return m.playback.currentPosition(time.Now())
}

func formatTime(ms int) string {
	if ms <= 0 {
		return "0:00"
	}
	sec := ms / 1000
	m := sec / 60
	s := sec % 60
	return fmt.Sprintf("%d:%02d", m, s)
}

func progressBar(pos, dur, width int) string {
	if dur <= 0 || width <= 0 {
		bar := "["
		for i := 0; i < width; i++ {
			bar += "-"
		}
		bar += "]"
		return bar
	}
	f := float64(pos) / float64(dur)
	if f < 0 {
		f = 0
	}
	if f > 1 {
		f = 1
	}
	filled := int(f * float64(width))
	bar := "["
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "#"
		} else {
			bar += "-"
		}
	}
	bar += "]"
	return bar
}
