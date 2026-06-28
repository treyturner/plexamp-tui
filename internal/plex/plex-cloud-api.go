package plex

import (
	"encoding/xml"
	"fmt"
	"net/http"
)

//curl "https://plex.tv/api/resources?includeHttps=1&includeRelay=1&X-Plex-Token=<token>"

const (
	plexCloudBaseURL = "https://plex.tv"
)

type PlexDeviceInfo struct {
	Name                 string           `xml:"name,attr"`
	Product              string           `xml:"product,attr"`
	ProductVersion       string           `xml:"productVersion,attr"`
	Platform             string           `xml:"platform,attr"`
	PlatformVersion      string           `xml:"platformVersion,attr"`
	Device               string           `xml:"device,attr"`
	ClientIdentifier     string           `xml:"clientIdentifier,attr"`
	CreatedAt            string           `xml:"createdAt,attr"`
	LastSeenAt           string           `xml:"lastSeenAt,attr"`
	Provides             string           `xml:"provides,attr"`
	Owned                string           `xml:"owned,attr"`
	SearchEnabled        string           `xml:"searchEnabled,attr"`
	PublicAddress        string           `xml:"publicAddress,attr"`
	PublicAddressMatches string           `xml:"publicAddressMatches,attr"`
	Presence             string           `xml:"presence,attr"`
	Connections          []PlexConnection `xml:"Connection"`
}

type PlexConnection struct {
	Protocol string `xml:"protocol,attr"`
	Address  string `xml:"address,attr"`
	Port     string `xml:"port,attr"`
	URI      string `xml:"uri,attr"`
	Local    string `xml:"local,attr"`
	Relay    string `xml:"relay,attr"`
}

type PlexDeviceContainer struct {
	XMLName xml.Name         `xml:"MediaContainer"`
	Size    int              `xml:"size,attr"`
	Devices []PlexDeviceInfo `xml:"Device"`
}

type PlexConnectionSelection struct {
	Name             string `xml:"name,attr"`
	ClientIdentifier string `xml:"clientIdentifier,attr"`
	Scheme           string `xml:"scheme,attr"`
	Address          string `xml:"address,attr"`
	Local            string `xml:"local,attr"`
	Port             string `xml:"port,attr"`
	URI              string `xml:"uri,attr"`
}

func (p *PlexClient) GetPlexServerInformation() ([]PlexConnectionSelection, error) {
	token := p.GetPlexToken()
	urlStr := fmt.Sprintf("%s/api/resources?includeHttps=1&includeRelay=1&X-Plex-Token=%s", plexCloudBaseURL, token)

	body, err := p.get(urlStr, http.StatusOK, http.StatusNoContent)
	if err != nil {
		p.logger.Debug("Request error: %v", err)
		return nil, fmt.Errorf("failed to connect to %s: %w", plexCloudBaseURL, err)
	}
	servers, err := parsePlexConnections(body, "server")
	if err != nil {
		return nil, err
	}

	return servers, nil
}

func (p *PlexClient) GetPlexPlayers() ([]PlexConnectionSelection, error) {
	token := p.GetPlexToken()
	urlStr := fmt.Sprintf("%s/api/resources?includeHttps=1&includeRelay=1&X-Plex-Token=%s", plexCloudBaseURL, token)

	body, err := p.get(urlStr, http.StatusOK, http.StatusNoContent)
	if err != nil {
		p.logger.Debug("Request error: %v", err)
		return nil, fmt.Errorf("failed to connect to %s: %w", plexCloudBaseURL, err)
	}
	servers, err := parsePlexConnections(body, "player")
	if err != nil {
		return nil, err
	}

	return servers, nil
}
