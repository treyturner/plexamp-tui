package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m model) playbackStatusView() string {
	info := lipgloss.NewStyle().Foreground(lipgloss.Color("#aaaaaa"))
	value := lipgloss.NewStyle().Foreground(lipgloss.Color("#00ffcc")).Bold(true)

	state := "⏸️ Paused"
	if m.playback.isPlaying {
		state = "▶️ Playing"
	}

	current := "None"
	if m.playback.currentTrack != "" {
		current = m.playback.currentTrack
	}

	elapsed := m.currentPosition()
	progress := formatTime(elapsed) + " / " + formatTime(m.playback.durationMs)
	bar := progressBar(elapsed, m.playback.durationMs, 20)

	body := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffaa00")).Render("Now Playing") + "\n\n"
	body += fmt.Sprintf(
		"%s: %s\n%s: %s\n%s: %s\n%s: %d\n",
		info.Render("State"), value.Render(state),
		info.Render("Track"), value.Render(current),
		info.Render("Progress"), value.Render(bar+"  "+progress),
		info.Render("Volume"), m.playback.volume,
	)

	return body
}

// =====================
// Playback Control Methods
// =====================

// togglePlayback toggles between play and pause
func (m *model) togglePlayback() tea.Cmd {
	previousPlaying := m.playback.isPlaying
	nextPlaying := !previousPlaying

	path := "playback/play"
	if previousPlaying {
		path = "playback/pause"
	}

	var toggleCommandID int
	if m.selected != "" {
		if m.playback.pendingToggleCommandID == 0 {
			m.playback.pendingToggleBasePlaying = previousPlaying
			m.playback.acknowledgedPlaying = previousPlaying
			m.playback.acknowledgedToggleID = 0
		}
		m.playback.isPlaying = nextPlaying
		m.playback.toggleCommandID++
		toggleCommandID = m.playback.toggleCommandID
		m.playback.pendingToggleCommandID = toggleCommandID
	}

	return playbackControlCmd(m.selected, path, playbackControlMsg{
		action:          playbackControlToggle,
		isPlaying:       nextPlaying,
		toggleCommandID: toggleCommandID,
		poll:            true,
	})
}

// nextTrack skips to the next track
func (m *model) nextTrack() tea.Cmd {
	return playbackControlCmd(m.selected, "playback/skipNext", playbackControlMsg{
		action: playbackControlNext,
		poll:   true,
	})
}

// previousTrack goes to the previous track
func (m *model) previousTrack() tea.Cmd {
	return playbackControlCmd(m.selected, "playback/skipPrevious", playbackControlMsg{
		action: playbackControlPrevious,
		poll:   true,
	})
}

// adjustVolume changes the volume by the specified delta (range: -100 to +100)
func (m *model) adjustVolume(delta int) tea.Cmd {
	newVol := m.playback.volume + delta
	if newVol < 0 {
		newVol = 0
	} else if newVol > 100 {
		newVol = 100
	}

	var volumeCommandID int
	if m.selected != "" {
		if m.playback.pendingVolumeCommandID == 0 {
			m.playback.pendingVolumeBase = m.playback.volume
			m.playback.acknowledgedVolume = m.playback.volume
			m.playback.acknowledgedVolumeID = 0
		}
		m.playback.volume = newVol
		m.playback.volumeCommandID++
		volumeCommandID = m.playback.volumeCommandID
		m.playback.pendingVolumeCommandID = volumeCommandID
	}

	path := fmt.Sprintf("playback/setParameters?volume=%d&commandID=1&type=music", newVol)
	return playbackControlCmd(m.selected, path, playbackControlMsg{
		action:          playbackControlVolume,
		volume:          newVol,
		volumeCommandID: volumeCommandID,
		poll:            true,
	})
}

// toggleShuffle toggles shuffle mode
func (m *model) toggleShuffle() tea.Cmd {
	nextShuffle := !m.shuffle
	path := "playback/shuffle/off"
	if nextShuffle {
		path = "playback/shuffle/on"
	}

	var shuffleCommandID int
	if m.selected != "" {
		if m.pendingShuffleCommandID == 0 {
			m.pendingShuffleBase = m.shuffle
			m.acknowledgedShuffle = m.shuffle
			m.acknowledgedShuffleID = 0
		}
		m.shuffle = nextShuffle
		m.shuffleCommandID++
		shuffleCommandID = m.shuffleCommandID
		m.pendingShuffleCommandID = shuffleCommandID
	}

	return playbackControlCmd(m.selected, path, playbackControlMsg{
		action:           playbackControlShuffle,
		shuffle:          nextShuffle,
		shuffleCommandID: shuffleCommandID,
	})
}

// will use the config to cycle through the library options, it will check the current selected library and increment to the next one, if it is the last one it will go back to the first one
func (m *model) cycleLibrary() tea.Cmd {
	currentLibraryKey := m.config.PlexLibraryID

	for i := range m.config.PlexLibraries {
		if m.config.PlexLibraries[i].Key == currentLibraryKey {
			if i == len(m.config.PlexLibraries)-1 {
				m.config.PlexLibraryID = m.config.PlexLibraries[0].Key
				m.config.PlexLibraryName = m.config.PlexLibraries[0].Title
			} else {
				m.config.PlexLibraryID = m.config.PlexLibraries[i+1].Key
				m.config.PlexLibraryName = m.config.PlexLibraries[i+1].Title
			}
			if err := m.deps.cfgManager.Save(m.config); err != nil {
				m.status = "Error saving library selection: " + err.Error()
				m.lastCommand = "Library Selection Save Failed"
				return nil
			}
			// Return a command that will refresh the current panel
			return m.refreshCurrentPanel()
		}
	}
	return nil
}
