package application

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
)

var (
	vidFormat = regexp.MustCompile(`^[a-zA-Z0-9_-]{10}[a-zA-Z0-9]$`)
	ytURL     = "https://www.youtube.com/oembed?format=json&url=https://www.youtube.com/watch?v=%s"

	ErrPlaylistEmpty = fmt.Errorf("queue is empty")
	ErrEndOfPlaylist = fmt.Errorf("no more videos in queue")
)

type VideoDetails struct {
	VideoID      string `json:"video_id"`
	Title        string `json:"title"`
	AuthorName   string `json:"author_name"`
	ThumbnailURL string `json:"thumbnail_url"`
	StartSeconds int    `json:"start_seconds"`
}

func NewDetails(vid string, start int) (VideoDetails, error) {
	d := VideoDetails{VideoID: vid, StartSeconds: start}

	client := http.Client{
		Timeout: 2 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf(ytURL, vid), nil)
	if err != nil {
		return d, fmt.Errorf("NewDetails - error creating request: %v", err)
	}

	req.Header.Set("User-Agent", "yt-queuer")
	res, err := client.Do(req)
	if err != nil {
		return d, fmt.Errorf("NewDetails - error making request: %v", err)
	}

	if res.StatusCode != http.StatusOK || res.Body == nil {
		return d, fmt.Errorf("NewDetails - youtube returned unexpected response or empty body: %d", res.StatusCode)
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return d, fmt.Errorf("NewDetails - error reading response body: %v", err)
	}

	err = json.Unmarshal(body, &d)
	if err != nil {
		return d, fmt.Errorf("NewDetails - error unmarshalling response body: %v", err)
	}

	return d, nil
}

type Playlist []VideoDetails

func NewPlaylist() Playlist {
	return make([]VideoDetails, 0)
}

func (pls Playlists) Save(db *SqliteDB, pbc PlaybackClient) error {
	if pbc.ID == "" {
		return fmt.Errorf("PlaybackClient.ID is empty")
	}

	if pbc.Name == "" {
		return fmt.Errorf("PlaybackClient.Name is empty")
	}

	if err := db.PlaylistUpdate(pbc, pls[pbc]); err != nil {
		return fmt.Errorf("Playlist.Save: %w", err)
	}

	return nil
}

type Playlists map[PlaybackClient]Playlist

func NewPlaylists() Playlists {
	return make(map[PlaybackClient]Playlist, 0)
}

func UnmarshalPlaylists(data []byte) (Playlist, error) {
	var pl Playlist
	err := json.Unmarshal(data, &pl)
	if err != nil {
		return pl, fmt.Errorf("UnmarshalPlaylists: %w", err)
	}

	return pl, nil
}

func validateVideoID(vid string) error {
	if vid == "" {
		return fmt.Errorf("video id is empty")
	}

	if !vidFormat.MatchString(vid) {
		return fmt.Errorf("invalid video id: %s", vid)
	}

	return nil
}

func (pl Playlist) isDuplicate(vid string) bool {
	for _, d := range pl {
		if d.VideoID == vid {
			return true
		}
	}

	return false
}

func (pls Playlists) LoadFromDB(db *SqliteDB) error {
	all, err := db.PlaylistGetAll()
	if err != nil {
		return fmt.Errorf("Playlists.LoadFromDB: %w", err)
	}

	if len(all) == 0 {
		return nil
	}

	for pbc, pl := range all {
		pls[pbc] = pl
	}

	return nil
}

func (pls Playlists) GetPBCs() []PlaybackClient {
	if len(pls) == 0 {
		return []PlaybackClient{}
	}

	pbcs := make([]PlaybackClient, 0, len(pls))
	for pbc := range pls {
		pbcs = append(pbcs, pbc)
	}

	return pbcs
}

func (pls Playlists) Add(pbc PlaybackClient, vid string, start int) error {
	if err := validateVideoID(vid); err != nil {
		return err
	}

	if pls[pbc].isDuplicate(vid) {
		return fmt.Errorf("video already in queue: %s", vid)
	}

	d, err := NewDetails(vid, start)
	if err != nil {
		return err
	}

	pls[pbc] = append(pls[pbc], d)

	return nil
}

func (pls Playlists) PlayNext(pbc PlaybackClient, vid string, start int) error {
	if err := validateVideoID(vid); err != nil {
		return err
	}

	if len(pls[pbc]) == 0 {
		pls.Add(pbc, vid, start)
		return nil
	}

	if pls[pbc].isDuplicate(vid) {
		return fmt.Errorf("video already in queue: %s", vid)
	}

	d, err := NewDetails(vid, start)
	if err != nil {
		return err
	}

	n := make([]VideoDetails, 0, len(pls[pbc])+1)
	n = append(n, d)
	pls[pbc] = append(n, pls[pbc]...)

	return nil
}

func (pls Playlists) GetNext(pbc PlaybackClient) (VideoDetails, error) {
	if len(pls) == 0 {
		return VideoDetails{}, ErrPlaylistEmpty
	}

	if len(pls[pbc]) == 0 {
		return VideoDetails{}, ErrPlaylistEmpty
	}

	d := pls[pbc][0]
	return d, nil
}

func (pls Playlists) PeekNext(pbc PlaybackClient) (VideoDetails, error) {
	if len(pls) < 2 {
		return VideoDetails{}, ErrEndOfPlaylist
	}

	return pls[pbc][1], nil
}

func (pls Playlists) Remove(pbc PlaybackClient, vid string) error {
	if len(pls[pbc]) == 0 {
		return ErrPlaylistEmpty
	}

	for i, d := range pls[pbc] {
		if d.VideoID == vid {
			pls[pbc] = append(pls[pbc][:i], pls[pbc][i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("video not found in queue: %s", vid)
}

func (pls Playlists) Clear(pbc PlaybackClient) {
	pls[pbc] = make([]VideoDetails, 0)
}
