package ytqueuer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

func (s *HTTPServer) AddRoutes() {
	// Setup middleware.
	mwLogger := LoggerMiddleware(s.Logger)
	// Handle static assets.
	s.Mux.Handle("/", mwLogger(http.StripPrefix("/", http.FileServer(http.Dir("public")))))

	// Handle queue operations.
	s.Mux.Handle("GET /queue/", mwLogger(s.QueueHandler()))
	s.Mux.Handle("GET /queue/add/{video_id}", mwLogger(s.AddHandler()))
	s.Mux.Handle("GET /queue/playnext/{video_id}", mwLogger(s.PlayNextHandler()))
	s.Mux.Handle("GET /queue/next", mwLogger(s.NextHandler(false)))
	s.Mux.Handle("GET /queue/peek", mwLogger(s.NextHandler(true)))
	s.Mux.Handle("GET /queue/remove/{video_id}", mwLogger(s.RemoveHandler()))
	s.Mux.Handle("GET /queue/clear", mwLogger(s.ClearHandler()))
}

// func renderJSON[T any](w http.ResponseWriter, r *http.Request, status int, obj T) error {
func RenderJSON[T any](w http.ResponseWriter, status int, obj T) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(obj); err != nil {
		return fmt.Errorf("encoder: %w", err)
	}

	return nil
}

func RenderHTML(w http.ResponseWriter, status int, s string) error {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(status)
	_, err := w.Write([]byte(s))
	return err
}

func (s *HTTPServer) QueueHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(s.Queue.Videos) == 0 {
			http.Error(w, "", http.StatusNoContent)
			return
		}

		if err := RenderJSON(w, http.StatusOK, s.Queue.Videos); err != nil {
			s.Logger.Printf("error rendering json: %v\n", err)
			http.Error(w, fmt.Sprintf("error rendering json: %v", err), http.StatusInternalServerError)
		}
	})
}

func (s *HTTPServer) AddHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the video ID from the path.
		id := r.PathValue("video_id")

		// Get the start time, in seconds, from the query string.
		start := 0
		startRaw := r.URL.Query().Get("start")
		// Skip this step if startRaw is an empty string.
		if startRaw != "" {
			// If startRaw can be successfully converted to an integer, set start to that value.
			if si, err := strconv.Atoi(startRaw); err == nil {
				start = si
			}
		}

		// Add the video to the queue. If there is an error, send a 400 Bad Request response.
		if err := s.Queue.Add(id, start); err != nil {
			s.Logger.Printf("error adding video to queue: %v\n", err)
			http.Error(w, fmt.Sprintf("error adding video to queue: %v", err), http.StatusBadRequest)
			return
		}

		// Send a JSON response.
		msg := struct {
			Message string    `json:"message"`
			Queue   []Details `json:"queue"`
		}{
			Message: "video added to queue",
			Queue:   s.Queue.Videos,
		}

		if err := RenderJSON(w, http.StatusOK, msg); err != nil {
			s.Logger.Printf("error rendering json: %v\n", err)
			http.Error(w, fmt.Sprintf("error rendering json: %v", err), http.StatusInternalServerError)
		}
	})
}

func (s *HTTPServer) PlayNextHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("video_id")

		start := 0
		// Get the start time, in seconds, from the query string.
		startRaw := r.URL.Query().Get("start")
		if si, err := strconv.Atoi(startRaw); err == nil {
			start = si
		}

		// Add the video to beginning of the queue. If there is an error,
		// send a 400 Bad Request response.
		if err := s.Queue.PlayNext(id, start); err != nil {
			s.Logger.Printf("error adding video to queue: %v\n", err)
			http.Error(w, fmt.Sprintf("error adding video to queue: %v", err), http.StatusBadRequest)
			return
		}

		// Return a JSON response.
		msg := struct {
			Message string    `json:"message"`
			Queue   []Details `json:"queue"`
		}{
			Message: "video added to start of queue",
			Queue:   s.Queue.Videos,
		}

		if err := RenderJSON(w, http.StatusOK, msg); err != nil {
			s.Logger.Printf("error rendering json: %v\n", err)
			http.Error(w, fmt.Sprintf("error rendering json: %v", err), http.StatusInternalServerError)
		}
	})
}

func (s *HTTPServer) NextHandler(peek bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var d Details
		var err error
		if peek {
			d, err = s.Queue.PeekNext()
		} else {
			d, err = s.Queue.GetNext()
		}

		if err != nil {
			if err == ErrQueueEmpty || err == ErrQueueNoNext {
				http.Error(w, "", http.StatusNoContent)
				return
			}

			s.Logger.Printf("error getting next video: %v\n", err)
			http.Error(w, fmt.Sprintf("error rendering html: %v", err), http.StatusInternalServerError)
			return
		}

		if err := RenderJSON(w, http.StatusOK, d); err != nil {
			s.Logger.Printf("error rendering json: %v\n", err)
			http.Error(w, fmt.Sprintf("error rendering json: %v", err), http.StatusInternalServerError)
		}
	})
}

func (s *HTTPServer) RemoveHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := s.Queue.Remove(r.PathValue("video_id"))
		if err != nil {
			s.Logger.Printf("error removing video from queue: %v\n", err)
			http.Error(w, fmt.Sprintf("error removing video from queue: %v", err), http.StatusBadRequest)
			return
		}

		http.Error(w, "", http.StatusNoContent)
	})
}

func (s *HTTPServer) ClearHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Queue.Clear()
		http.Error(w, "", http.StatusNoContent)
	})
}
