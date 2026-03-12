package ui

import (
	"testing"
	"time"
)

func TestPreviousTrackResetsPlayheadAndInvalidatesInFlightPolls(t *testing.T) {
	m := model{
		positionMs:        45000,
		lastUpdate:        time.Now().Add(-5 * time.Second),
		suppressTimeline:  true,
		timelineRequestID: 3,
	}

	cmd := m.previousTrack()
	if cmd != nil {
		t.Fatalf("expected nil command when no player is selected, got non-nil")
	}

	if m.positionMs != 0 {
		t.Fatalf("expected position to reset to 0, got %d", m.positionMs)
	}
	if m.timelineRequestID != 4 {
		t.Fatalf("expected timelineRequestID to increment to 4, got %d", m.timelineRequestID)
	}
	if m.suppressTimeline {
		t.Fatalf("expected suppressTimeline to be false")
	}
	if m.lastUpdate.IsZero() {
		t.Fatalf("expected lastUpdate to be set")
	}
	if m.lastCommand != "Previous" {
		t.Fatalf("expected lastCommand to be Previous, got %q", m.lastCommand)
	}
	if m.status != "No Plexamp instance selected" {
		t.Fatalf("expected no-player status, got %q", m.status)
	}
}
