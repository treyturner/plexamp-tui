package ui

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"plexamp-tui/internal/config"
)

func TestTogglePlaybackUpdatesOptimisticallyBeforeCommandResponses(t *testing.T) {
	m := model{
		selected: "127.0.0.1",
		playback: playbackState{
			isPlaying: true,
		},
	}

	if cmd := m.togglePlayback(); cmd == nil {
		t.Fatalf("expected pause command")
	}
	if m.playback.isPlaying {
		t.Fatalf("expected first optimistic toggle to pause playback")
	}

	if cmd := m.togglePlayback(); cmd == nil {
		t.Fatalf("expected play command")
	}
	if !m.playback.isPlaying {
		t.Fatalf("expected second optimistic toggle to resume playback")
	}
	if m.playback.pendingToggleCommandID != 2 {
		t.Fatalf("expected latest pending toggle command id to be 2, got %d", m.playback.pendingToggleCommandID)
	}
}

func TestStaleToggleResultDoesNotOverwriteOptimisticState(t *testing.T) {
	m := model{
		selected: "127.0.0.1",
		playback: playbackState{
			isPlaying:              true,
			toggleCommandID:        2,
			pendingToggleCommandID: 2,
		},
	}

	updatedModel, cmd := m.Update(playbackControlMsg{
		action:          playbackControlToggle,
		path:            "playback/pause",
		selected:        "127.0.0.1",
		isPlaying:       false,
		toggleCommandID: 1,
		poll:            true,
	})
	if cmd != nil {
		t.Fatalf("expected stale toggle result to be ignored without polling")
	}

	updated := updatedModel.(model)
	if !updated.playback.isPlaying {
		t.Fatalf("expected stale toggle result to leave playback playing")
	}
	if updated.playback.pendingToggleCommandID != 2 {
		t.Fatalf("expected pending toggle command id to remain 2, got %d", updated.playback.pendingToggleCommandID)
	}
}

func TestTogglePlaybackFailureRollsBackToBurstBaseState(t *testing.T) {
	m := model{
		selected: "127.0.0.1",
		playback: playbackState{
			isPlaying: true,
		},
	}

	if cmd := m.togglePlayback(); cmd == nil {
		t.Fatalf("expected pause command")
	}
	if cmd := m.togglePlayback(); cmd == nil {
		t.Fatalf("expected play command")
	}

	updatedModel, cmd := m.Update(playbackControlMsg{
		action:          playbackControlToggle,
		selected:        "127.0.0.1",
		isPlaying:       true,
		toggleCommandID: 2,
		err:             errors.New("request failed"),
	})
	if cmd != nil {
		t.Fatalf("expected nil command after failed toggle")
	}

	updated := updatedModel.(model)
	if !updated.playback.isPlaying {
		t.Fatalf("expected failed toggle burst to roll back to playing")
	}
	if updated.playback.pendingToggleCommandID != 0 {
		t.Fatalf("expected pending toggle command id to clear, got %d", updated.playback.pendingToggleCommandID)
	}
}

func TestTogglePlaybackFailureRollsBackToLastAcknowledgedState(t *testing.T) {
	m := model{
		selected: "127.0.0.1",
		playback: playbackState{
			isPlaying: true,
		},
	}

	if cmd := m.togglePlayback(); cmd == nil {
		t.Fatalf("expected first toggle command")
	}
	if cmd := m.togglePlayback(); cmd == nil {
		t.Fatalf("expected second toggle command")
	}

	updatedModel, cmd := m.Update(playbackControlMsg{
		action:          playbackControlToggle,
		selected:        "127.0.0.1",
		isPlaying:       false,
		toggleCommandID: 1,
	})
	if cmd != nil {
		t.Fatalf("expected stale toggle success to return no command")
	}

	updated := updatedModel.(model)
	if !updated.playback.isPlaying {
		t.Fatalf("expected pending optimistic toggle state to remain playing")
	}
	if updated.playback.acknowledgedPlaying {
		t.Fatalf("expected first successful toggle command to be acknowledged as paused")
	}
	if updated.playback.acknowledgedToggleID != 1 {
		t.Fatalf("expected acknowledged toggle command id to be 1, got %d", updated.playback.acknowledgedToggleID)
	}

	updatedModel, cmd = updated.Update(playbackControlMsg{
		action:          playbackControlToggle,
		selected:        "127.0.0.1",
		isPlaying:       true,
		toggleCommandID: 2,
		err:             errors.New("request failed"),
	})
	if cmd != nil {
		t.Fatalf("expected nil command after failed latest toggle")
	}

	updated = updatedModel.(model)
	if updated.playback.isPlaying {
		t.Fatalf("expected failed latest toggle to roll back to acknowledged paused state")
	}
	if updated.playback.pendingToggleCommandID != 0 {
		t.Fatalf("expected pending toggle command id to clear, got %d", updated.playback.pendingToggleCommandID)
	}
}

