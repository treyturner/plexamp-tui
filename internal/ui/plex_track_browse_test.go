package ui

import (
	"errors"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func initTestLogger(t *testing.T) {
	t.Helper()
}

func TestTrackPlaybackMsgIgnoresStaleResponse(t *testing.T) {
	initTestLogger(t)

	m := model{
		trackPlaybackReqID: 2,
		playback: playbackState{
			currentTrack: "Artist - Old Track (Album)",
		},
		status:      "existing",
		lastCommand: "existing",
	}

	updatedModel, cmd := m.Update(trackPlaybackMsg{
		success:   true,
		requestID: 1,
	})
	if cmd != nil {
		t.Fatalf("expected no command for stale response, got non-nil")
	}

	updated := updatedModel.(model)
	if updated.playback.currentTrack != "Artist - Old Track (Album)" {
		t.Fatalf("expected current track to remain unchanged, got %q", updated.playback.currentTrack)
	}
	if updated.status != "existing" {
		t.Fatalf("expected status to remain unchanged, got %q", updated.status)
	}
}

func TestTrackBrowseEnterIgnoresItemWithoutRatingKey(t *testing.T) {
	initTestLogger(t)

	m := model{
		panelMode:          "plex-album-tracks",
		status:             "Loading tracks for Album A...",
		trackPlaybackReqID: 4,
		trackList:          list.New([]list.Item{trackItem{title: "Loading tracks..."}}, list.NewDefaultDelegate(), 0, 0),
		playback: playbackState{
			currentTrack: "Existing Track",
		},
	}

	updatedModel, cmd := m.handleTrackBrowseUpdate(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("expected nil command when selected track has no rating key, got non-nil")
	}

	updated := updatedModel.(*model)
	if updated.trackPlaybackReqID != 4 {
		t.Fatalf("expected trackPlaybackReqID to remain unchanged, got %d", updated.trackPlaybackReqID)
	}
	if updated.playback.currentTrack != "Existing Track" {
		t.Fatalf("expected current track to remain unchanged, got %q", updated.playback.currentTrack)
	}
	if updated.playback.pendingTrackKey != "" {
		t.Fatalf("expected pending track key to remain empty, got %q", updated.playback.pendingTrackKey)
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
	if updated.playback.currentTrack != "Loading..." {
		t.Fatalf("expected pending track text, got %q", updated.playback.currentTrack)
	}
}

func TestTrackPlaybackMsgFromPreviousPlayerIsIgnored(t *testing.T) {
	initTestLogger(t)

	m := model{
		selected:           "player-b",
		trackPlaybackReqID: 2,
		playback: playbackState{
			currentTrack: "Existing Track",
		},
		status:      "existing",
		lastCommand: "existing",
	}

	updatedModel, cmd := m.Update(trackPlaybackMsg{
		success:   true,
		selected:  "player-a",
		requestID: 2,
		ratingKey: "new-track",
	})
	if cmd != nil {
		t.Fatalf("expected no command for old-player track playback response, got non-nil")
	}

	updated := updatedModel.(model)
	if updated.playback.currentTrack != "Existing Track" {
		t.Fatalf("expected current track to remain unchanged, got %q", updated.playback.currentTrack)
	}
	if updated.status != "existing" {
		t.Fatalf("expected status to remain unchanged, got %q", updated.status)
	}
	if updated.lastCommand != "existing" {
		t.Fatalf("expected lastCommand to remain unchanged, got %q", updated.lastCommand)
	}
}

func TestTrackPlaybackFailureUsesEarlierSuccessFromCurrentBurst(t *testing.T) {
	initTestLogger(t)

	m := model{
		selected:           "player-a",
		trackPlaybackReqID: 2,
		playback: playbackState{
			timelineRequestID: 3,
		},
	}

	updatedModel, cmd := m.Update(trackPlaybackMsg{
		success:   true,
		selected:  "player-a",
		requestID: 1,
		ratingKey: "track-a",
	})
	if cmd != nil {
		t.Fatalf("expected stale track playback success to be acknowledged without command")
	}

	updated := updatedModel.(model)
	if updated.ackTrackPlaybackReqID != 1 {
		t.Fatalf("expected acknowledged track playback request id to be 1, got %d", updated.ackTrackPlaybackReqID)
	}
	if updated.ackTrackPlaybackKey != "track-a" {
		t.Fatalf("expected acknowledged track key track-a, got %q", updated.ackTrackPlaybackKey)
	}

	updatedModel, cmd = updated.Update(trackPlaybackMsg{
		success:   false,
		selected:  "player-a",
		requestID: 2,
		ratingKey: "track-b",
		err:       errors.New("request failed"),
	})
	if cmd == nil {
		t.Fatalf("expected timeline refresh after latest track request failed with earlier success acknowledged")
	}

	updated = updatedModel.(model)
	if updated.lastCommand != "Track Playback Started" {
		t.Fatalf("expected track playback started marker, got %q", updated.lastCommand)
	}
	if updated.status != "Playback triggered successfully" {
		t.Fatalf("expected playback success status, got %q", updated.status)
	}
	if updated.playback.pendingTrackKey != "track-a" {
		t.Fatalf("expected pending track key to use acknowledged track-a, got %q", updated.playback.pendingTrackKey)
	}
}

func TestTrackPlaybackRequestClearsEarlierAcknowledgedSuccess(t *testing.T) {
	initTestLogger(t)

	now := time.Now()
	m := model{
		selected:              "player-a",
		trackPlaybackReqID:    1,
		ackTrackPlaybackReqID: 1,
		ackTrackPlaybackKey:   "track-a",
		playback: playbackState{
			timelineRequestID: 3,
			currentTrack:      "Loading track...",
			currentTrackKey:   "old-key",
			isPlaying:         true,
			durationMs:        123000,
			positionMs:        45000,
			lastUpdate:        now,
			suppressTimeline:  true,
			pendingTrackKey:   "track-b",
			pendingTrackUntil: now.Add(8 * time.Second),
		},
	}

	requestID := m.nextTrackPlaybackRequestID()
	if requestID != 2 {
		t.Fatalf("expected new track playback request id 2, got %d", requestID)
	}
	if m.ackTrackPlaybackReqID != 0 {
		t.Fatalf("expected new track playback request to clear acknowledged id, got %d", m.ackTrackPlaybackReqID)
	}
	if m.ackTrackPlaybackKey != "" {
		t.Fatalf("expected new track playback request to clear acknowledged key, got %q", m.ackTrackPlaybackKey)
	}

	updatedModel, cmd := m.Update(trackPlaybackMsg{
		success:   false,
		selected:  "player-a",
		requestID: requestID,
		ratingKey: "track-b",
		err:       errors.New("request failed"),
	})
	if cmd != nil {
		t.Fatalf("expected failed new track playback request to return no command")
	}

	updated := updatedModel.(model)
	if updated.lastCommand != "Playback Failed" {
		t.Fatalf("expected playback failed marker, got %q", updated.lastCommand)
	}
	if updated.status != "Playback error: request failed" {
		t.Fatalf("expected playback failure status, got %q", updated.status)
	}
	if updated.playback.currentTrack != "" {
		t.Fatalf("expected failed latest request to clear current track, got %q", updated.playback.currentTrack)
	}
	if updated.playback.currentTrackKey != "" {
		t.Fatalf("expected failed latest request to clear current track key, got %q", updated.playback.currentTrackKey)
	}
	if updated.playback.isPlaying {
		t.Fatalf("expected failed latest request to clear playing state")
	}
	if updated.playback.durationMs != 0 {
		t.Fatalf("expected failed latest request to clear duration, got %d", updated.playback.durationMs)
	}
	if updated.playback.positionMs != 0 {
		t.Fatalf("expected failed latest request to clear position, got %d", updated.playback.positionMs)
	}
	if updated.playback.pendingTrackKey != "" {
		t.Fatalf("expected failed latest request to clear pending track key, got %q", updated.playback.pendingTrackKey)
	}
}

func TestTimelineUpdateClearsPendingOnNonRequestedTrackKey(t *testing.T) {
	initTestLogger(t)

	m := model{
		playback: playbackState{
			timelineRequestID: 3,
			pendingTrackKey:   "222",
			currentTrack:      "Loading...",
		},
		status: "Playback triggered successfully",
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
	if updated.playback.currentTrack != "Artist - Old Track (Album)" {
		t.Fatalf("expected current track to update, got %q", updated.playback.currentTrack)
	}
	if updated.playback.pendingTrackKey != "" {
		t.Fatalf("expected pending track key to clear, got %q", updated.playback.pendingTrackKey)
	}
	if updated.playback.positionMs != 25000 {
		t.Fatalf("expected position to update, got %d", updated.playback.positionMs)
	}
}

func TestTimelineUpdateAppliesRequestedTrackKey(t *testing.T) {
	initTestLogger(t)

	m := model{
		playback: playbackState{
			timelineRequestID: 3,
			pendingTrackKey:   "222",
			currentTrack:      "Loading...",
		},
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
	if updated.playback.currentTrack != "Artist - New Track (Album)" {
		t.Fatalf("expected current track to update, got %q", updated.playback.currentTrack)
	}
	if updated.playback.pendingTrackKey != "" {
		t.Fatalf("expected pending track key to clear, got %q", updated.playback.pendingTrackKey)
	}
	if updated.playback.positionMs != 1000 {
		t.Fatalf("expected position to update, got %d", updated.playback.positionMs)
	}
}

func TestTimelineUpdateKeepsPendingWhenTrackKeyIsMissing(t *testing.T) {
	initTestLogger(t)

	m := model{
		playback: playbackState{
			timelineRequestID: 3,
			pendingTrackKey:   "222",
			currentTrack:      "Loading...",
		},
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
	if updated.playback.currentTrack != "Artist - Pending Resolution (Album)" {
		t.Fatalf("expected current track to update, got %q", updated.playback.currentTrack)
	}
	if updated.playback.pendingTrackKey != "222" {
		t.Fatalf("expected pending track key to remain set, got %q", updated.playback.pendingTrackKey)
	}
}

func TestPlaybackTriggeredIgnoresOldTrackEchoUntilTrackChanges(t *testing.T) {
	initTestLogger(t)

	m := model{
		playback: playbackState{
			timelineRequestID: 5,
			currentTrack:      "Artist - Old Track (Album)",
			currentTrackKey:   "old-key",
			durationMs:        200000,
			positionMs:        90000,
			lastUpdate:        time.Now(),
		},
	}

	updatedModel, cmd := m.Update(playbackTriggeredMsg{success: true})
	if cmd != nil {
		t.Fatalf("expected nil command when no player is selected, got non-nil")
	}

	updated := updatedModel.(model)
	if updated.playback.currentTrack != "Loading..." {
		t.Fatalf("expected pending track text after trigger, got %q", updated.playback.currentTrack)
	}
	if updated.playback.timelineRequestID != 6 {
		t.Fatalf("expected timeline request ID to increment, got %d", updated.playback.timelineRequestID)
	}

	echoModel, echoCmd := updated.Update(trackMsgWithState{
		RequestID: updated.playback.timelineRequestID,
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
	if echo.playback.currentTrack != "Loading..." {
		t.Fatalf("expected stale echo to be ignored, got currentTrack=%q", echo.playback.currentTrack)
	}
	if echo.playback.positionMs != 0 {
		t.Fatalf("expected playhead to remain reset, got %d", echo.playback.positionMs)
	}

	finalModel, finalCmd := echo.Update(trackMsgWithState{
		RequestID: echo.playback.timelineRequestID,
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
	if final.playback.currentTrack != "Artist - New Track (New Album)" {
		t.Fatalf("expected new track to apply, got %q", final.playback.currentTrack)
	}
	if final.playback.currentTrackKey != "new-key" {
		t.Fatalf("expected current track key to update, got %q", final.playback.currentTrackKey)
	}
}

func TestPlaybackTriggeredDoesNotBlockRestartNearBeginning(t *testing.T) {
	initTestLogger(t)

	m := model{
		playback: playbackState{
			timelineRequestID: 8,
			currentTrack:      "Artist - Track (Album)",
			currentTrackKey:   "same-key",
			durationMs:        200000,
			positionMs:        900,
			lastUpdate:        time.Now(),
		},
	}

	updatedModel, cmd := m.Update(playbackTriggeredMsg{success: true})
	if cmd != nil {
		t.Fatalf("expected nil command when no player is selected, got non-nil")
	}

	updated := updatedModel.(model)
	restartModel, restartCmd := updated.Update(trackMsgWithState{
		RequestID: updated.playback.timelineRequestID,
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
	if restarted.playback.currentTrack != "Artist - Track (Album)" {
		t.Fatalf("expected restart update to apply immediately, got %q", restarted.playback.currentTrack)
	}
	if restarted.playback.positionMs != 0 {
		t.Fatalf("expected position to update to 0, got %d", restarted.playback.positionMs)
	}
}

func TestPlaybackTriggeredInvalidatesPendingTrackPlaybackResponses(t *testing.T) {
	initTestLogger(t)

	m := model{
		trackPlaybackReqID: 4,
		playback: playbackState{
			timelineRequestID: 12,
			currentTrack:      "Artist - Old Track (Album)",
			currentTrackKey:   "old-key",
			positionMs:        10000,
			lastUpdate:        time.Now(),
		},
	}

	updatedModel, cmd := m.Update(playbackTriggeredMsg{success: true})
	if cmd != nil {
		t.Fatalf("expected nil command when no player is selected, got non-nil")
	}

	updated := updatedModel.(model)
	if updated.trackPlaybackReqID != 5 {
		t.Fatalf("expected trackPlaybackReqID to increment, got %d", updated.trackPlaybackReqID)
	}
	if updated.lastCommand != "Playback Started" {
		t.Fatalf("expected playback-triggered command marker, got %q", updated.lastCommand)
	}

	staleModel, staleCmd := updated.Update(trackPlaybackMsg{
		success:   true,
		requestID: 4,
		ratingKey: "stale-track-key",
	})
	if staleCmd != nil {
		t.Fatalf("expected nil command for stale track playback response, got non-nil")
	}

	stale := staleModel.(model)
	if stale.lastCommand != "Playback Started" {
		t.Fatalf("expected stale response to be ignored, got lastCommand=%q", stale.lastCommand)
	}
	if stale.playback.pendingTrackKey != "" {
		t.Fatalf("expected pendingTrackKey to stay empty, got %q", stale.playback.pendingTrackKey)
	}
}

func TestTrackPlaybackMsgFailureClearsPendingNowPlayingState(t *testing.T) {
	initTestLogger(t)

	m := model{
		trackPlaybackReqID: 9,
		playback: playbackState{
			currentTrack:      "Loading track...",
			currentTrackKey:   "old-key",
			isPlaying:         true,
			durationMs:        123000,
			positionMs:        45000,
			lastUpdate:        time.Now(),
			suppressTimeline:  true,
			pendingTrackKey:   "new-key",
			pendingTrackUntil: time.Now().Add(8 * time.Second),
			ignoreTrackKey:    "ignore-key",
			ignoreTrackPosMs:  30000,
			ignoreTrackUntil:  time.Now().Add(4 * time.Second),
		},
	}

	updatedModel, cmd := m.Update(trackPlaybackMsg{
		success:   false,
		err:       errors.New("no server selected"),
		requestID: 9,
	})
	if cmd != nil {
		t.Fatalf("expected nil command for playback failure, got non-nil")
	}

	updated := updatedModel.(model)
	if updated.playback.currentTrack != "" {
		t.Fatalf("expected current track to clear, got %q", updated.playback.currentTrack)
	}
	if updated.playback.currentTrackKey != "" {
		t.Fatalf("expected current track key to clear, got %q", updated.playback.currentTrackKey)
	}
	if updated.playback.isPlaying {
		t.Fatalf("expected playing state to clear")
	}
	if updated.playback.durationMs != 0 {
		t.Fatalf("expected duration to reset, got %d", updated.playback.durationMs)
	}
	if updated.playback.positionMs != 0 {
		t.Fatalf("expected position to reset, got %d", updated.playback.positionMs)
	}
	if !updated.playback.lastUpdate.IsZero() {
		t.Fatalf("expected lastUpdate to reset, got %v", updated.playback.lastUpdate)
	}
	if updated.playback.pendingTrackKey != "" {
		t.Fatalf("expected pending track key to clear, got %q", updated.playback.pendingTrackKey)
	}
	if updated.playback.suppressTimeline {
		t.Fatalf("expected timeline suppression to clear")
	}
}
