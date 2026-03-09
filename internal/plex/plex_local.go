package plex

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"plexamp-tui/internal/config"
	"sort"
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

	p.logger.Debug(fmt.Sprintf("Fetching artists from: %s", urlStr))

	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch artists: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		p.logger.Debug(fmt.Sprintf("Server returned status %d", resp.StatusCode))
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		p.logger.Debug(fmt.Sprintf("Failed to read response: %v", err))
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var container PlexMediaContainer
	if err := xml.Unmarshal(body, &container); err != nil {
		p.logger.Debug(fmt.Sprintf("Failed to parse XML: %v", err))
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	var artists []PlexArtist
	for _, dir := range container.Directories {
		if dir.Type == "artist" {
			artists = append(artists, PlexArtist{
				RatingKey: dir.RatingKey,
				Title:     dir.Title,
				Type:      dir.Type,
			})
		}
	}

	p.logger.Debug(fmt.Sprintf("Fetched %d artists", len(artists)))

	// Sort artists alphabetically by title
	sort.Slice(artists, func(i, j int) bool {
		return artists[i].Title < artists[j].Title
	})

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

	p.logger.Debug(fmt.Sprintf("Fetching albums from: %s", urlStr))

	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch albums: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		p.logger.Debug(fmt.Sprintf("Server returned status %d", resp.StatusCode))
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		p.logger.Debug(fmt.Sprintf("Failed to read response: %v", err))
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var container PlexMediaContainer
	if err := xml.Unmarshal(body, &container); err != nil {
		p.logger.Debug(fmt.Sprintf("Failed to parse XML: %v", err))
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	var albums []PlexAlbum
	for _, dir := range container.Directories {
		if dir.Type == "album" {
			albums = append(albums, PlexAlbum{
				RatingKey:   dir.RatingKey,
				Title:       dir.Title,
				ParentTitle: dir.ParentTitle,
				Year:        dir.Year,
				Type:        dir.Type,
			})
		}
	}

	p.logger.Debug(fmt.Sprintf("Fetched %d albums", len(albums)))

	// Sort albums alphabetically by title
	sort.Slice(albums, func(i, j int) bool {
		return albums[i].ParentTitle < albums[j].ParentTitle
	})

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

	p.logger.Debug(fmt.Sprintf("Fetching albums for artist %s from: %s", artistRatingKey, urlStr))

	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch artist albums: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var container PlexMediaContainer
	if err := xml.Unmarshal(body, &container); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	var albums []PlexAlbum
	for _, dir := range container.Directories {
		if dir.Type == "album" {
			albums = append(albums, PlexAlbum{
				RatingKey:   dir.RatingKey,
				Title:       dir.Title,
				ParentTitle: dir.ParentTitle,
				Year:        dir.Year,
				Type:        dir.Type,
			})
		}
	}

	sort.Slice(albums, func(i, j int) bool {
		return albums[i].Title < albums[j].Title
	})

	return albums, nil
}

func (p *PlexClient) FetchPlaylists(serverAddr, token string) ([]PlexPlaylist, error) {
	urlStr := buildPlexURL(serverAddr,
		fmt.Sprintf("/playlists?X-Plex-Token=%s", url.QueryEscape(token)),
	)

	p.logger.Debug(fmt.Sprintf("Fetching playlists from: %s", urlStr))

	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch playlists: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var container PlexPlaylistContainer
	if err := xml.Unmarshal(body, &container); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	return container.Playlists, nil
}

func (p *PlexClient) FetchAlbumTracks(serverAddr, albumRatingKey, token string) ([]PlexTrack, error) {
	urlStr := buildPlexURL(serverAddr,
		fmt.Sprintf("/library/metadata/%s/children?X-Plex-Token=%s",
			albumRatingKey,
			url.QueryEscape(token),
		),
	)

	p.logger.Debug(fmt.Sprintf("Fetching tracks for album %s from: %s", albumRatingKey, urlStr))

	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch album tracks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var container PlexTrackContainer
	if err := xml.Unmarshal(body, &container); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	sort.SliceStable(container.Tracks, func(i, j int) bool {
		if container.Tracks[i].ParentIndex == container.Tracks[j].ParentIndex {
			return container.Tracks[i].Index < container.Tracks[j].Index
		}
		return container.Tracks[i].ParentIndex < container.Tracks[j].ParentIndex
	})

	return container.Tracks, nil
}

func (p *PlexClient) FetchPlaylistTracks(serverAddr, playlistRatingKey, token string) ([]PlexTrack, error) {
	urlStr := buildPlexURL(serverAddr,
		fmt.Sprintf("/playlists/%s/items?X-Plex-Token=%s",
			playlistRatingKey,
			url.QueryEscape(token),
		),
	)

	p.logger.Debug(fmt.Sprintf("Fetching tracks for playlist %s from: %s", playlistRatingKey, urlStr))

	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch playlist tracks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var container PlexTrackContainer
	if err := xml.Unmarshal(body, &container); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	return container.Tracks, nil
}

func (p *PlexClient) FetchLibrary(serverAddr string) ([]config.PlexLibrary, error) {
	token := p.GetPlexToken()
	urlStr := buildPlexURL(serverAddr,
		fmt.Sprintf("/library/sections?X-Plex-Token=%s", url.QueryEscape(token)),
	)

	p.logger.Debug(fmt.Sprintf("Fetching library from: %s", urlStr))

	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch library: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var container PlexLibraryContainer
	if err := xml.Unmarshal(body, &container); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	p.logger.Debug(fmt.Sprintf("Fetched %d libraries", len(container.Libraries)))
	// filter just artist libraries
	var libraries []config.PlexLibrary
	for _, lib := range container.Libraries {
		if lib.Type == "artist" {
			libraries = append(libraries, config.PlexLibrary{
				Key:   lib.Key,
				Title: lib.Title,
				Type:  lib.Type,
			})
		}
	}

	p.logger.Debug(fmt.Sprintf("Fetched %d artist libraries", len(libraries)))

	return libraries, nil
}