func TestTimelineDoesNotOverwritePendingOptimisticToggle(t *testing.T) {
	m := model{
		playback: playbackState{
			isPlaying:              false,
			pendingToggleCommandID: 1,
		},
	}

	applied := m.playback.applyTimeline(trackMsgWithState{
		TrackText: "Song",
		TrackKey:  "track-1",
		IsPlaying: true,
	}, time.Now(), func(string, ...interface{}) {})
	if !applied {
		t.Fatalf("expected timeline to apply")
	}
	if m.playback.isPlaying {
		t.Fatalf("expected pending optimistic pause to remain paused")
	}
}

func TestToggleShuffleUpdatesOptimisticallyBeforeCommandResponses(t *testing.T) {
	m := model{
		selected: "127.0.0.1",
		shuffle:  false,
	}

	if cmd := m.toggleShuffle(); cmd == nil {
		t.Fatalf("expected shuffle command")
	}
	if !m.shuffle {
		t.Fatalf("expected first optimistic shuffle toggle to turn shuffle on")
	}

	if cmd := m.toggleShuffle(); cmd == nil {
		t.Fatalf("expected shuffle command")
	}
	if m.shuffle {
		t.Fatalf("expected second optimistic shuffle toggle to turn shuffle off")
	}
	if m.pendingShuffleCommandID != 2 {
		t.Fatalf("expected latest pending shuffle command id to be 2, got %d", m.pendingShuffleCommandID)
	}
}

func TestStaleShuffleResultDoesNotOverwriteOptimisticState(t *testing.T) {
	m := model{
		selected:                "127.0.0.1",
		shuffle:                 false,
		shuffleCommandID:        2,
		pendingShuffleCommandID: 2,
	}

	updatedModel, cmd := m.Update(playbackControlMsg{
		action:           playbackControlShuffle,
		path:             "playback/shuffle/on",
		selected:         "127.0.0.1",
		shuffle:          true,
		shuffleCommandID: 1,
	})
	if cmd != nil {
		t.Fatalf("expected stale shuffle result to be ignored without command")
	}

	updated := updatedModel.(model)
	if updated.shuffle {
		t.Fatalf("expected stale shuffle result to leave shuffle off")
	}
	if updated.pendingShuffleCommandID != 2 {
		t.Fatalf("expected pending shuffle command id to remain 2, got %d", updated.pendingShuffleCommandID)
	}
	if !updated.acknowledgedShuffle {
		t.Fatalf("expected stale successful shuffle result to update acknowledged shuffle state")
	}
	if updated.acknowledgedShuffleID != 1 {
		t.Fatalf("expected acknowledged shuffle command id to be 1, got %d", updated.acknowledgedShuffleID)
	}
}

func TestShuffleFailureRollsBackToLastAcknowledgedState(t *testing.T) {
	m := model{
		selected: "127.0.0.1",
		shuffle:  false,
	}

	if cmd := m.toggleShuffle(); cmd == nil {
		t.Fatalf("expected first shuffle command")
	}
	if cmd := m.toggleShuffle(); cmd == nil {
		t.Fatalf("expected second shuffle command")
	}

	updatedModel, cmd := m.Update(playbackControlMsg{
		action:           playbackControlShuffle,
		selected:         "127.0.0.1",
		shuffle:          true,
		shuffleCommandID: 1,
	})
	if cmd != nil {
		t.Fatalf("expected stale shuffle success to return no command")
	}

	updated := updatedModel.(model)
	if updated.shuffle {
		t.Fatalf("expected pending optimistic shuffle state to remain off")
	}
	if !updated.acknowledgedShuffle {
		t.Fatalf("expected first successful shuffle command to be acknowledged as on")
	}
	if updated.pendingShuffleCommandID != 2 {
		t.Fatalf("expected latest pending shuffle command id to remain 2, got %d", updated.pendingShuffleCommandID)
	}

	updatedModel, cmd = updated.Update(playbackControlMsg{
		action:           playbackControlShuffle,
		selected:         "127.0.0.1",
		shuffle:          false,
		shuffleCommandID: 2,
		err:              errors.New("request failed"),
	})
	if cmd != nil {
		t.Fatalf("expected nil command after failed latest shuffle")
	}

	updated = updatedModel.(model)
	if !updated.shuffle {
		t.Fatalf("expected failed latest shuffle to roll back to acknowledged on state")
	}
	if updated.pendingShuffleCommandID != 0 {
		t.Fatalf("expected pending shuffle command id to clear, got %d", updated.pendingShuffleCommandID)
	}
}

