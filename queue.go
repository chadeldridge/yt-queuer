package ytqueuer

import "fmt"

var ErrQueueEmpty = fmt.Errorf("Queue is empty")

type Details struct {
	VideoId      string `json:"videoId"`
	StartSeconds int    `json:"startSeconds"`
}

type Queue struct {
	Videos []Details
}

func NewQueue() Queue {
	return Queue{Videos: make([]Details, 0)}
}

func (q *Queue) Add(vid string, start int) {
	q.Videos = append(q.Videos, Details{VideoId: vid, StartSeconds: start})
}

func (q *Queue) PlayNext(vid string, start int) {
	if len(q.Videos) == 0 {
		q.Add(vid, start)
		return
	}

	n := make([]Details, 0, len(q.Videos)+1)
	n = append(n, Details{VideoId: vid, StartSeconds: start})
	q.Videos = append(n, q.Videos...)
}

func (q *Queue) GetNext() (Details, error) {
	if len(q.Videos) == 0 {
		return Details{}, ErrQueueEmpty
	}

	d := q.Videos[0]
	q.Videos = q.Videos[1:]
	return d, nil
}

func (q Queue) PeekNext() (Details, error) {
	if len(q.Videos) == 0 {
		return Details{}, ErrQueueEmpty
	}

	return q.Videos[0], nil
}
