package ui

import (
	"fmt"
	"plexamp-tui/internal/plex"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type playerSelectMsg struct {
	success bool
	err     error
	player  playerItem
}

// playerItem represents a player in the list
type playerItem struct {
	title            string
	clientIdentifier string
	address          string
	local            string
	port             string
}

// playersFetchedMsg is a message containing fetched players
type playersFetchedMsg struct {
	players []plex.PlexConnectionSelection
	err     error
}

// Title returns the playlist title
func (i playerItem) Title() string {
	return fmt.Sprintf("%s - %s", i.title, i.address)
}

// Description returns the playlist description (empty for now)
func (i playerItem) Description() string { return "" }

// FilterValue implements list.Item
func (i playerItem) FilterValue() string {
	return i.title + " " + i.clientIdentifier
}

// fetchPlayersCmd fetches players from the Plex server
func (m *model) fetchPlayersCmd() tea.Cmd {
	m.debug("Fetching players...")
	resizeBrowseListForFetch(&m.playerList, m.width, m.height)
	if m.config == nil {
		return func() tea.Msg {
			return playersFetchedMsg{err: fmt.Errorf("no config available")}
		}
	}

	token := m.deps.plexClient.GetPlexToken()
	if token == "" {
		return func() tea.Msg {
			return playersFetchedMsg{err: fmt.Errorf("no Plex token found - run with --auth flag")}
		}
	}

	return func() tea.Msg {
		players, err := m.deps.plexClient.GetPlexPlayers()
		return playersFetchedMsg{players: players, err: err}
	}
}

// initPlayerBrowse creates a new player browser
func (m *model) initPlayerBrowse() {
	m.panelMode = panelModePlexPlayers
	m.status = "Loading players..."

	items := []list.Item{playerItem{title: "Loading players..."}}
	m.playerList = newBrowseList("Plex Players", items, m.width, m.height)
}
func (m *model) selectPlayerCmd(player playerItem) tea.Cmd {
	if m.config == nil {
		return func() tea.Msg {
			return playerSelectMsg{success: false, err: fmt.Errorf("no config available")}
		}
	}

	// Return a message with the player information
	return func() tea.Msg {
		return playerSelectMsg{
			success: true,
			player:  player,
		}
	}
}

func (m *model) handlePlayerBrowseUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.debug("handlePlayerBrowseUpdate received message: %T", msg)

	// If we're in filtering mode, let the list handle the input
	if m.playerList.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.playerList, cmd = m.playerList.Update(msg)
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
			// Select Server
			if selected, ok := m.playerList.SelectedItem().(playerItem); ok {
				m.debug("Selecting player: %s (clientIdentifier: %s)", selected.title, selected.clientIdentifier)
				m.lastCommand = fmt.Sprintf("Selecting %s", selected.title)
				return m, m.selectPlayerCmd(selected)
			}
			return m, nil

		case "R":
			// Refresh player list
			m.status = "Refreshing players..."
			return m, m.fetchPlayersCmd()

		default:

			// Otherwise try the common controls
			if cmd, handled := m.handleControl(key); handled {
				return m, cmd
			}
		}

	case playersFetchedMsg:
		m.debug("playersFetchedMsg received with %d players, error: %v", len(msg.players), msg.err)
		if msg.err != nil {
			errMsg := fmt.Sprintf("Error fetching players: %v", msg.err)
			m.status = errMsg
			m.debug("%s", errMsg)
			return m, nil
		}

		// Convert servers to list items
		var items []list.Item
		for i, player := range msg.players {
			if i < 5 { // Only log first 5 servers to avoid log spam
				m.debug("Adding player %d: %s (ratingKey: %s)", i+1, player.Name, player.ClientIdentifier)
			}
			items = append(items, playerItem{
				title:            player.Name,
				clientIdentifier: player.ClientIdentifier,
				address:          player.Address,
				local:            player.Local,
				port:             player.Port,
			})
		}

		m.debug("Creating new list with %d items", len(items))
		replaceBrowseListItems(&m.playerList, items)
		m.status = fmt.Sprintf("Loaded %d players", len(msg.players))
		m.debug("Updated model with new player list. List has %d items", m.playerList.VisibleItems())

		// Force a redraw
		return m, tea.Batch(tea.ClearScreen, func() tea.Msg { return nil })

	case playerSelectMsg:
		if msg.success {
			m.lastCommand = "Player Selected"
			m.status = "Player selected successfully"
		} else {
			m.lastCommand = "Player Selection Failed"
			m.status = fmt.Sprintf("Player selection error: %v", msg.err)
		}
		// Return the updated model and no command
		return m, nil
	}

	// Update the player list and get the command
	var listCmd tea.Cmd
	m.playerList, listCmd = m.playerList.Update(msg)
	// Return the current model (as a pointer) and the command
	return m, listCmd
}

// View renders the player browser
func (m *model) ViewPlayer() string {
	return m.playerList.View() + "\n" + m.status
}