func TestShuffleFailureRollsBackToBurstBaseState(t *testing.T) {
	m := model{
		selected: "127.0.0.1",
		shuffle:  false,
	}

	if cmd := m.toggleShuffle(); cmd == nil {
		t.Fatalf("expected shuffle command")
	}

	updatedModel, cmd := m.Update(playbackControlMsg{
		action:           playbackControlShuffle,
		selected:         "127.0.0.1",
		shuffle:          true,
		shuffleCommandID: 1,
		err:              errors.New("request failed"),
	})
	if cmd != nil {
		t.Fatalf("expected nil command after failed shuffle")
	}

	updated := updatedModel.(model)
	if updated.shuffle {
		t.Fatalf("expected failed shuffle toggle to roll back to off")
	}
	if updated.pendingShuffleCommandID != 0 {
		t.Fatalf("expected pending shuffle command id to clear, got %d", updated.pendingShuffleCommandID)
	}
}

func TestPlaybackControlReplyFromPreviousPlayerIsIgnored(t *testing.T) {
	m := model{
		selected: "player-b",
		playback: playbackState{
			volume:                 60,
			pendingVolumeCommandID: 2,
			positionMs:             45000,
			timelineRequestID:      3,
		},
	}

	updatedModel, cmd := m.Update(playbackControlMsg{
		action:          playbackControlVolume,
		path:            "playback/setParameters?volume=55&commandID=1&type=music",
		selected:        "player-a",
		volume:          55,
		volumeCommandID: 2,
		poll:            true,
	})
	if cmd != nil {
		t.Fatalf("expected old-player control reply to be ignored without polling")
	}

	updated := updatedModel.(model)
	if updated.playback.volume != 60 {
		t.Fatalf("expected old-player volume reply to leave volume at 60, got %d", updated.playback.volume)
	}
	if updated.playback.pendingVolumeCommandID != 2 {
		t.Fatalf("expected current pending volume command id to remain 2, got %d", updated.playback.pendingVolumeCommandID)
	}
	if updated.playback.positionMs != 45000 {
		t.Fatalf("expected old-player reply to leave position unchanged, got %d", updated.playback.positionMs)
	}
	if updated.playback.timelineRequestID != 3 {
		t.Fatalf("expected old-player reply to leave timelineRequestID at 3, got %d", updated.playback.timelineRequestID)
	}
}

func TestPreviousTrackReplyFromPreviousPlayerIsIgnored(t *testing.T) {
	m := model{
		selected: "player-b",
		playback: playbackState{
			positionMs:        45000,
			timelineRequestID: 3,
		},
	}

	updatedModel, cmd := m.Update(playbackControlMsg{
		action:   playbackControlPrevious,
		path:     "playback/skipPrevious",
		selected: "player-a",
		poll:     true,
	})
	if cmd != nil {
		t.Fatalf("expected old-player previous result to be ignored without polling")
	}

	updated := updatedModel.(model)
	if updated.playback.positionMs != 45000 {
		t.Fatalf("expected old-player previous result to leave position at 45000, got %d", updated.playback.positionMs)
	}
	if updated.playback.timelineRequestID != 3 {
		t.Fatalf("expected old-player previous result to leave timelineRequestID at 3, got %d", updated.playback.timelineRequestID)
	}
	if updated.lastCommand != "" {
		t.Fatalf("expected old-player previous result not to set lastCommand, got %q", updated.lastCommand)
	}
}

