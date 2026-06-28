package ui

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func TestAlbumBrowseFavoriteIgnoresItemWithoutRatingKey(t *testing.T) {
	initTestLogger(t)

	m := model{
		panelMode:    panelModePlexAlbums,
		status:       "Loading albums...",
		lastCommand:  "existing",
		albumList:    list.New([]list.Item{albumItem{title: "Loading albums..."}}, list.NewDefaultDelegate(), 0, 0),
		playbackList: list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
	}

	updatedModel, cmd := m.handleAlbumBrowseUpdate(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if cmd != nil {
		t.Fatalf("expected nil command when selected album has no rating key, got non-nil")
	}

	updated := updatedModel.(*model)
	if updated.panelMode != panelModePlexAlbums {
		t.Fatalf("expected panelMode to stay on albums, got %q", updated.panelMode)
	}
	if updated.lastCommand != "existing" {
		t.Fatalf("expected lastCommand to remain unchanged, got %q", updated.lastCommand)
	}
	if len(updated.playbackList.Items()) != 0 {
		t.Fatalf("expected playback list to remain unchanged, got %d items", len(updated.playbackList.Items()))
	}
	selected, ok := updated.albumList.SelectedItem().(albumItem)
	if !ok {
		t.Fatalf("expected selected item to be albumItem, got %T", updated.albumList.SelectedItem())
	}
	if selected.title != "Loading albums..." {
		t.Fatalf("expected placeholder album title to remain unchanged, got %q", selected.title)
	}
}
