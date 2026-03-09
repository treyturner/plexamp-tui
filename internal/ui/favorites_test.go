package ui

import (
	"testing"

	"plexamp-tui/internal/config"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func TestPlaybackEnterDrillsDownToArtistAlbums(t *testing.T) {
	initTestLogger(t)

	m := model{
		panelMode: "playback",
		playbackList: list.New(
			[]list.Item{item{Name: "Artist A", Type: "artist", MetadataKey: "artist-a"}},
			list.NewDefaultDelegate(),
			0,
			0,
		),
		playbackConfig: &config.Favorites{
			Items: []config.FavoriteItem{
				{Name: "Artist A", Type: "artist", MetadataKey: "artist-a"},
			},
		},
	}

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected fetch command for artist drill-down")
	}

	updated := updatedModel.(model)
	if updated.panelMode != "plex-artist-albums" {
		t.Fatalf("expected panelMode plex-artist-albums, got %q", updated.panelMode)
	}
	if updated.currentArtistKey != "artist-a" {
		t.Fatalf("expected currentArtistKey artist-a, got %q", updated.currentArtistKey)
	}
	if updated.artistAlbumReturnMode != "playback" {
		t.Fatalf("expected artistAlbumReturnMode playback, got %q", updated.artistAlbumReturnMode)
	}
}

func TestPlaybackEnterDrillsDownToAlbumTracks(t *testing.T) {
	initTestLogger(t)

	m := model{
		panelMode: "playback",
		playbackList: list.New(
			[]list.Item{item{Name: "Album A", Type: "album", MetadataKey: "album-a"}},
			list.NewDefaultDelegate(),
			0,
			0,
		),
		playbackConfig: &config.Favorites{
			Items: []config.FavoriteItem{
				{Name: "Album A", Type: "album", MetadataKey: "album-a"},
			},
		},
	}

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected fetch command for album drill-down")
	}

	updated := updatedModel.(model)
	if updated.panelMode != "plex-album-tracks" {
		t.Fatalf("expected panelMode plex-album-tracks, got %q", updated.panelMode)
	}
	if updated.currentAlbumKey != "album-a" {
		t.Fatalf("expected currentAlbumKey album-a, got %q", updated.currentAlbumKey)
	}
	if updated.trackReturnMode != "playback" {
		t.Fatalf("expected trackReturnMode playback, got %q", updated.trackReturnMode)
	}
}

func TestPlaybackEnterDrillsDownToPlaylistTracks(t *testing.T) {
	initTestLogger(t)

	m := model{
		panelMode: "playback",
		playbackList: list.New(
			[]list.Item{item{Name: "Playlist A", Type: "playlist", MetadataKey: "playlist-a"}},
			list.NewDefaultDelegate(),
			0,
			0,
		),
		playbackConfig: &config.Favorites{
			Items: []config.FavoriteItem{
				{Name: "Playlist A", Type: "playlist", MetadataKey: "playlist-a"},
			},
		},
	}

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected fetch command for playlist drill-down")
	}

	updated := updatedModel.(model)
	if updated.panelMode != "plex-playlist-tracks" {
		t.Fatalf("expected panelMode plex-playlist-tracks, got %q", updated.panelMode)
	}
	if updated.currentPlaylistKey != "playlist-a" {
		t.Fatalf("expected currentPlaylistKey playlist-a, got %q", updated.currentPlaylistKey)
	}
	if updated.trackReturnMode != "playback" {
		t.Fatalf("expected trackReturnMode playback, got %q", updated.trackReturnMode)
	}
}

func TestPlaybackPTriggersDirectPlay(t *testing.T) {
	initTestLogger(t)

	m := model{
		panelMode: "playback",
		playbackList: list.New(
			[]list.Item{item{Name: "Album A", Type: "album", MetadataKey: "album-a"}},
			list.NewDefaultDelegate(),
			0,
			0,
		),
		playbackConfig: &config.Favorites{
			Items: []config.FavoriteItem{
				{Name: "Album A", Type: "album", MetadataKey: "album-a"},
			},
		},
		selected: "127.0.0.1",
		config:   &config.Config{},
	}

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'P'}})
	if cmd == nil {
		t.Fatalf("expected playback command for P key")
	}

	updated := updatedModel.(model)
	if updated.panelMode != "playback" {
		t.Fatalf("expected to remain in playback mode, got %q", updated.panelMode)
	}
	if updated.lastCommand != "Playing Album A" {
		t.Fatalf("expected direct-play command text, got %q", updated.lastCommand)
	}
	if updated.status != "Starting playback for Album A..." {
		t.Fatalf("expected immediate playback status, got %q", updated.status)
	}
}

func TestTriggerFavoritePlaybackReturnsPlaybackTriggeredMsg(t *testing.T) {
	initTestLogger(t)

	m := model{}
	cmd := m.triggerFavoritePlayback(config.FavoriteItem{
		Name:        "Album A",
		Type:        "album",
		MetadataKey: "album-a",
	})
	if cmd == nil {
		t.Fatalf("expected command")
	}

	msg := cmd()
	playbackMsg, ok := msg.(playbackTriggeredMsg)
	if !ok {
		t.Fatalf("expected playbackTriggeredMsg, got %T", msg)
	}
	if playbackMsg.success {
		t.Fatalf("expected failure when no server selected")
	}
}
