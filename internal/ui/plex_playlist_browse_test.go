package ui

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func TestPlaylistBrowseFavoriteIgnoresItemWithoutRatingKey(t *testing.T) {
	initTestLogger(t)

	m := model{
		panelMode:    "plex-playlists",
		status:       "Loading playlists...",
		lastCommand:  "existing",
		playlistList: list.New([]list.Item{playlistItem{title: "Loading playlists..."}}, list.NewDefaultDelegate(), 0, 0),
		playbackList: list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
	}

	updatedModel, cmd := m.handlePlaylistBrowseUpdate(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if cmd != nil {
		t.Fatalf("expected nil command when selected playlist has no rating key, got non-nil")
	}

	updated := updatedModel.(*model)
	if updated.panelMode != "plex-playlists" {
		t.Fatalf("expected panelMode to stay on playlists, got %q", updated.panelMode)
	}
	if updated.lastCommand != "existing" {
		t.Fatalf("expected lastCommand to remain unchanged, got %q", updated.lastCommand)
	}
	if len(updated.playbackList.Items()) != 0 {
		t.Fatalf("expected playback list to remain unchanged, got %d items", len(updated.playbackList.Items()))
	}
	selected, ok := updated.playlistList.SelectedItem().(playlistItem)
	if !ok {
		t.Fatalf("expected selected item to be playlistItem, got %T", updated.playlistList.SelectedItem())
	}
	if selected.title != "Loading playlists..." {
		t.Fatalf("expected placeholder playlist title to remain unchanged, got %q", selected.title)
	}
}
