package ui

type panelMode string

const (
	panelModeServers            panelMode = "servers"
	panelModePlayback           panelMode = "playback"
	panelModeEdit               panelMode = "edit"
	panelModePlexServers        panelMode = "plex-servers"
	panelModePlexPlayers        panelMode = "plex-players"
	panelModePlexLibraries      panelMode = "plex-libraries"
	panelModePlexArtists        panelMode = "plex-artists"
	panelModePlexArtistAlbums   panelMode = "plex-artist-albums"
	panelModePlexAlbums         panelMode = "plex-albums"
	panelModePlexAlbumTracks    panelMode = "plex-album-tracks"
	panelModePlexPlaylists      panelMode = "plex-playlists"
	panelModePlexPlaylistTracks panelMode = "plex-playlist-tracks"
)

type editMode string

const (
	editModeServer   editMode = "server"
	editModePlayback editMode = "playback"
)

type favoriteType string

const (
	favoriteTypeArtist   favoriteType = "artist"
	favoriteTypeAlbum    favoriteType = "album"
	favoriteTypePlaylist favoriteType = "playlist"
	favoriteTypeTrack    favoriteType = "track"
)

type trackBrowseContext string

const (
	trackBrowseContextAlbum    trackBrowseContext = "album"
	trackBrowseContextPlaylist trackBrowseContext = "playlist"
)
