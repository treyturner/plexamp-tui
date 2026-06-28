package ui

import (
	"fmt"
	"plexamp-tui/internal/config"
	"plexamp-tui/internal/plex"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// serverItem represents a server in the list
type serverItem struct {
	title            string
	clientIdentifier string
	scheme           string
	address          string
	local            string
	port             string
}

// serversFetchedMsg is a message containing fetched servers
type serversFetchedMsg struct {
	servers []plex.PlexConnectionSelection
	err     error
}

type serverSelectMsg struct {
	success   bool
	err       error
	server    serverItem
	libraries []config.PlexLibrary
}

// Title returns the playlist title
func (i serverItem) Title() string {
	return fmt.Sprintf("%s - %s", i.title, i.address)
}

// Description returns the playlist description (empty for now)
func (i serverItem) Description() string { return "" }

// FilterValue implements list.Item
func (i serverItem) FilterValue() string {
	return i.title + " " + i.clientIdentifier
}

// fetchServersCmd fetches servers from the Plex server
func (m *model) fetchServersCmd() tea.Cmd {
	m.debug("Fetching servers...")
	resizeBrowseListForFetch(&m.serverList, m.width, m.height)
	if m.config == nil {
		return func() tea.Msg {
			return serversFetchedMsg{err: fmt.Errorf("no config available")}
		}
	}

	token := m.deps.plexClient.GetPlexToken()
	if token == "" {
		return func() tea.Msg {
			return serversFetchedMsg{err: fmt.Errorf("no Plex token found - run with --auth flag")}
		}
	}

	return func() tea.Msg {
		servers, err := m.deps.plexClient.GetPlexServerInformation()
		return serversFetchedMsg{servers: servers, err: err}
	}
}

// initServerBrowse creates a new server browser
func (m *model) initServerBrowse() {
	m.panelMode = panelModePlexServers
	m.status = "Loading servers..."

	items := []list.Item{serverItem{title: "Loading servers..."}}
	m.serverList = newBrowseList("Plex Servers", items, m.width, m.height)
}
func (m *model) selectServerCmd(server serverItem) tea.Cmd {
	if m.selected == "" {
		return func() tea.Msg {
			return serverSelectMsg{success: false, err: fmt.Errorf("no server selected")}
		}
	}

	if m.config == nil {
		return func() tea.Msg {
			return serverSelectMsg{success: false, err: fmt.Errorf("no config available")}
		}
	}

	return func() tea.Msg {

		serverAddr := fmt.Sprintf("%s:%s", server.address, server.port)
		if server.scheme != "" {
			serverAddr = fmt.Sprintf("%s://%s", server.scheme, serverAddr)
		}
		libraries, err := m.deps.plexClient.FetchLibrary(serverAddr)
		m.debug("Fetched libraries: %v", libraries)

		if err != nil {
			m.debug("Error fetching libraries: %v", err)
		}

		// When a server is selected we will write the serverId and serverAddress:port to the config file and save it to disk
		// m.config.ServerID = server.clientIdentifier
		// m.config.PlexServerAddr = server.address + ":" + server.port
		// m.saveServerConfig()
		return serverSelectMsg{success: true, server: server, libraries: libraries}
	}
}

func (m *model) handleServerBrowseUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.debug("handleServerBrowseUpdate received message: %T", msg)

	// If we're in filtering mode, let the list handle the input
	if m.serverList.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.serverList, cmd = m.serverList.Update(msg)
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
			if selected, ok := m.serverList.SelectedItem().(serverItem); ok {
				m.debug("Selecting server: %s (clientIdentifier: %s)", selected.title, selected.clientIdentifier)
				m.lastCommand = fmt.Sprintf("Selecting %s", selected.title)
				return m, m.selectServerCmd(selected)
			}
			return m, nil

		case "R":
			// Refresh server list
			m.status = "Refreshing servers..."
			return m, m.fetchServersCmd()

		default:

			// Otherwise try the common controls
			if cmd, handled := m.handleControl(key); handled {
				return m, cmd
			}
		}

	case serversFetchedMsg:
		m.debug("serversFetchedMsg received with %d servers, error: %v", len(msg.servers), msg.err)
		if msg.err != nil {
			errMsg := fmt.Sprintf("Error fetching servers: %v", msg.err)
			m.status = errMsg
			m.debug("%s", errMsg)
			return m, nil
		}

		// Convert servers to list items
		var items []list.Item
		for i, server := range msg.servers {
			if i < 5 { // Only log first 5 servers to avoid log spam
				m.debug("Adding server %d: %s (ratingKey: %s)", i+1, server.Name, server.ClientIdentifier)
			}
			items = append(items, serverItem{
				title:            server.Name,
				clientIdentifier: server.ClientIdentifier,
				scheme:           server.Scheme,
				address:          server.Address,
				local:            server.Local,
				port:             server.Port,
			})
		}

		m.debug("Creating new list with %d items", len(items))
		replaceBrowseListItems(&m.serverList, items)
		m.status = fmt.Sprintf("Loaded %d servers", len(msg.servers))
		m.debug("Updated model with new server list. List has %d items", m.serverList.VisibleItems())

		// Force a redraw
		return m, tea.Batch(tea.ClearScreen, func() tea.Msg { return nil })

	case serverSelectMsg:
		if msg.success {
			m.lastCommand = "Server Selected"
			m.status = "Server selected successfully"
		} else {
			m.lastCommand = "Server Selection Failed"
			m.status = fmt.Sprintf("Server selection error: %v", msg.err)
		}
		// Return the updated model and no command
		return m, nil
	}

	// Update the server list and get the command
	var listCmd tea.Cmd
	m.serverList, listCmd = m.serverList.Update(msg)
	// Return the current model (as a pointer) and the command
	return m, listCmd
}

// View renders the server browser
func (m *model) ViewServer() string {
	return m.serverList.View() + "\n" + m.status
}
