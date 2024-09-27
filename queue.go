package ytqueuer

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
)

var (
	ErrQueueEmpty  = fmt.Errorf("queue is empty")
	ErrQueueNoNext = fmt.Errorf("no more videos in queue")
	vidFormat      = regexp.MustCompile(`^[a-zA-Z0-9_-]{10}[a-zA-Z0-9]$`)
	ytURL          = "https://www.youtube.com/oembed?format=json&url=https://www.youtube.com/watch?v=%s"
)

type Details struct {
	VideoId      string `json:"video_id"`
	Title        string `json:"title"`
	AuthorName   string `json:"author_name"`
	ThumbnailURL string `json:"thumbnail_url"`
	StartSeconds int    `json:"start_seconds"`
}

type Queue struct {
	Videos []Details
}

func NewQueue() Queue {
	return Queue{Videos: make([]Details, 0)}
}

func NewDetails(vid string, start int) (Details, error) {
	d := Details{VideoId: vid, StartSeconds: start}

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

func validateVideoID(vid string) error {
	if vid == "" {
		return fmt.Errorf("video id is empty")
	}

	if !vidFormat.MatchString(vid) {
		return fmt.Errorf("invalid video id: %s", vid)
	}

	return nil
}

func (q Queue) isDuplicate(vid string) bool {
	for _, d := range q.Videos {
		if d.VideoId == vid {
			return true
		}
	}

	return false
}

func (q *Queue) Add(vid string, start int) error {
	if err := validateVideoID(vid); err != nil {
		return err
	}

	if q.isDuplicate(vid) {
		return fmt.Errorf("video already in queue: %s", vid)
	}

	d, err := NewDetails(vid, start)
	if err != nil {
		return err
	}

	q.Videos = append(q.Videos, d)
	return nil
}

func (q *Queue) PlayNext(vid string, start int) error {
	if err := validateVideoID(vid); err != nil {
		return err
	}

	if len(q.Videos) == 0 {
		q.Add(vid, start)
		return nil
	}

	if q.isDuplicate(vid) {
		return fmt.Errorf("video already in queue: %s", vid)
	}

	d, err := NewDetails(vid, start)
	if err != nil {
		return err
	}

	n := make([]Details, 0, len(q.Videos)+1)
	n = append(n, d)
	q.Videos = append(n, q.Videos...)
	return nil
}

func (q *Queue) GetNext() (Details, error) {
	if len(q.Videos) == 0 {
		return Details{}, ErrQueueEmpty
	}

	d := q.Videos[0]
	return d, nil
}

func (q Queue) PeekNext() (Details, error) {
	if len(q.Videos) < 2 {
		return Details{}, ErrQueueNoNext
	}

	return q.Videos[1], nil
}

func (q *Queue) Remove(vid string) error {
	if len(q.Videos) == 0 {
		return ErrQueueEmpty
	}

	for i, d := range q.Videos {
		if d.VideoId == vid {
			q.Videos = append(q.Videos[:i], q.Videos[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("video not found in queue: %s", vid)
}

func (q *Queue) Clear() {
	q.Videos = make([]Details, 0)
}
