package ytqueuer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

func (s *HTTPServer) AddRoutes() {
	// Handle static assets
	s.Mux.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("public"))))

	// Handle queue operations
	// s.Mux.Handle("/queue", s.QueueHandler())
	s.Mux.Handle("GET /queue/add/{videoID}", s.AddHandler())
	s.Mux.Handle("GET /queue/playnext/{videoID}", s.PlayNextHandler())
	s.Mux.Handle("GET /queue/next", s.NextHandler(false))
	s.Mux.Handle("GET /queue/peek", s.NextHandler(true))
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
		s.Logger.Printf("queue handler")
		page := `<html><body><h1>Queue</h1></body></html>`
		if err := RenderHTML(w, http.StatusOK, page); err != nil {
			s.Logger.Printf("error rendering html: %v\n", err)
			http.Error(w, fmt.Sprintf("error rendering html: %v", err), http.StatusInternalServerError)
		}
	})
}

func (s *HTTPServer) AddHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Logger.Printf("add handler")
		id := r.PathValue("videoID")

		start := 0
		// Get the start time, in seconds, from the query string.
		startRaw := r.URL.Query().Get("start")
		if si, err := strconv.Atoi(startRaw); err == nil {
			start = si
		}

		// Add the video to the queue.
		s.Queue.Add(id, start)

		// Return a JSON response.
		msg := struct {
			Message string `json:"message"`
		}{"video added to queue"}
		if err := RenderJSON(w, http.StatusOK, msg); err != nil {
			s.Logger.Printf("error rendering json: %v\n", err)
			http.Error(w, fmt.Sprintf("error rendering json: %v", err), http.StatusInternalServerError)
		}
	})
}

func (s *HTTPServer) PlayNextHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Logger.Printf("play next handler")
		id := r.PathValue("videoID")

		start := 0
		// Get the start time, in seconds, from the query string.
		startRaw := r.URL.Query().Get("start")
		if si, err := strconv.Atoi(startRaw); err == nil {
			start = si
		}

		// Add the video to the queue.
		s.Queue.PlayNext(id, start)

		// Return a JSON response.
		msg := struct {
			Message string `json:"message"`
		}{"video added to queue"}
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
			s.Logger.Printf("peek handler")
			d, err = s.Queue.PeekNext()
		} else {
			s.Logger.Printf("next handler")
			d, err = s.Queue.GetNext()
		}

		if err != nil {
			if err == ErrQueueEmpty {
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
