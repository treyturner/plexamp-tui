package plex

import (
	"encoding/xml"
	"fmt"
	"sort"
	"strings"

	"plexamp-tui/internal/config"
)

func parsePlexConnections(body []byte, provides string) ([]PlexConnectionSelection, error) {
	var container PlexDeviceContainer
	if err := xml.Unmarshal(body, &container); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	var connections []PlexConnectionSelection
	for _, device := range container.Devices {
		if !strings.Contains(device.Provides, provides) {
			continue
		}
		for _, connection := range device.Connections {
			connections = append(connections, PlexConnectionSelection{
				Name:             device.Name,
				ClientIdentifier: device.ClientIdentifier,
				Scheme:           connection.Protocol,
				Address:          connection.Address,
				Local:            connection.Local,
				Port:             connection.Port,
				URI:              connection.URI,
			})
		}
	}
	return connections, nil
}

func parseArtists(body []byte) ([]PlexArtist, error) {
	container, err := parseMediaContainer(body)
	if err != nil {
		return nil, err
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
	sort.Slice(artists, func(i, j int) bool {
		return artists[i].Title < artists[j].Title
	})
	return artists, nil
}

func parseAlbums(body []byte) ([]PlexAlbum, error) {
	container, err := parseMediaContainer(body)
	if err != nil {
		return nil, err
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
		return albums[i].ParentTitle < albums[j].ParentTitle
	})
	return albums, nil
}

func parseArtistAlbums(body []byte) ([]PlexAlbum, error) {
	albums, err := parseAlbums(body)
	if err != nil {
		return nil, err
	}
	sort.Slice(albums, func(i, j int) bool {
		return albums[i].Title < albums[j].Title
	})
	return albums, nil
}

func parsePlaylists(body []byte) ([]PlexPlaylist, error) {
	var container PlexPlaylistContainer
	if err := xml.Unmarshal(body, &container); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}
	return container.Playlists, nil
}

func parseAlbumTracks(body []byte) ([]PlexTrack, error) {
	tracks, err := parseTracks(body)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(tracks, func(i, j int) bool {
		if tracks[i].ParentIndex == tracks[j].ParentIndex {
			return tracks[i].Index < tracks[j].Index
		}
		return tracks[i].ParentIndex < tracks[j].ParentIndex
	})
	return tracks, nil
}

func parseTracks(body []byte) ([]PlexTrack, error) {
	var container PlexTrackContainer
	if err := xml.Unmarshal(body, &container); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}
	return container.Tracks, nil
}

func parseArtistLibraries(body []byte) ([]config.PlexLibrary, error) {
	var container PlexLibraryContainer
	if err := xml.Unmarshal(body, &container); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

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
	return libraries, nil
}

func parsePlexUser(body []byte) (*PlexUser, error) {
	var user PlexUser
	if err := xml.Unmarshal(body, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func parseMediaContainer(body []byte) (PlexMediaContainer, error) {
	var container PlexMediaContainer
	if err := xml.Unmarshal(body, &container); err != nil {
		return PlexMediaContainer{}, fmt.Errorf("failed to parse XML: %w", err)
	}
	return container, nil
}
