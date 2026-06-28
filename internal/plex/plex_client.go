package plex

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"plexamp-tui/internal/logger"
)

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type PlexClient struct {
	logger     *logger.Logger
	httpClient httpDoer
}

func NewPlexClient(logger *logger.Logger) *PlexClient {
	return NewPlexClientWithHTTPClient(logger, &http.Client{Timeout: 10 * time.Second})
}

func NewPlexClientWithHTTPClient(logger *logger.Logger, httpClient httpDoer) *PlexClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &PlexClient{
		logger:     logger,
		httpClient: httpClient,
	}
}

func (p *PlexClient) get(urlStr string, okStatuses ...int) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if !statusAllowed(resp.StatusCode, okStatuses) {
		if p.logger != nil {
			p.logger.Debug("Server returned status %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	return body, nil
}

func statusAllowed(status int, okStatuses []int) bool {
	if len(okStatuses) == 0 {
		okStatuses = []int{http.StatusOK}
	}
	for _, okStatus := range okStatuses {
		if status == okStatus {
			return true
		}
	}
	return false
}
