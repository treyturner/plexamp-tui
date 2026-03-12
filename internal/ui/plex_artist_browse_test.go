package ui

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func TestArtistPlayKeyUpdatesStatusImmediately(t *testing.T) {
	initTestLogger(t)

	m := model{
		panelMode: "plex-artists",
		artistList: list.New(
			[]list.Item{
				artistItem{
					title:     "Artist A",
					ratingKey: "artist-a",
				},
			},
			list.NewDefaultDelegate(),
			0,
			0,
		),
	}

	_, cmd := m.handleArtistBrowseUpdate(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'P'},
	})

	if cmd == nil {
		t.Fatalf("expected play command to be returned")
	}
	if m.lastCommand != "Playing Artist A" {
		t.Fatalf("expected lastCommand to update, got %q", m.lastCommand)
	}
	if m.status != "Starting playback for Artist A..." {
		t.Fatalf("expected immediate status update, got %q", m.status)
	}
}

func TestArtistBrowseFavoriteIgnoresItemWithoutRatingKey(t *testing.T) {
	initTestLogger(t)

	m := model{
		panelMode:    "plex-artists",
		status:       "Loading artists...",
		lastCommand:  "existing",
		artistList:   list.New([]list.Item{artistItem{title: "Loading artists..."}}, list.NewDefaultDelegate(), 0, 0),
		playbackList: list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
	}

	updatedModel, cmd := m.handleArtistBrowseUpdate(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if cmd != nil {
		t.Fatalf("expected nil command when selected artist has no rating key, got non-nil")
	}

	updated := updatedModel.(*model)
	if updated.panelMode != "plex-artists" {
		t.Fatalf("expected panelMode to stay on artists, got %q", updated.panelMode)
	}
	if updated.lastCommand != "existing" {
		t.Fatalf("expected lastCommand to remain unchanged, got %q", updated.lastCommand)
	}
	if len(updated.playbackList.Items()) != 0 {
		t.Fatalf("expected playback list to remain unchanged, got %d items", len(updated.playbackList.Items()))
	}
	selected, ok := updated.artistList.SelectedItem().(artistItem)
	if !ok {
		t.Fatalf("expected selected item to be artistItem, got %T", updated.artistList.SelectedItem())
	}
	if selected.title != "Loading artists..." {
		t.Fatalf("expected placeholder artist title to remain unchanged, got %q", selected.title)
	}
}
