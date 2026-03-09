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
