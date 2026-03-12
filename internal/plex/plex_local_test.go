package plex

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"plexamp-tui/internal/logger"
)

func TestFetchAlbumTracksSortsByDiscThenTrack(t *testing.T) {
	const albumKey = "album123"
	const token = "test-token"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/library/metadata/"+albumKey+"/children" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<MediaContainer size="4">
  <Track ratingKey="201" title="Disc 2 Track 1" parentIndex="2" index="1" duration="180000" />
  <Track ratingKey="102" title="Disc 1 Track 2" parentIndex="1" index="2" duration="180000" />
  <Track ratingKey="101" title="Disc 1 Track 1" parentIndex="1" index="1" duration="180000" />
  <Track ratingKey="202" title="Disc 2 Track 2" parentIndex="2" index="2" duration="180000" />
</MediaContainer>`))
	}))
	defer server.Close()

	testLogger, err := logger.NewLogger(false, "")
	if err != nil {
		t.Fatalf("failed to create test logger: %v", err)
	}

	client := NewPlexClient(testLogger)
	tracks, err := client.FetchAlbumTracks(server.URL, albumKey, token)
	if err != nil {
		t.Fatalf("FetchAlbumTracks returned error: %v", err)
	}

	if len(tracks) != 4 {
		t.Fatalf("expected 4 tracks, got %d", len(tracks))
	}

	gotOrder := []string{tracks[0].RatingKey, tracks[1].RatingKey, tracks[2].RatingKey, tracks[3].RatingKey}
	wantOrder := []string{"101", "102", "201", "202"}

	for i := range wantOrder {
		if gotOrder[i] != wantOrder[i] {
			t.Fatalf("unexpected track order at index %d: got %v want %v", i, gotOrder, wantOrder)
		}
	}
}