func TestPlayerSelectionClearsPendingPlaybackControls(t *testing.T) {
	cfgManager, err := config.NewManager(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatalf("failed to create config manager: %v", err)
	}

	m := model{
		selected: "player-a",
		config:   &config.Config{},
		deps: uiDeps{
			cfgManager: cfgManager,
		},
		playback: playbackState{
			pendingToggleCommandID: 7,
			pendingVolumeCommandID: 8,
			acknowledgedToggleID:   10,
			acknowledgedVolumeID:   11,
			timelineRequestID:      12,
		},
		playbackRequestID:       5,
		ackPlaybackRequestID:    4,
		trackPlaybackReqID:      6,
		ackTrackPlaybackReqID:   3,
		ackTrackPlaybackKey:     "old-track",
		pendingShuffleCommandID: 9,
		acknowledgedShuffleID:   13,
	}

	updatedModel, cmd := m.Update(playerSelectMsg{
		success: true,
		player: playerItem{
			title:   "Player B",
			address: "player-b",
		},
	})
	if cmd != nil {
		t.Fatalf("expected player selection to return no command")
	}

	updated := updatedModel.(model)
	if updated.selected != "player-b" {
		t.Fatalf("expected selected player-b, got %q", updated.selected)
	}
	if updated.playback.pendingToggleCommandID != 0 {
		t.Fatalf("expected pending toggle command id to clear, got %d", updated.playback.pendingToggleCommandID)
	}
	if updated.playback.pendingVolumeCommandID != 0 {
		t.Fatalf("expected pending volume command id to clear, got %d", updated.playback.pendingVolumeCommandID)
	}
	if updated.pendingShuffleCommandID != 0 {
		t.Fatalf("expected pending shuffle command id to clear, got %d", updated.pendingShuffleCommandID)
	}
	if updated.acknowledgedShuffleID != 0 {
		t.Fatalf("expected acknowledged shuffle command id to clear, got %d", updated.acknowledgedShuffleID)
	}
	if updated.playback.acknowledgedToggleID != 0 {
		t.Fatalf("expected acknowledged toggle command id to clear, got %d", updated.playback.acknowledgedToggleID)
	}
	if updated.playback.acknowledgedVolumeID != 0 {
		t.Fatalf("expected acknowledged volume command id to clear, got %d", updated.playback.acknowledgedVolumeID)
	}
	if updated.playbackRequestID != 6 {
		t.Fatalf("expected playbackRequestID to increment to 6, got %d", updated.playbackRequestID)
	}
	if updated.ackPlaybackRequestID != 0 {
		t.Fatalf("expected acknowledged playback request id to clear, got %d", updated.ackPlaybackRequestID)
	}
	if updated.trackPlaybackReqID != 7 {
		t.Fatalf("expected trackPlaybackReqID to increment to 7, got %d", updated.trackPlaybackReqID)
	}
	if updated.ackTrackPlaybackReqID != 0 {
		t.Fatalf("expected acknowledged track playback request id to clear, got %d", updated.ackTrackPlaybackReqID)
	}
	if updated.ackTrackPlaybackKey != "" {
		t.Fatalf("expected acknowledged track playback key to clear, got %q", updated.ackTrackPlaybackKey)
	}
	if updated.playback.timelineRequestID != 13 {
		t.Fatalf("expected timelineRequestID to increment to 13, got %d", updated.playback.timelineRequestID)
	}
}

func TestAdjustVolumeUpdatesOptimisticallyBeforeCommandResponses(t *testing.T) {
	m := model{
		selected: "127.0.0.1",
		playback: playbackState{
			volume: 50,
		},
	}

	if cmd := m.adjustVolume(5); cmd == nil {
		t.Fatalf("expected volume command")
	}
	if m.playback.volume != 55 {
		t.Fatalf("expected first optimistic volume to be 55, got %d", m.playback.volume)
	}

	if cmd := m.adjustVolume(5); cmd == nil {
		t.Fatalf("expected volume command")
	}
	if m.playback.volume != 60 {
		t.Fatalf("expected second optimistic volume to be 60, got %d", m.playback.volume)
	}

	if cmd := m.adjustVolume(5); cmd == nil {
		t.Fatalf("expected volume command")
	}
	if m.playback.volume != 65 {
		t.Fatalf("expected third optimistic volume to be 65, got %d", m.playback.volume)
	}
	if m.playback.pendingVolumeCommandID != 3 {
		t.Fatalf("expected latest pending volume command id to be 3, got %d", m.playback.pendingVolumeCommandID)
	}
}

