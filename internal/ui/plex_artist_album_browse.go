package ui

import (
	"fmt"

	"plexamp-tui/internal/plex"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type artistAlbumsFetchedMsg struct {
	albums     []plex.PlexAlbum
	requestKey string
	err        error
}

func (m *model) fetchArtistAlbumsCmd(artistRatingKey string) tea.Cmd {
	log.Debug("Fetching artist albums...")
	footerHeight := 3
	availableHeight := m.height - footerHeight - 5
	m.artistAlbumList.SetSize(m.width/2-4, availableHeight)

	if m.config == nil {
		return func() tea.Msg {
			return artistAlbumsFetchedMsg{
				requestKey: artistRatingKey,
				err:        fmt.Errorf("no config available"),
			}
		}
	}

	token := plexClient.GetPlexToken()
	if token == "" {
		return func() tea.Msg {
			return artistAlbumsFetchedMsg{
				requestKey: artistRatingKey,
				err:        fmt.Errorf("no Plex token found - run with --auth flag"),
			}
		}
	}

	serverAddr := m.config.PlexServerAddr

	return func() tea.Msg {
		albums, err := plexClient.FetchArtistAlbums(serverAddr, artistRatingKey, token)
		return artistAlbumsFetchedMsg{
			albums:     albums,
			requestKey: artistRatingKey,
			err:        err,
		}
	}
}

func (m *model) initArtistAlbumBrowse(artist artistItem) {
	m.artistAlbumReturnMode = m.panelMode
	m.panelMode = "plex-artist-albums"
	m.status = fmt.Sprintf("Loading albums for %s...", artist.title)
	m.currentArtistKey = artist.ratingKey
	m.currentArtistName = artist.title

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false

	items := []list.Item{albumItem{title: "Loading albums..."}}
	m.artistAlbumList = list.New(items, delegate, 0, 0)
	m.artistAlbumList.Title = fmt.Sprintf("Albums - %s", artist.title)
	m.artistAlbumList.SetShowFilter(true)
	m.artistAlbumList.SetFilteringEnabled(true)
	m.artistAlbumList.Styles.Title = titleStyle
	m.artistAlbumList.Styles.PaginationStyle = paginationStyle
	m.artistAlbumList.Styles.HelpStyle = helpStyle
	m.artistAlbumList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("f"),
				key.WithHelp("f", "favs"),
			),
		}
	}
	m.artistAlbumList.AdditionalFullHelpKeys = func() []key.Binding {
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

	if m.width > 0 && m.height > 0 {
		m.artistAlbumList.SetSize(m.width/2-4, m.height-4)
	}
}

func (m *model) handleArtistAlbumBrowseUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Debug(fmt.Sprintf("handleArtistAlbumBrowseUpdate received message: %T", msg))

	if m.artistAlbumList.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.artistAlbumList, cmd = m.artistAlbumList.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		switch key {
		case "esc", "q":
			if m.artistAlbumReturnMode != "" {
				m.panelMode = m.artistAlbumReturnMode
			} else {
				m.panelMode = "plex-artists"
			}
			m.status = ""
			return m, nil

		case "enter":
			if selected, ok := m.artistAlbumList.SelectedItem().(albumItem); ok {
				if selected.ratingKey == "" {
					log.Debug("Ignoring album track browse for item without rating key")
					return m, nil
				}
				log.Debug(fmt.Sprintf("Opening album tracks: %s (ratingKey: %s)", selected.title, selected.ratingKey))
				m.lastCommand = fmt.Sprintf("Viewing %s", selected.title)
				m.trackReturnMode = "plex-artist-albums"
				m.initAlbumTrackBrowse(selected.title, selected.ratingKey)
				return m, m.fetchAlbumTracksCmd(selected.ratingKey)
			}
			return m, nil

		case "P":
			if selected, ok := m.artistAlbumList.SelectedItem().(albumItem); ok {
				if selected.ratingKey == "" {
					log.Debug("Ignoring album playback for item without rating key")
					return m, nil
				}
				log.Debug(fmt.Sprintf("Playing album: %s (ratingKey: %s)", selected.title, selected.ratingKey))
				m.lastCommand = fmt.Sprintf("Playing %s", selected.title)
				return m, m.playAlbumCmd(selected.ratingKey)
			}
			return m, nil

		case "f":
			if selected, ok := m.artistAlbumList.SelectedItem().(albumItem); ok {
				if selected.ratingKey == "" {
					log.Debug("Ignoring album favorite toggle for item without rating key")
					return m, nil
				}
				log.Debug(fmt.Sprintf("Toggling favorite for album: %s (ratingKey: %s)", selected.title, selected.ratingKey))
				m.lastCommand = fmt.Sprintf("Toggling favorite for %s", selected.title)

				_, cmd := m.addRemoveFavorite(selected.title, selected.ratingKey, "album")
				selected.ToggleFavorite()
				m.artistAlbumList.SetItem(m.artistAlbumList.Index(), selected)
				return m, cmd
			}

		case "R":
			m.status = "Refreshing albums..."
			m.lastCommand = "Refreshing artist albums"
			return m, m.fetchArtistAlbumsCmd(m.currentArtistKey)

		default:
			if cmd, handled := m.handleControl(key); handled {
				return m, cmd
			}
		}

	case artistAlbumsFetchedMsg:
		log.Debug(fmt.Sprintf(
			"artistAlbumsFetchedMsg received with %d albums, requestKey=%s, error: %v",
			len(msg.albums), msg.requestKey, msg.err),
		)
		if msg.requestKey != m.currentArtistKey {
			log.Debug(fmt.Sprintf(
				"Ignoring stale artist album response (requestKey=%s, currentArtistKey=%s, panelMode=%s)",
				msg.requestKey, m.currentArtistKey, m.panelMode),
			)
			return m, nil
		}
		if msg.err != nil {
			errMsg := fmt.Sprintf("Error fetching albums: %v", msg.err)
			m.status = errMsg
			log.Debug(errMsg)
			return m, nil
		}

		favSet := make(map[string]struct{})
		for _, pItem := range m.playbackList.Items() {
			pItem := pItem.(item)
			favSet[pItem.GetMetadataKey()] = struct{}{}
		}

		var items []list.Item
		for i, album := range msg.albums {
			if i < 5 {
				log.Debug(fmt.Sprintf("Adding artist album %d: %s (ratingKey: %s)", i+1, album.Title, album.RatingKey))
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

		filterState := m.artistAlbumList.FilterState()
		filterValue := m.artistAlbumList.FilterValue()

		m.artistAlbumList.SetItems(items)
		m.artistAlbumList.ResetSelected()

		if filterState == list.Filtering {
			m.artistAlbumList.ResetFilter()
			m.artistAlbumList.FilterInput.SetValue(filterValue)
		}

		m.status = fmt.Sprintf("Loaded %d albums", len(msg.albums))
		return m, tea.Batch(tea.ClearScreen, func() tea.Msg { return nil })

	}

	var listCmd tea.Cmd
	m.artistAlbumList, listCmd = m.artistAlbumList.Update(msg)
	return m, listCmd
}
