package plex

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"plexamp-tui/internal/config"
	"strings"
)

// =====================
// Plex Library Types
// =====================

type PlexLibraryContainer struct {
	XMLName   xml.Name      `xml:"MediaContainer"`
	Size      int           `xml:"size,attr"`
	Libraries []PlexLibrary `xml:"Directory"`
}

type PlexLibrary struct {
	Key   string `xml:"key,attr"`
	Title string `xml:"title,attr"`
	Type  string `xml:"type,attr"`
}

// PlexDirectory represents a generic directory item from Plex
type PlexDirectory struct {
	XMLName     xml.Name `xml:"Directory"`
	RatingKey   string   `xml:"ratingKey,attr"`
	Title       string   `xml:"title,attr"`
	Type        string   `xml:"type,attr"`
	ParentTitle string   `xml:"parentTitle,attr"` // For albums
	Year        string   `xml:"year,attr"`
}

// PlexArtist represents an artist from the Plex library
type PlexArtist struct {
	RatingKey string `xml:"ratingKey,attr"`
	Title     string `xml:"title,attr"`
	Type      string `xml:"type,attr"`
}

// PlexAlbum represents an album from the Plex library
type PlexAlbum struct {
	RatingKey   string `xml:"ratingKey,attr"`
	Title       string `xml:"title,attr"`
	ParentTitle string `xml:"parentTitle,attr"` // Artist name
	Year        string `xml:"year,attr"`
	Type        string `xml:"type,attr"`
}

// PlexPlaylist represents a playlist from the Plex library
type PlexPlaylist struct {
	RatingKey string `xml:"ratingKey,attr"`
	Title     string `xml:"title,attr"`
	Type      string `xml:"playlistType,attr"`
}

// PlexTrack represents a track from the Plex library
type PlexTrack struct {
	RatingKey        string `xml:"ratingKey,attr"`
	Title            string `xml:"title,attr"`
	ParentTitle      string `xml:"parentTitle,attr"`
	GrandparentTitle string `xml:"grandparentTitle,attr"`
	ParentIndex      int    `xml:"parentIndex,attr"`
	Index            int    `xml:"index,attr"`
	Duration         int    `xml:"duration,attr"`
}

// PlexMediaContainer is the root element for Plex API responses
type PlexMediaContainer struct {
	XMLName     xml.Name        `xml:"MediaContainer"`
	Size        int             `xml:"size,attr"`
	Directories []PlexDirectory `xml:"Directory"`
}

type PlexTrackContainer struct {
	XMLName xml.Name    `xml:"MediaContainer"`
	Size    int         `xml:"size,attr"`
	Tracks  []PlexTrack `xml:"Track"`
}

type PlexPlaylistContainer struct {
	XMLName   xml.Name       `xml:"MediaContainer"`
	Size      int            `xml:"size,attr"`
	Playlists []PlexPlaylist `xml:"Playlist"`
}

// =====================
// Library Fetching
// =====================

func buildPlexURL(serverAddr, path string) string {
	addrLower := strings.ToLower(serverAddr)
	if strings.HasPrefix(addrLower, "http://") || strings.HasPrefix(addrLower, "https://") {
		return fmt.Sprintf("%s%s", strings.TrimRight(serverAddr, "/"), path)
	}
	return fmt.Sprintf("http://%s%s", serverAddr, path)
}

// FetchArtists retrieves all artists from the Plex library
func (p *PlexClient) FetchArtists(serverAddr, libraryID, token string) ([]PlexArtist, error) {
	urlStr := buildPlexURL(serverAddr,
		fmt.Sprintf("/library/sections/%s/all?type=8&X-Plex-Token=%s",
			libraryID,
			url.QueryEscape(token),
		),
	)

	p.logger.Debug("Fetching artists from: %s", urlStr)

	body, err := p.get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch artists: %w", err)
	}
	artists, err := parseArtists(body)
	if err != nil {
		p.logger.Debug("Failed to parse artists: %v", err)
		return nil, err
	}

	p.logger.Debug("Fetched %d artists", len(artists))

	return artists, nil
}