func TestVolumeFailureRollsBackToBurstBaseVolume(t *testing.T) {
	m := model{
		selected: "127.0.0.1",
		playback: playbackState{
			volume: 50,
		},
	}

	if cmd := m.adjustVolume(5); cmd == nil {
		t.Fatalf("expected first volume command")
	}
	if cmd := m.adjustVolume(5); cmd == nil {
		t.Fatalf("expected second volume command")
	}

	updatedModel, cmd := m.Update(playbackControlMsg{
		action:          playbackControlVolume,
		selected:        "127.0.0.1",
		volume:          60,
		volumeCommandID: 2,
		err:             errors.New("request failed"),
	})
	if cmd != nil {
		t.Fatalf("expected nil command after failed volume change")
	}

	updated := updatedModel.(model)
	if updated.playback.volume != 50 {
		t.Fatalf("expected failed volume burst to roll back to 50, got %d", updated.playback.volume)
	}
	if updated.playback.pendingVolumeCommandID != 0 {
		t.Fatalf("expected pending volume command id to clear, got %d", updated.playback.pendingVolumeCommandID)
	}
}

func TestVolumeFailureRollsBackToLastAcknowledgedVolume(t *testing.T) {
	m := model{
		selected: "127.0.0.1",
		playback: playbackState{
			volume: 50,
		},
	}

	if cmd := m.adjustVolume(5); cmd == nil {
		t.Fatalf("expected first volume command")
	}
	if cmd := m.adjustVolume(5); cmd == nil {
		t.Fatalf("expected second volume command")
	}

	updatedModel, cmd := m.Update(playbackControlMsg{
		action:          playbackControlVolume,
		selected:        "127.0.0.1",
		volume:          55,
		volumeCommandID: 1,
	})
	if cmd != nil {
		t.Fatalf("expected stale volume success to return no command")
	}

	updated := updatedModel.(model)
	if updated.playback.volume != 60 {
		t.Fatalf("expected pending optimistic volume to remain 60, got %d", updated.playback.volume)
	}
	if updated.playback.acknowledgedVolume != 55 {
		t.Fatalf("expected first successful volume command to acknowledge 55, got %d", updated.playback.acknowledgedVolume)
	}
	if updated.playback.acknowledgedVolumeID != 1 {
		t.Fatalf("expected acknowledged volume command id to be 1, got %d", updated.playback.acknowledgedVolumeID)
	}

	updatedModel, cmd = updated.Update(playbackControlMsg{
		action:          playbackControlVolume,
		selected:        "127.0.0.1",
		volume:          60,
		volumeCommandID: 2,
		err:             errors.New("request failed"),
	})
	if cmd != nil {
		t.Fatalf("expected nil command after failed latest volume")
	}

	updated = updatedModel.(model)
	if updated.playback.volume != 55 {
		t.Fatalf("expected failed latest volume to roll back to acknowledged 55, got %d", updated.playback.volume)
	}
	if updated.playback.pendingVolumeCommandID != 0 {
		t.Fatalf("expected pending volume command id to clear, got %d", updated.playback.pendingVolumeCommandID)
	}
}

func TestStaleVolumeResultDoesNotOverwriteOptimisticVolume(t *testing.T) {
	m := model{
		selected: "127.0.0.1",
		playback: playbackState{
			volume:                 60,
			volumeCommandID:        2,
			pendingVolumeCommandID: 2,
		},
	}

	updatedModel, cmd := m.Update(playbackControlMsg{
		action:          playbackControlVolume,
		path:            "playback/setParameters?volume=55&commandID=1&type=music",
		selected:        "127.0.0.1",
		volume:          55,
		volumeCommandID: 1,
		poll:            true,
	})
	if cmd != nil {
		t.Fatalf("expected stale volume result to be ignored without polling")
	}

	updated := updatedModel.(model)
	if updated.playback.volume != 60 {
		t.Fatalf("expected stale volume result to leave volume at 60, got %d", updated.playback.volume)
	}
	if updated.playback.pendingVolumeCommandID != 2 {
		t.Fatalf("expected pending volume command id to remain 2, got %d", updated.playback.pendingVolumeCommandID)
	}
}

