package motify

import (
	"github.com/zmb3/spotify"
	zs "github.com/zmb3/spotify"
)

// Client handles accessing the Spotify APIs
type Client struct {
	zsc zs.Client
}

func newClient(client zs.Client) Client {
	return Client{
		zsc: client,
	}
}

// TODO comment on these wrappers b/c they are public

func (c *Client) AddTracksToPlaylist(playlistID zs.ID, trackIDs ...zs.ID) (snapshotID string, err error) {
	return c.zsc.AddTracksToPlaylist(playlistID, trackIDs...)
}

func (c *Client) CreatePlaylistForUser(userID, playlistName, description string, public bool) (*zs.FullPlaylist, error) {
	return c.zsc.CreatePlaylistForUser(userID, playlistName, description, public)
}

func (c *Client) CurrentUser() (*zs.PrivateUser, error) {
	return c.zsc.CurrentUser()
}

func (c *Client) CurrentUsersAlbumsOpt(opt *zs.Options) (*zs.SavedAlbumPage, error) {
	return c.zsc.CurrentUsersAlbumsOpt(opt)
}

func (c *Client) CurrentUsersPlaylistsOpt(opt *zs.Options) (*zs.SimplePlaylistPage, error) {
	return c.zsc.CurrentUsersPlaylistsOpt(opt)
}

func (c *Client) CurrentUsersTracksOpt(opt *zs.Options) (*zs.SavedTrackPage, error) {
	return c.zsc.CurrentUsersTracksOpt(opt)
}

func (c *Client) GetAlbum(id zs.ID) (*zs.FullAlbum, error) {
	return c.zsc.GetAlbum(id)
}

func (c *Client) GetAlbumTracksOpt(id zs.ID, opt *zs.Options) (*spotify.SimpleTrackPage, error) {
	return c.zsc.GetAlbumTracksOpt(id, *opt.Limit, *opt.Offset)
}

func (c *Client) GetPlaylistOpt(playlistID zs.ID, fields string) (*zs.FullPlaylist, error) {
	return c.zsc.GetPlaylistOpt(playlistID, fields)
}

func (c *Client) GetPlaylistTracksOpt(playlistID zs.ID, opt *zs.Options, fields string) (*zs.PlaylistTrackPage, error) {
	return c.zsc.GetPlaylistTracksOpt(playlistID, opt, fields)
}

func (c *Client) UnfollowPlaylist(owner, playlist zs.ID) error {
	return c.zsc.UnfollowPlaylist(owner, playlist)
}
