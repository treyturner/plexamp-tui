package ui

import (
	"testing"

	"plexamp-tui/internal/logger"
)

func initTestLogger(t *testing.T) {
	t.Helper()

	if log != nil {
		return
	}

	l, err := logger.NewLogger(false, "")
	if err != nil {
		t.Fatalf("failed to init test logger: %v", err)
	}
	log = l
}

func TestTrackPlaybackMsgIgnoresStaleResponse(t *testing.T) {
	initTestLogger(t)

	m := model{
		trackPlaybackReqID: 2,
		currentTrack:       "Artist - Old Track (Album)",
		status:             "existing",
		lastCommand:        "existing",
	}

	updatedModel, cmd := m.Update(trackPlaybackMsg{
		success:   true,
		requestID: 1,
	})
	if cmd != nil {
		t.Fatalf("expected no command for stale response, got non-nil")
	}

	updated := updatedModel.(model)
	if updated.currentTrack != "Artist - Old Track (Album)" {
		t.Fatalf("expected current track to remain unchanged, got %q", updated.currentTrack)
	}
	if updated.status != "existing" {
		t.Fatalf("expected status to remain unchanged, got %q", updated.status)
	}
}

func TestTrackPlaybackMsgAppliesLatestResponse(t *testing.T) {
	initTestLogger(t)

	m := model{
		trackPlaybackReqID: 2,
		selected:           "",
	}

	updatedModel, cmd := m.Update(trackPlaybackMsg{
		success:   true,
		requestID: 2,
	})
	if cmd != nil {
		t.Fatalf("expected nil command when no player is selected, got non-nil")
	}

	updated := updatedModel.(model)
	if updated.lastCommand != "Track Playback Started" {
		t.Fatalf("expected lastCommand to update, got %q", updated.lastCommand)
	}
	if updated.status != "Playback triggered successfully" {
		t.Fatalf("expected playback success status, got %q", updated.status)
	}
	if updated.currentTrack != "Loading..." {
		t.Fatalf("expected pending track text, got %q", updated.currentTrack)
	}
}

func TestTimelineUpdateClearsPendingOnNonRequestedTrackKey(t *testing.T) {
	initTestLogger(t)

	m := model{
		timelineRequestID: 3,
		pendingTrackKey:   "222",
		currentTrack:      "Loading...",
		status:            "Playback triggered successfully",
	}

	updatedModel, cmd := m.Update(trackMsgWithState{
		RequestID: 3,
		TrackText: "Artist - Old Track (Album)",
		TrackKey:  "111",
		IsPlaying: true,
		Duration:  100000,
		Position:  25000,
		Volume:    70,
	})
	if cmd != nil {
		t.Fatalf("expected nil command for timeline update, got non-nil")
	}

	updated := updatedModel.(model)
	if updated.currentTrack != "Artist - Old Track (Album)" {
		t.Fatalf("expected current track to update, got %q", updated.currentTrack)
	}
	if updated.pendingTrackKey != "" {
		t.Fatalf("expected pending track key to clear, got %q", updated.pendingTrackKey)
	}
	if updated.positionMs != 25000 {
		t.Fatalf("expected position to update, got %d", updated.positionMs)
	}
}

func TestTimelineUpdateAppliesRequestedTrackKey(t *testing.T) {
	initTestLogger(t)

	m := model{
		timelineRequestID: 3,
		pendingTrackKey:   "222",
		currentTrack:      "Loading...",
	}

	updatedModel, cmd := m.Update(trackMsgWithState{
		RequestID: 3,
		TrackText: "Artist - New Track (Album)",
		TrackKey:  "222",
		IsPlaying: true,
		Duration:  90000,
		Position:  1000,
		Volume:    65,
	})
	if cmd != nil {
		t.Fatalf("expected nil command for timeline update, got non-nil")
	}

	updated := updatedModel.(model)
	if updated.currentTrack != "Artist - New Track (Album)" {
		t.Fatalf("expected current track to update, got %q", updated.currentTrack)
	}
	if updated.pendingTrackKey != "" {
		t.Fatalf("expected pending track key to clear, got %q", updated.pendingTrackKey)
	}
	if updated.positionMs != 1000 {
		t.Fatalf("expected position to update, got %d", updated.positionMs)
	}
}

func TestTimelineUpdateKeepsPendingWhenTrackKeyIsMissing(t *testing.T) {
	initTestLogger(t)

	m := model{
		timelineRequestID: 3,
		pendingTrackKey:   "222",
		currentTrack:      "Loading...",
	}

	updatedModel, cmd := m.Update(trackMsgWithState{
		RequestID: 3,
		TrackText: "Artist - Pending Resolution (Album)",
		TrackKey:  "",
		IsPlaying: true,
		Duration:  90000,
		Position:  1000,
		Volume:    65,
	})
	if cmd != nil {
		t.Fatalf("expected nil command for timeline update, got non-nil")
	}

	updated := updatedModel.(model)
	if updated.currentTrack != "Artist - Pending Resolution (Album)" {
		t.Fatalf("expected current track to update, got %q", updated.currentTrack)
	}
	if updated.pendingTrackKey != "222" {
		t.Fatalf("expected pending track key to remain set, got %q", updated.pendingTrackKey)
	}
}