// FetchAlbums retrieves all albums from the Plex library
func (p *PlexClient) FetchAlbums(serverAddr, libraryID, token string) ([]PlexAlbum, error) {
	urlStr := buildPlexURL(serverAddr,
		fmt.Sprintf("/library/sections/%s/all?type=9&X-Plex-Token=%s",
			libraryID,
			url.QueryEscape(token),
		),
	)

	p.logger.Debug("Fetching albums from: %s", urlStr)

	body, err := p.get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch albums: %w", err)
	}
	albums, err := parseAlbums(body)
	if err != nil {
		p.logger.Debug("Failed to parse albums: %v", err)
		return nil, err
	}

	p.logger.Debug("Fetched %d albums", len(albums))

	return albums, nil
}

// FetchArtistAlbums retrieves albums for a specific artist
func (p *PlexClient) FetchArtistAlbums(serverAddr, artistRatingKey, token string) ([]PlexAlbum, error) {
	urlStr := buildPlexURL(serverAddr,
		fmt.Sprintf("/library/metadata/%s/children?X-Plex-Token=%s",
			artistRatingKey,
			url.QueryEscape(token),
		),
	)

	p.logger.Debug("Fetching albums for artist %s from: %s", artistRatingKey, urlStr)

	body, err := p.get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch artist albums: %w", err)
	}
	albums, err := parseArtistAlbums(body)
	if err != nil {
		return nil, err
	}

	return albums, nil
}

func (p *PlexClient) FetchPlaylists(serverAddr, token string) ([]PlexPlaylist, error) {
	urlStr := buildPlexURL(serverAddr,
		fmt.Sprintf("/playlists?X-Plex-Token=%s", url.QueryEscape(token)),
	)

	p.logger.Debug("Fetching playlists from: %s", urlStr)

	body, err := p.get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch playlists: %w", err)
	}
	playlists, err := parsePlaylists(body)
	if err != nil {
		return nil, err
	}

	return playlists, nil
}

func (p *PlexClient) FetchAlbumTracks(serverAddr, albumRatingKey, token string) ([]PlexTrack, error) {
	urlStr := buildPlexURL(serverAddr,
		fmt.Sprintf("/library/metadata/%s/children?X-Plex-Token=%s",
			albumRatingKey,
			url.QueryEscape(token),
		),
	)

	p.logger.Debug("Fetching tracks for album %s from: %s", albumRatingKey, urlStr)

	body, err := p.get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch album tracks: %w", err)
	}
	tracks, err := parseAlbumTracks(body)
	if err != nil {
		return nil, err
	}

	return tracks, nil
}

func (p *PlexClient) FetchPlaylistTracks(serverAddr, playlistRatingKey, token string) ([]PlexTrack, error) {
	urlStr := buildPlexURL(serverAddr,
		fmt.Sprintf("/playlists/%s/items?X-Plex-Token=%s",
			playlistRatingKey,
			url.QueryEscape(token),
		),
	)

	p.logger.Debug("Fetching tracks for playlist %s from: %s", playlistRatingKey, urlStr)

	body, err := p.get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch playlist tracks: %w", err)
	}
	tracks, err := parseTracks(body)
	if err != nil {
		return nil, err
	}

	return tracks, nil
}

func (p *PlexClient) FetchLibrary(serverAddr string) ([]config.PlexLibrary, error) {
	token := p.GetPlexToken()
	urlStr := buildPlexURL(serverAddr,
		fmt.Sprintf("/library/sections?X-Plex-Token=%s", url.QueryEscape(token)),
	)

	p.logger.Debug("Fetching library from: %s", urlStr)

	body, err := p.get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch library: %w", err)
	}
	libraries, err := parseArtistLibraries(body)
	if err != nil {
		return nil, err
	}

	p.logger.Debug("Fetched %d artist libraries", len(libraries))

	return libraries, nil
}