func TestTimelineDoesNotOverwritePendingOptimisticVolume(t *testing.T) {
	m := model{
		playback: playbackState{
			volume:                 65,
			pendingVolumeCommandID: 3,
		},
	}

	applied := m.playback.applyTimeline(trackMsgWithState{
		TrackText: "Song",
		TrackKey:  "track-1",
		Volume:    50,
	}, time.Now(), func(string, ...interface{}) {})
	if !applied {
		t.Fatalf("expected timeline to apply")
	}
	if m.playback.volume != 65 {
		t.Fatalf("expected pending optimistic volume to remain 65, got %d", m.playback.volume)
	}
}

func TestTimelineFromPreviousPlayerIsIgnored(t *testing.T) {
	m := model{
		selected: "player-b",
		playback: playbackState{
			currentTrack:      "Existing Track",
			positionMs:        45000,
			timelineRequestID: 3,
		},
	}

	updatedModel, cmd := m.Update(trackMsgWithState{
		Selected:  "player-a",
		RequestID: 3,
		TrackText: "Wrong Player Track",
		TrackKey:  "wrong-track",
		IsPlaying: true,
		Duration:  100000,
		Position:  1000,
		Volume:    70,
	})
	if cmd != nil {
		t.Fatalf("expected nil command for old-player timeline update")
	}

	updated := updatedModel.(model)
	if updated.playback.currentTrack != "Existing Track" {
		t.Fatalf("expected old-player timeline to leave current track unchanged, got %q", updated.playback.currentTrack)
	}
	if updated.playback.positionMs != 45000 {
		t.Fatalf("expected old-player timeline to leave position unchanged, got %d", updated.playback.positionMs)
	}
}

func TestPlaybackTriggeredFromPreviousPlayerIsIgnored(t *testing.T) {
	m := model{
		selected:           "player-b",
		playbackRequestID:  4,
		trackPlaybackReqID: 8,
		playback: playbackState{
			currentTrack:      "Existing Track",
			timelineRequestID: 3,
		},
	}

	updatedModel, cmd := m.Update(playbackTriggeredMsg{
		success:   true,
		selected:  "player-a",
		requestID: 4,
	})
	if cmd != nil {
		t.Fatalf("expected nil command for old-player playback trigger")
	}

	updated := updatedModel.(model)
	if updated.playback.currentTrack != "Existing Track" {
		t.Fatalf("expected old-player playback trigger to leave current track unchanged, got %q", updated.playback.currentTrack)
	}
	if updated.trackPlaybackReqID != 8 {
		t.Fatalf("expected old-player playback trigger to leave trackPlaybackReqID at 8, got %d", updated.trackPlaybackReqID)
	}
	if updated.lastCommand != "" {
		t.Fatalf("expected old-player playback trigger not to set lastCommand, got %q", updated.lastCommand)
	}
}

func TestPlaybackTriggeredFailureUsesEarlierSuccessFromCurrentBurst(t *testing.T) {
	m := model{
		selected:          "player-a",
		playbackRequestID: 2,
		playback: playbackState{
			timelineRequestID: 3,
		},
	}

	updatedModel, cmd := m.Update(playbackTriggeredMsg{
		success:   true,
		selected:  "player-a",
		requestID: 1,
	})
	if cmd != nil {
		t.Fatalf("expected stale playback success to be acknowledged without command")
	}

	updated := updatedModel.(model)
	if updated.ackPlaybackRequestID != 1 {
		t.Fatalf("expected acknowledged playback request id to be 1, got %d", updated.ackPlaybackRequestID)
	}

	updatedModel, cmd = updated.Update(playbackTriggeredMsg{
		success:   false,
		selected:  "player-a",
		requestID: 2,
		err:       errors.New("request failed"),
	})
	if cmd == nil {
		t.Fatalf("expected timeline refresh after latest playback request failed with earlier success acknowledged")
	}

	updated = updatedModel.(model)
	if updated.lastCommand != "Playback Started" {
		t.Fatalf("expected playback started marker, got %q", updated.lastCommand)
	}
	if updated.status != "Playback triggered successfully" {
		t.Fatalf("expected playback success status, got %q", updated.status)
	}
}

