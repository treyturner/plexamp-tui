package ui

import "time"

type playbackState struct {
	isPlaying         bool
	currentTrack      string
	currentTrackKey   string
	volume            int
	durationMs        int
	positionMs        int
	lastUpdate        time.Time
	suppressTimeline  bool
	timelineRequestID int
	pendingTrackKey   string
	pendingTrackUntil time.Time
	ignoreTrackKey    string
	ignoreTrackPosMs  int
	ignoreTrackUntil  time.Time
}

func (p playbackState) currentPosition(now time.Time) int {
	pos := p.positionMs
	if p.isPlaying && !p.lastUpdate.IsZero() {
		pos += int(now.Sub(p.lastUpdate).Milliseconds())
	}
	if pos < 0 {
		pos = 0
	}
	if p.durationMs > 0 && pos > p.durationMs {
		pos = p.durationMs
	}
	return pos
}

func (p *playbackState) restartPrevious(now time.Time) {
	p.positionMs = 0
	p.lastUpdate = now
	p.suppressTimeline = false
	p.timelineRequestID++
}

func (p *playbackState) beginRefresh(pendingText, trackKey string, now time.Time) {
	if pendingText == "" {
		pendingText = "Loading..."
	}
	prevTrackKey := p.currentTrackKey
	prevPos := p.currentPosition(now)

	p.currentTrack = pendingText
	p.isPlaying = true
	p.durationMs = 0
	p.positionMs = 0
	p.lastUpdate = time.Time{}
	p.suppressTimeline = false
	p.pendingTrackKey = trackKey
	if trackKey != "" {
		p.pendingTrackUntil = now.Add(8 * time.Second)
	} else {
		p.pendingTrackUntil = time.Time{}
	}
	if prevTrackKey != "" && prevPos > 1000 {
		p.ignoreTrackKey = prevTrackKey
		p.ignoreTrackPosMs = prevPos
		p.ignoreTrackUntil = now.Add(4 * time.Second)
	} else {
		p.ignoreTrackKey = ""
		p.ignoreTrackPosMs = 0
		p.ignoreTrackUntil = time.Time{}
	}
	p.timelineRequestID++
}

func (p *playbackState) beginPending(pendingText, trackKey string, now time.Time) {
	if pendingText == "" {
		pendingText = "Loading..."
	}
	p.currentTrack = pendingText
	p.isPlaying = true
	p.durationMs = 0
	p.positionMs = 0
	p.lastUpdate = time.Time{}
	p.suppressTimeline = true
	p.pendingTrackKey = trackKey
	if trackKey != "" {
		p.pendingTrackUntil = now.Add(8 * time.Second)
	} else {
		p.pendingTrackUntil = time.Time{}
	}
	p.ignoreTrackKey = ""
	p.ignoreTrackPosMs = 0
	p.ignoreTrackUntil = time.Time{}
	p.timelineRequestID++
}

func (p *playbackState) clearAfterFailure() {
	p.currentTrack = ""
	p.currentTrackKey = ""
	p.isPlaying = false
	p.durationMs = 0
	p.positionMs = 0
	p.lastUpdate = time.Time{}
	p.suppressTimeline = false
	p.pendingTrackKey = ""
	p.pendingTrackUntil = time.Time{}
	p.ignoreTrackKey = ""
	p.ignoreTrackPosMs = 0
	p.ignoreTrackUntil = time.Time{}
}

func (p *playbackState) applyTimeline(msg trackMsgWithState, now time.Time, debug func(string, ...interface{})) bool {
	if msg.RequestID != p.timelineRequestID || p.suppressTimeline {
		return false
	}

	if p.ignoreTrackKey != "" {
		ignoreThreshold := p.ignoreTrackPosMs - 2000
		minThreshold := p.ignoreTrackPosMs / 2
		if ignoreThreshold < minThreshold {
			ignoreThreshold = minThreshold
		}
		if msg.TrackKey == p.ignoreTrackKey && now.Before(p.ignoreTrackUntil) && msg.Position >= ignoreThreshold {
			debug(
				"Ignoring stale transition timeline (trackKey=%s, pos=%d, threshold=%d)",
				msg.TrackKey, msg.Position, ignoreThreshold,
			)
			return false
		}
		if msg.TrackKey != p.ignoreTrackKey || !now.Before(p.ignoreTrackUntil) || msg.Position < ignoreThreshold {
			p.ignoreTrackKey = ""
			p.ignoreTrackPosMs = 0
			p.ignoreTrackUntil = time.Time{}
		}
	}

	if p.pendingTrackKey != "" {
		switch msg.TrackKey {
		case p.pendingTrackKey:
			p.pendingTrackKey = ""
			p.pendingTrackUntil = time.Time{}
		case "":
			if !p.pendingTrackUntil.IsZero() && now.Before(p.pendingTrackUntil) {
				debug(
					"Ignoring timeline with empty track key while waiting for pending track key=%s",
					p.pendingTrackKey,
				)
				return false
			}
			if !p.pendingTrackUntil.IsZero() {
				debug(
					"Pending track key timeout reached with empty track key; clearing filter (pending=%s)",
					p.pendingTrackKey,
				)
				p.pendingTrackKey = ""
				p.pendingTrackUntil = time.Time{}
			}
		default:
			if !p.pendingTrackUntil.IsZero() && now.Before(p.pendingTrackUntil) {
				debug(
					"Ignoring mismatched timeline track while waiting (got=%s, want=%s)",
					msg.TrackKey, p.pendingTrackKey,
				)
				return false
			}
			debug(
				"Pending track key timeout reached; clearing filter (got=%s, want=%s)",
				msg.TrackKey, p.pendingTrackKey,
			)
			p.pendingTrackKey = ""
			p.pendingTrackUntil = time.Time{}
		}
	}

	p.currentTrack = msg.TrackText
	p.currentTrackKey = msg.TrackKey
	p.isPlaying = msg.IsPlaying
	p.durationMs = msg.Duration
	p.positionMs = msg.Position
	p.volume = msg.Volume
	p.lastUpdate = now
	return true
}
