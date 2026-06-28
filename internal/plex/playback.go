package plex

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

const (
	plexListenBaseURL = "https://listen.plex.tv"
	plexURIPrefix     = "server://%s/com.plexapp.plugins.library/library/metadata/%s"
)

type PlaybackURLBuilder struct {
	serverID string
}

func NewPlaybackURLBuilder(serverID string) *PlaybackURLBuilder {
	return &PlaybackURLBuilder{
		serverID: serverID,
	}
}

func (b *PlaybackURLBuilder) BuildPlaylistURL(metadataID string) string {
	uri := fmt.Sprintf(plexURIPrefix, b.serverID, metadataID)
	return fmt.Sprintf("%s/player/playback/createPlayQueue?source=%s&uri=%s&playlistID=%s&type=audio",
		plexListenBaseURL,
		url.QueryEscape(b.serverID),
		url.QueryEscape(uri),
		metadataID,
	)
}

func (b *PlaybackURLBuilder) BuildPlayQueueURL(metadataID string) string {
	uri := fmt.Sprintf(plexURIPrefix, b.serverID, metadataID)
	return fmt.Sprintf("%s/player/playback/createPlayQueue?uri=%s", plexListenBaseURL, url.QueryEscape(uri))
}

func (b *PlaybackURLBuilder) BuildArtistRadioURL(metadataID, stationUUID string) string {
	uri := fmt.Sprintf(plexURIPrefix+"/station/%s", b.serverID, metadataID, stationUUID)
	return fmt.Sprintf("%s/player/playback/playMedia?type=10&type=audio&uri=%s",
		plexListenBaseURL, url.QueryEscape(uri))
}

func ApplyShuffle(urlStr string, shuffle bool) (string, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return urlStr, err
	}

	q := u.Query()
	if shuffle {
		q.Set("shuffle", "1")
	} else {
		q.Del("shuffle")
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}

func (p *PlexClient) SendPlaybackURL(serverIP, fullURL string, shuffle bool) error {
	modifiedURL := fullURL
	if shuffleURL, err := ApplyShuffle(fullURL, shuffle); err == nil {
		modifiedURL = shuffleURL
	}

	localURL := strings.Replace(modifiedURL, "https://listen.plex.tv", fmt.Sprintf("http://%s:32500", serverIP), 1)
	localURL = strings.Replace(localURL, "http://listen.plex.tv", fmt.Sprintf("http://%s:32500", serverIP), 1)

	p.debug("Sending playback URL: %s", localURL)

	req, err := http.NewRequest(http.MethodGet, localURL, nil)
	if err != nil {
		return err
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		p.debug("Request error: %v", err)
		return fmt.Errorf("failed to connect to %s: %w", serverIP, err)
	}
	defer func() { _ = resp.Body.Close() }()

	p.debug("Response status: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}

func (p *PlexClient) PlayMetadata(serverIP, serverID, metadataID string, shuffle bool) error {
	builder := NewPlaybackURLBuilder(serverID)
	playbackURL := builder.BuildPlayQueueURL(metadataID)
	return p.SendPlaybackURL(serverIP, playbackURL, shuffle)
}

func (p *PlexClient) PlayArtistRadio(serverIP, serverID, metadataID string, shuffle bool) error {
	stationUUID := uuid.New().String()
	builder := NewPlaybackURLBuilder(serverID)
	playbackURL := builder.BuildArtistRadioURL(metadataID, stationUUID)
	return p.SendPlaybackURL(serverIP, playbackURL, shuffle)
}

func (p *PlexClient) PlayPlaylist(serverIP, serverID, metadataID string, shuffle bool) error {
	builder := NewPlaybackURLBuilder(serverID)
	playbackURL := builder.BuildPlaylistURL(metadataID)
	return p.SendPlaybackURL(serverIP, playbackURL, shuffle)
}