func TestPlaybackRequestClearsEarlierAcknowledgedSuccess(t *testing.T) {
	m := model{
		selected:             "player-a",
		playbackRequestID:    1,
		ackPlaybackRequestID: 1,
		playback: playbackState{
			timelineRequestID: 3,
		},
	}

	requestID := m.nextPlaybackRequestID()
	if requestID != 2 {
		t.Fatalf("expected new playback request id 2, got %d", requestID)
	}
	if m.ackPlaybackRequestID != 0 {
		t.Fatalf("expected new playback request to clear acknowledged id, got %d", m.ackPlaybackRequestID)
	}

	updatedModel, cmd := m.Update(playbackTriggeredMsg{
		success:   false,
		selected:  "player-a",
		requestID: requestID,
		err:       errors.New("request failed"),
	})
	if cmd != nil {
		t.Fatalf("expected failed new playback request to return no command")
	}

	updated := updatedModel.(model)
	if updated.lastCommand != "Playback Failed" {
		t.Fatalf("expected playback failed marker, got %q", updated.lastCommand)
	}
	if updated.status != "Playback error: request failed" {
		t.Fatalf("expected playback failure status, got %q", updated.status)
	}
	if updated.playback.timelineRequestID != 3 {
		t.Fatalf("expected playback failure not to start timeline refresh, got request id %d", updated.playback.timelineRequestID)
	}
}

func TestPreviousTrackCommandRequiresSelectedPlayer(t *testing.T) {
	m := model{
		playback: playbackState{
			positionMs:        45000,
			lastUpdate:        time.Now().Add(-5 * time.Second),
			suppressTimeline:  true,
			timelineRequestID: 3,
		},
	}

	cmd := m.previousTrack()
	if cmd != nil {
		updatedModel, updatedCmd := m.Update(cmd())
		if updatedCmd != nil {
			t.Fatalf("expected nil command after no-player result, got non-nil")
		}
		updated := updatedModel.(model)
		if updated.status != "No Plexamp instance selected" {
			t.Fatalf("expected no-player status, got %q", updated.status)
		}
		if updated.playback.positionMs != 45000 {
			t.Fatalf("expected position to remain unchanged, got %d", updated.playback.positionMs)
		}
		if updated.playback.timelineRequestID != 3 {
			t.Fatalf("expected timelineRequestID to remain unchanged, got %d", updated.playback.timelineRequestID)
		}
		return
	}

	t.Fatalf("expected command for previous track")
}

func TestPreviousTrackResultResetsPlayheadAndInvalidatesInFlightPolls(t *testing.T) {
	m := model{
		selected: "127.0.0.1",
		playback: playbackState{
			positionMs:        45000,
			lastUpdate:        time.Now().Add(-5 * time.Second),
			suppressTimeline:  true,
			timelineRequestID: 3,
		},
	}

	updatedModel, cmd := m.Update(playbackControlMsg{
		action:   playbackControlPrevious,
		path:     "playback/skipPrevious",
		selected: "127.0.0.1",
		poll:     true,
	})
	if cmd == nil {
		t.Fatalf("expected poll command after previous-track result")
	}

	updated := updatedModel.(model)
	if m.playback.positionMs != 45000 {
		t.Fatalf("expected original position to remain unchanged, got %d", m.playback.positionMs)
	}
	if updated.playback.positionMs != 0 {
		t.Fatalf("expected position to reset to 0, got %d", updated.playback.positionMs)
	}
	if updated.playback.timelineRequestID != 4 {
		t.Fatalf("expected timelineRequestID to increment to 4, got %d", updated.playback.timelineRequestID)
	}
	if updated.playback.suppressTimeline {
		t.Fatalf("expected suppressTimeline to be false")
	}
	if updated.playback.lastUpdate.IsZero() {
		t.Fatalf("expected lastUpdate to be set")
	}
	if updated.lastCommand != "Previous" {
		t.Fatalf("expected lastCommand to be Previous, got %q", updated.lastCommand)
	}
	if updated.status != "[127.0.0.1] Sent playback/skipPrevious" {
		t.Fatalf("expected previous status, got %q", updated.status)
	}
}
