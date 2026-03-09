package ui

import (
	"testing"
	"time"

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

func TestPlaybackTriggeredIgnoresOldTrackEchoUntilTrackChanges(t *testing.T) {
	initTestLogger(t)

	m := model{
		timelineRequestID: 5,
		currentTrack:      "Artist - Old Track (Album)",
		currentTrackKey:   "old-key",
		durationMs:        200000,
		positionMs:        90000,
		lastUpdate:        time.Now(),
	}

	updatedModel, cmd := m.Update(playbackTriggeredMsg{success: true})
	if cmd != nil {
		t.Fatalf("expected nil command when no player is selected, got non-nil")
	}

	updated := updatedModel.(model)
	if updated.currentTrack != "Loading..." {
		t.Fatalf("expected pending track text after trigger, got %q", updated.currentTrack)
	}
	if updated.timelineRequestID != 6 {
		t.Fatalf("expected timeline request ID to increment, got %d", updated.timelineRequestID)
	}

	echoModel, echoCmd := updated.Update(trackMsgWithState{
		RequestID: updated.timelineRequestID,
		TrackText: "Artist - Old Track (Album)",
		TrackKey:  "old-key",
		IsPlaying: true,
		Duration:  200000,
		Position:  91000,
		Volume:    70,
	})
	if echoCmd != nil {
		t.Fatalf("expected nil command for stale echo timeline update, got non-nil")
	}

	echo := echoModel.(model)
	if echo.currentTrack != "Loading..." {
		t.Fatalf("expected stale echo to be ignored, got currentTrack=%q", echo.currentTrack)
	}
	if echo.positionMs != 0 {
		t.Fatalf("expected playhead to remain reset, got %d", echo.positionMs)
	}

	finalModel, finalCmd := echo.Update(trackMsgWithState{
		RequestID: echo.timelineRequestID,
		TrackText: "Artist - New Track (New Album)",
		TrackKey:  "new-key",
		IsPlaying: true,
		Duration:  180000,
		Position:  1000,
		Volume:    70,
	})
	if finalCmd != nil {
		t.Fatalf("expected nil command for applied timeline update, got non-nil")
	}

	final := finalModel.(model)
	if final.currentTrack != "Artist - New Track (New Album)" {
		t.Fatalf("expected new track to apply, got %q", final.currentTrack)
	}
	if final.currentTrackKey != "new-key" {
		t.Fatalf("expected current track key to update, got %q", final.currentTrackKey)
	}
}

func TestPlaybackTriggeredDoesNotBlockRestartNearBeginning(t *testing.T) {
	initTestLogger(t)

	m := model{
		timelineRequestID: 8,
		currentTrack:      "Artist - Track (Album)",
		currentTrackKey:   "same-key",
		durationMs:        200000,
		positionMs:        900,
		lastUpdate:        time.Now(),
	}

	updatedModel, cmd := m.Update(playbackTriggeredMsg{success: true})
	if cmd != nil {
		t.Fatalf("expected nil command when no player is selected, got non-nil")
	}

	updated := updatedModel.(model)
	restartModel, restartCmd := updated.Update(trackMsgWithState{
		RequestID: updated.timelineRequestID,
		TrackText: "Artist - Track (Album)",
		TrackKey:  "same-key",
		IsPlaying: true,
		Duration:  200000,
		Position:  0,
		Volume:    70,
	})
	if restartCmd != nil {
		t.Fatalf("expected nil command for timeline update, got non-nil")
	}

	restarted := restartModel.(model)
	if restarted.currentTrack != "Artist - Track (Album)" {
		t.Fatalf("expected restart update to apply immediately, got %q", restarted.currentTrack)
	}
	if restarted.positionMs != 0 {
		t.Fatalf("expected position to update to 0, got %d", restarted.positionMs)
	}
}
