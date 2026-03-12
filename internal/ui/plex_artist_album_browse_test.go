package ui

import (
	"testing"

	"plexamp-tui/internal/plex"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func TestArtistAlbumsFetchedMsgIgnoresStaleResponse(t *testing.T) {
	initTestLogger(t)

	m := model{
		panelMode:        "plex-artist-albums",
		currentArtistKey: "artist-b",
		status:           "existing",
		artistAlbumList: list.New(
			[]list.Item{
				albumItem{
					title:     "Current Album",
					artist:    "Artist B",
					year:      "2024",
					ratingKey: "alb-b",
				},
			},
			list.NewDefaultDelegate(),
			0,
			0,
		),
		playbackList: list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
	}

	updatedModel, cmd := m.Update(artistAlbumsFetchedMsg{
		requestKey: "artist-a",
		albums: []plex.PlexAlbum{
			{
				Title:       "Should Be Ignored",
				ParentTitle: "Artist A",
				Year:        "2026",
				RatingKey:   "alb-a",
			},
		},
	})
	if cmd != nil {
		t.Fatalf("expected no command for stale response, got non-nil")
	}

	updated := updatedModel.(model)
	if updated.status != "existing" {
		t.Fatalf("expected status to remain unchanged, got %q", updated.status)
	}

	items := updated.artistAlbumList.Items()
	if len(items) != 1 {
		t.Fatalf("expected 1 existing album item, got %d", len(items))
	}
	selected, ok := items[0].(albumItem)
	if !ok {
		t.Fatalf("expected albumItem, got %T", items[0])
	}
	if selected.title != "Current Album" || selected.ratingKey != "alb-b" {
		t.Fatalf("expected existing album to remain, got title=%q ratingKey=%q", selected.title, selected.ratingKey)
	}
}

func TestArtistAlbumsFetchedMsgAppliesMatchingResponse(t *testing.T) {
	initTestLogger(t)

	m := model{
		panelMode:        "plex-artist-albums",
		currentArtistKey: "artist-b",
		artistAlbumList:  list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
		playbackList:     list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
	}

	updatedModel, _ := m.Update(artistAlbumsFetchedMsg{
		requestKey: "artist-b",
		albums: []plex.PlexAlbum{
			{
				Title:       "New Album",
				ParentTitle: "Artist B",
				Year:        "2026",
				RatingKey:   "alb-new",
			},
		},
	})

	updated := updatedModel.(model)
	items := updated.artistAlbumList.Items()
	if len(items) != 1 {
		t.Fatalf("expected 1 album item, got %d", len(items))
	}

	selected, ok := items[0].(albumItem)
	if !ok {
		t.Fatalf("expected albumItem, got %T", items[0])
	}
	if selected.title != "New Album" || selected.ratingKey != "alb-new" {
		t.Fatalf("unexpected album item values: title=%q ratingKey=%q", selected.title, selected.ratingKey)
	}
	if updated.status != "Loaded 1 albums" {
		t.Fatalf("expected success status, got %q", updated.status)
	}
}

func TestArtistAlbumBrowseEscReturnsToOriginPanel(t *testing.T) {
	initTestLogger(t)

	m := model{
		panelMode:             "plex-artist-albums",
		artistAlbumReturnMode: "playback",
		artistAlbumList:       list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
	}

	updatedModel, cmd := m.handleArtistAlbumBrowseUpdate(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Fatalf("expected nil command on escape, got non-nil")
	}

	updated := updatedModel.(*model)
	if updated.panelMode != "playback" {
		t.Fatalf("expected panelMode playback, got %q", updated.panelMode)
	}
}

func TestInitArtistAlbumBrowseCapturesCurrentPanelAsReturnMode(t *testing.T) {
	initTestLogger(t)

	m := model{panelMode: "playback"}
	m.initArtistAlbumBrowse(artistItem{
		title:     "Artist A",
		ratingKey: "artist-a",
	})

	if m.panelMode != "plex-artist-albums" {
		t.Fatalf("expected panelMode plex-artist-albums, got %q", m.panelMode)
	}
	if m.artistAlbumReturnMode != "playback" {
		t.Fatalf("expected artistAlbumReturnMode playback, got %q", m.artistAlbumReturnMode)
	}
}

func TestArtistAlbumBrowseEnterIgnoresItemWithoutRatingKey(t *testing.T) {
	initTestLogger(t)

	m := model{
		panelMode:       "plex-artist-albums",
		status:          "Loading albums for Artist A...",
		artistAlbumList: list.New([]list.Item{albumItem{title: "Loading albums..."}}, list.NewDefaultDelegate(), 0, 0),
	}

	updatedModel, cmd := m.handleArtistAlbumBrowseUpdate(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("expected nil command when selected album has no rating key, got non-nil")
	}

	updated := updatedModel.(*model)
	if updated.panelMode != "plex-artist-albums" {
		t.Fatalf("expected panelMode to stay on artist albums, got %q", updated.panelMode)
	}
	if updated.trackReturnMode != "" {
		t.Fatalf("expected trackReturnMode to remain unchanged, got %q", updated.trackReturnMode)
	}
	if updated.currentAlbumKey != "" {
		t.Fatalf("expected currentAlbumKey to remain unset, got %q", updated.currentAlbumKey)
	}
}
