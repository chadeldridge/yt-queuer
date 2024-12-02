package application

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
)

func (s *HTTPServer) AddRoutes() {
	// Setup middleware.
	mwLogger := LoggerMiddleware(s.Logger)
	// Handle static assets.
	s.Mux.Handle("/", mwLogger(http.StripPrefix("/", http.FileServer(http.Dir("public")))))

	// ---- Playback Client Routes ----
	s.Mux.Handle("GET /pbcs", mwLogger(s.PBCListHandler()))
	s.Mux.Handle("POST /pbcs/register", mwLogger(s.PBCRegisterHandler())) // ?name="playback client name"

	// ---- Playlist Routes ----
	s.Mux.Handle("GET /playlists", mwLogger(s.PlaylistsHandler()))
	s.Mux.Handle("GET /playlists/{pbcID}", mwLogger(s.PlaylistHandler()))
	s.Mux.Handle(
		"POST /playlists/{pbcID}/{video_id}",
		mwLogger(s.AddHandler(false)),
	) // ?start=<start time in seconds>
	s.Mux.Handle(
		"POST /playlists/{pbcID}/{video_id}/next",
		mwLogger(s.AddHandler(true)),
	) // ?start=<start time in seconds>
	s.Mux.Handle("GET /playlists/{pbcID}/next", mwLogger(s.NextHandler(false)))
	s.Mux.Handle("GET /playlists/{pbcID}/peek", mwLogger(s.NextHandler(true)))
	s.Mux.Handle("DELETE /playlists/{pbcID}/{video_id}", mwLogger(s.RemoveHandler()))
	s.Mux.Handle("DELETE /playlists/{pbcID}", mwLogger(s.ClearHandler()))

	// ---- Wake On LAN Routes ----
	// s.Mux.Handle("GET /wol", mwLogger(s.WakeHandler()))
	s.Mux.Handle(
		"POST /wol/{pbcID}",
		mwLogger(s.WOLCreateHandler()),
	) // ?alias=<alias>&iface=<interface>&mac=<mac address>&port=<port>
	s.Mux.Handle("GET /wol/{pbcID}", mwLogger(s.WOLGetHandler()))
	s.Mux.Handle(
		"PUT /wol/{pbcID}",
		mwLogger(s.WOLUpdateHandler()),
	) // ?alias=<alias>&iface=<interface>&mac=<mac address>&port=<port>
	s.Mux.Handle("DELETE /wol/{pbcID}", mwLogger(s.WOLDeleteHandler()))
	s.Mux.Handle("POST /wol/{pbcID}/wake", mwLogger(s.WakeHandler())) // ?mac=<mac address>&port=<port>

	// ---- CEC Routes ----
	s.Mux.Handle(
		"POST /cec/{pbcID}",
		mwLogger(s.CECCreateHandler()),
	) // ?alias=<alias>&device=<device>&logical_addr=<logical address>&physical_addr=<physical address>
	s.Mux.Handle("GET /cec/{pbcID}", mwLogger(s.CECGetHandler()))
	s.Mux.Handle("PUT /cec/{pbcID}", mwLogger(s.CECUpdateHandler()))
	s.Mux.Handle("DELETE /cec/{pbcID}", mwLogger(s.CECDeleteHandler()))
	s.Mux.Handle("GET /cec/{pbcID}/power/status", mwLogger(s.CECPowerHandler("status")))
	s.Mux.Handle("POST /cec/{pbcID}/power/on", mwLogger(s.CECPowerHandler("on")))
	s.Mux.Handle("POST /cec/{pbcID}/power/off", mwLogger(s.CECPowerHandler("off")))
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

func RenderError(w http.ResponseWriter, msg string, status int) {
	if err := RenderJSON(w, status, struct{ message string }{message: msg}); err != nil {
		log.Printf("error rendering json: %v\n", err)
		http.Error(w, fmt.Sprintf("error rendering json: %v", err), http.StatusInternalServerError)
	}
}

func (s *HTTPServer) GetPBC(w http.ResponseWriter, r *http.Request) (PlaybackClient, error) {
	pbcID := r.PathValue("pbcID")
	if pbcID == "" {
		RenderError(w, "pbcID is empty", http.StatusBadRequest)
		return PlaybackClient{}, fmt.Errorf("pbcID is empty")
	}

	pbc, _, err := s.DB.PlaylistGet(pbcID)
	if err != nil {
		RenderError(w, "pbcID not found", http.StatusNotFound)
		return PlaybackClient{}, fmt.Errorf("pbcID not found: %s", pbcID)
	}

	return pbc, nil
}

func (s *HTTPServer) PlaylistsHandler() http.Handler {
	pbcs := s.Playlists.GetPBCs()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := RenderJSON(w, http.StatusOK, pbcs); err != nil {
			s.Logger.Printf("error rendering json: %v\n", err)
			RenderError(w, fmt.Sprintf("error rendering json: %v", err), http.StatusInternalServerError)
		}
	})
}

// ############################################################################################## //
// ####################################       Handlers       #################################### //
// ############################################################################################## //

// Use the following format to define a new handler:
/*
func (s *HTTPServer) HandlerFuncName() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handler actions here.
	})
}
*/

// PBCListHandler returns a http.Handler that lists all playback clients.
func (s *HTTPServer) PBCListHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pbcs, err := s.DB.PlaybackClientList()
		if err != nil {
			s.Logger.Printf("error listing playback clients: %v\n", err)
			RenderError(w, fmt.Sprintf("error listing playback clients: %v", err), http.StatusInternalServerError)
			return
		}

		if err := RenderJSON(w, http.StatusOK, pbcs); err != nil {
			s.Logger.Printf("error rendering json: %v\n", err)
			RenderError(w, fmt.Sprintf("error rendering json: %v", err), http.StatusInternalServerError)
		}
	})
}

// PBCRegisterHandler returns a http.Handler that registers a new playback client.
func (s *HTTPServer) PBCRegisterHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Logger.Printf("---- registering playback client ----")
		name := r.URL.Query().Get("name")
		if name == "" {
			s.Logger.Printf("name is empty\n")
			RenderError(w, "name is empty", http.StatusBadRequest)
			return
		}

		pbc, pl, err := s.DB.PlaylistGetByName(name)
		if err == nil {
			if _, ok := s.Playlists[pbc]; !ok {
				s.Playlists[pbc] = pl
			}
		} else {
			pbc, err = NewPlaybackClient(name)
			if err != nil {
				s.Logger.Printf("error creating playback client: %v\n", err)
				RenderError(w, fmt.Sprintf("error creating playback client: %v", err), http.StatusBadRequest)
			}

			pl = NewPlaylist()
			s.Playlists[pbc] = pl

			err := s.DB.PlaylistCreate(pbc, s.Playlists[pbc])
			if err != nil {
				s.Logger.Printf("error registering playback client: %v\n", err)
				RenderError(w, fmt.Sprintf("error registering playback client: %v", err), http.StatusInternalServerError)
				return
			}
		}

		/*
			// Create a new cookie with the playback client data.
			pbcCookie, err := NewPlaybackClientCookie(pbc)
			if err != nil {
				s.Logger.Printf("error creating playback client cookie: %v\n", err)
				RenderError(w, fmt.Sprintf("error creating playback client cookie: %v", err), http.StatusInternalServerError)
				return
			}
			pbcCookie.Write(w)
		*/

		if err := RenderJSON(w, http.StatusOK, pbc); err != nil {
			s.Logger.Printf("error rendering json: %v\n", err)
			RenderError(w, fmt.Sprintf("error rendering json: %v", err), http.StatusInternalServerError)
			return
		}
	})
}

// PlaylistHandler returns a http.Handler that lists the current playlist for the provided playback client ID.
func (s *HTTPServer) PlaylistHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pbc, err := s.GetPBC(w, r)
		if err != nil {
			return
		}

		pl, ok := s.Playlists[pbc]
		if !ok {
			s.Logger.Printf("error getting playlist: playlist not found\n")
			http.Error(w, "", http.StatusNotFound)
			return
		}

		if len(pl) == 0 {
			http.Error(w, "", http.StatusNoContent)
			return
		}

		if err := RenderJSON(w, http.StatusOK, pl); err != nil {
			s.Logger.Printf("error rendering json: %v\n", err)
			RenderError(w, fmt.Sprintf("error rendering json: %v", err), http.StatusInternalServerError)
		}
	})
}

// AddHandler returns a http.Handler that adds a video to the playback client playlist for the
// provided playback client ID. If next is true, the video is added to the beginning of the
// playlist.
func (s *HTTPServer) AddHandler(next bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pbc, err := s.GetPBC(w, r)
		if err != nil {
			return
		}

		// Get the video ID from the path.
		vid := r.PathValue("video_id")

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

		if next {
			if err := s.Playlists.PlayNext(pbc, vid, start); err != nil {
				s.Logger.Printf("error adding video to playlist: %v\n", err)
				RenderError(w, fmt.Sprintf("error adding video to playlist: %v", err), http.StatusBadRequest)
				return
			}
		} else {
			// Add the video to the playlist. If there is an error, send a 400 Bad Request response.
			if err := s.Playlists.Add(pbc, vid, start); err != nil {
				s.Logger.Printf("error adding video to playlist: %v\n", err)
				RenderError(w, fmt.Sprintf("error adding video to playlist: %v", err), http.StatusBadRequest)
				return
			}
		}

		// Write playlist
		err = s.Playlists.Save(s.DB, pbc)
		if err != nil {
			s.Logger.Printf("error saving playlist: %v\n", err)
			RenderError(w, fmt.Sprintf("error saving playlist: %v", err), http.StatusInternalServerError)
			return
		}

		// Send a JSON response.
		msg := struct {
			Message  string   `json:"message"`
			Playlist Playlist `json:"playlist"`
		}{
			Message:  "video added to playlist",
			Playlist: s.Playlists[pbc],
		}

		if err := RenderJSON(w, http.StatusOK, msg); err != nil {
			s.Logger.Printf("error rendering json: %v\n", err)
			RenderError(w, fmt.Sprintf("error rendering json: %v", err), http.StatusInternalServerError)
		}
	})
}

// NextHandler returns a http.Handler that returns the next or "currently playing" video in the
// playback client playlist for the provided. If peek is true, NexHandler returns the second video
// in the playlist.
func (s *HTTPServer) NextHandler(peek bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pbc, err := s.GetPBC(w, r)
		if err != nil {
			return
		}

		var d VideoDetails
		if peek {
			d, err = s.Playlists.PeekNext(pbc)
		} else {
			d, err = s.Playlists.GetNext(pbc)
		}

		if err != nil {
			if err == ErrPlaylistEmpty || err == ErrEndOfPlaylist {
				http.Error(w, "", http.StatusNoContent)
				return
			}

			s.Logger.Printf("error getting next video: %v\n", err)
			RenderError(w, fmt.Sprintf("error getting next video: %v", err), http.StatusInternalServerError)
			return
		}

		// Return video details.
		if err := RenderJSON(w, http.StatusOK, d); err != nil {
			s.Logger.Printf("error rendering json: %v\n", err)
			RenderError(w, fmt.Sprintf("error rendering json: %v", err), http.StatusInternalServerError)
		}
	})
}

// RemoveHandler returns a http.Handler that removes a video from the playlist for the provided
// playback client ID.
func (s *HTTPServer) RemoveHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pbc, err := s.GetPBC(w, r)
		if err != nil {
			return
		}

		vid := r.PathValue("video_id")
		if vid == "" {
			s.Logger.Printf("video ID is empty\n")
			RenderError(w, "video ID is empty", http.StatusBadRequest)
			return
		}

		err = s.Playlists.Remove(pbc, vid)
		if err != nil {
			s.Logger.Printf("error removing video from playlist: %v\n", err)
			RenderError(w, fmt.Sprintf("error removing video from playlist: %v", err), http.StatusBadRequest)
			return
		}

		http.Error(w, "", http.StatusNoContent)

		// Write playlist
		err = s.Playlists.Save(s.DB, pbc)
		if err != nil {
			s.Logger.Printf("error saving playlist: %v\n", err)
			RenderError(w, fmt.Sprintf("error saving playlist: %v", err), http.StatusInternalServerError)
			return
		}
	})
}

// ClearHandler returns a http.Handler that clears the playlist for the provided playback client ID.
func (s *HTTPServer) ClearHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pbc, err := s.GetPBC(w, r)
		if err != nil {
			return
		}

		s.Playlists.Clear(pbc)
		http.Error(w, "", http.StatusNoContent)

		// Write playlist
		err = s.Playlists.Save(s.DB, pbc)
		if err != nil {
			s.Logger.Printf("error saving playlist: %v\n", err)
			RenderError(w, fmt.Sprintf("error saving playlist: %v", err), http.StatusInternalServerError)
			return
		}
	})
}

// CreateHandler returns a http.Handler that creates a new Wake On LAN entry.
func (s *HTTPServer) WOLCreateHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pbcID := r.PathValue("pbcID")
		if pbcID == "" {
			RenderError(w, "pbcID is empty", http.StatusBadRequest)
			return
		}

		alias := r.URL.Query().Get("alias")
		if alias == "" {
			RenderError(w, "alias is empty", http.StatusBadRequest)
			return
		}

		iface := r.URL.Query().Get("iface")
		if iface == "" {
			RenderError(w, "iface is empty", http.StatusBadRequest)
			return
		}

		mac := r.URL.Query().Get("mac")
		if mac == "" {
			RenderError(w, "mac is empty", http.StatusBadRequest)
			return
		}

		portParam := r.URL.Query().Get("port")
		if portParam == "" {
			RenderError(w, "port is empty", http.StatusBadRequest)
			return
		}

		port, err := strconv.Atoi(portParam)
		if err != nil {
			RenderError(w, "invalid port", http.StatusBadRequest)
			return
		}

		s.Logger.Printf(
			"creating Wake On LAN entry: {pbcID: %s, alias: %s, iface: %s, mac: %s, port: %d\n",
			pbcID,
			alias,
			iface,
			mac,
			port,
		)

		wol, err := NewWOL(pbcID, alias, iface, mac, port)
		if err != nil {
			s.Logger.Printf("error creating Wake On LAN struct: %v\n", err)
			RenderError(w, fmt.Sprintf("error creating Wake On LAN struct: %v", err), http.StatusBadRequest)
			return
		}

		err = s.DB.WOLCreate(wol)
		if err != nil {
			s.Logger.Printf("error creating Wake On LAN entry: %v\n", err)
			RenderError(w, fmt.Sprintf("error creating Wake On LAN entry: %v", err), http.StatusInternalServerError)
			return
		}

		http.Error(w, "", http.StatusCreated)
	})
}

func (s *HTTPServer) WOLGetHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pbcID := r.PathValue("pbcID")
		if pbcID == "" {
			RenderError(w, "pbcID is empty", http.StatusBadRequest)
			return
		}

		wol, err := s.DB.WOLGet(pbcID)
		if err != nil {
			// Return No Content status of no record was found.
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "", http.StatusNoContent)
				return
			}

			// Return any other errors.
			s.Logger.Printf("error getting Wake On LAN entry: %v\n", err)
			RenderError(w, fmt.Sprintf("error getting Wake On LAN entry: %v", err), http.StatusInternalServerError)
			return
		}

		// Return WOL entry.
		if err := RenderJSON(w, http.StatusOK, wol); err != nil {
			s.Logger.Printf("error rendering json: %v\n", err)
			RenderError(w, fmt.Sprintf("error rendering json: %v", err), http.StatusInternalServerError)
		}
	})
}

func (s *HTTPServer) WOLUpdateHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pbcID := r.PathValue("pbcID")
		if pbcID == "" {
			RenderError(w, "pbcID is empty", http.StatusBadRequest)
			return
		}

		alias := r.URL.Query().Get("alias")
		if alias == "" {
			RenderError(w, "alias is empty", http.StatusBadRequest)
			return
		}

		iface := r.URL.Query().Get("iface")
		if iface == "" {
			RenderError(w, "iface is empty", http.StatusBadRequest)
			return
		}

		mac := r.URL.Query().Get("mac")
		if mac == "" {
			RenderError(w, "mac is empty", http.StatusBadRequest)
			return
		}

		portParam := r.URL.Query().Get("port")
		if portParam == "" {
			RenderError(w, "port is empty", http.StatusBadRequest)
			return
		}

		port, err := strconv.Atoi(portParam)
		if err != nil {
			RenderError(w, "invalid port", http.StatusBadRequest)
			return
		}

		wol, err := NewWOL(pbcID, alias, iface, mac, port)
		if err != nil {
			s.Logger.Printf("error updating Wake On LAN entry: %v\n", err)
			RenderError(w, fmt.Sprintf("error updating Wake On LAN entry: %v", err), http.StatusBadRequest)
			return
		}

		err = s.DB.WOLUpdate(wol)
		if err != nil {
			s.Logger.Printf("error updating Wake On LAN entry: %v\n", err)
			RenderError(w, fmt.Sprintf("error updating Wake On LAN entry: %v", err), http.StatusInternalServerError)
			return
		}

		http.Error(w, "", http.StatusOK)
	})
}

func (s *HTTPServer) WOLDeleteHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pbcID := r.PathValue("pbcID")
		if pbcID == "" {
			RenderError(w, "pbcID is empty", http.StatusBadRequest)
			return
		}

		err := s.DB.WOLDelete(pbcID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			s.Logger.Printf("error deleting Wake On LAN entry: %v\n", err)
			RenderError(w, fmt.Sprintf("error deleting Wake On LAN entry: %v", err), http.StatusInternalServerError)
			return
		}

		http.Error(w, "", http.StatusNoContent)
	})
}

// WakeHandler returns a http.Handler that sends a Wake On LAN packet to the provided interface.
func (s *HTTPServer) WakeHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pbcID := r.PathValue("pbcID")
		if pbcID == "" {
			RenderError(w, "pbcID is empty", http.StatusBadRequest)
			return
		}

		wol, err := s.DB.WOLGet(pbcID)
		if err != nil {
			// Return No Content status of no record was found.
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "", http.StatusNoContent)
				return
			}

			// Return any other errors.
			s.Logger.Printf("error getting Wake On LAN entry: %v\n", err)
			RenderError(w, fmt.Sprintf("error getting Wake On LAN entry: %v", err), http.StatusInternalServerError)
			return
		}

		cmd := exec.Command("sudo", "wol", wol.Interface, wol.MAC, strconv.Itoa(wol.Port))
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			s.Logger.Printf(out.String())
			s.Logger.Printf("error waking device: %v\n", err)
			RenderError(w, fmt.Sprintf("error waking device: %v", err), http.StatusInternalServerError)
			return
		}

		http.Error(w, "", http.StatusOK)
	})
}

// cec/{pbcID}?alias=<alias>&device=<device>&logical_addr=<logical address>&physical_addr=<physical address>
func (s *HTTPServer) CECCreateHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pbcID := r.PathValue("pbcID")
		if pbcID == "" {
			RenderError(w, "pbcID is empty", http.StatusBadRequest)
			return
		}

		alias := r.URL.Query().Get("alias")
		if alias == "" {
			RenderError(w, "alias is empty", http.StatusBadRequest)
			return
		}

		device := r.URL.Query().Get("device")
		if device == "" {
			RenderError(w, "device is empty", http.StatusBadRequest)
			return
		}

		la := r.URL.Query().Get("logical_addr")
		if la == "" {
			RenderError(w, "logical_addr is empty", http.StatusBadRequest)
			return
		}

		logicalAddr, err := strconv.Atoi(la)
		if err != nil {
			RenderError(w, "invalid logical_addr: must be int 0 - 15", http.StatusBadRequest)
			return
		}

		physicalAddr := r.URL.Query().Get("physical_addr")
		if physicalAddr == "" {
			RenderError(w, "physical_addr is empty", http.StatusBadRequest)
			return
		}

		cec, err := NewCEC(pbcID, alias, device, physicalAddr, logicalAddr)
		if err != nil {
			s.Logger.Printf("error creating CEC struct: %v\n", err)
			RenderError(w, fmt.Sprintf("error creating CEC struct: %v", err), http.StatusBadRequest)
			return
		}

		err = s.DB.CECCreate(cec)
		if err != nil {
			s.Logger.Printf("error creating CEC entry: %v\n", err)
			RenderError(w, fmt.Sprintf("error creating CEC entry: %v", err), http.StatusInternalServerError)
			return
		}

		if err := RenderJSON(w, http.StatusCreated, cec); err != nil {
			s.Logger.Printf("error rendering json: %v\n", err)
			RenderError(w, fmt.Sprintf("error rendering json: %v", err), http.StatusInternalServerError)
		}
	})
}

func (s *HTTPServer) CECGetHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pbcID := r.PathValue("pbcID")
		if pbcID == "" {
			RenderError(w, "pbcID is empty", http.StatusBadRequest)
			return
		}

		cec, err := s.DB.CECGet(pbcID)
		if err != nil {
			// Return No Content status of no record was found.
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "", http.StatusNoContent)
				return
			}

			// Return any other errors.
			s.Logger.Printf("error getting CEC entry: %v\n", err)
			RenderError(w, fmt.Sprintf("error getting CEC entry: %v", err), http.StatusInternalServerError)
			return
		}

		if err := RenderJSON(w, http.StatusOK, cec); err != nil {
			s.Logger.Printf("error rendering json: %v\n", err)
			RenderError(w, fmt.Sprintf("error rendering json: %v", err), http.StatusInternalServerError)
		}
	})
}

// cec/{pbcID}?alias=<alias>&device=<device>&logical_addr=<logical address>&physical_addr=<physical address>
func (s *HTTPServer) CECUpdateHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pbcID := r.PathValue("pbcID")
		if pbcID == "" {
			RenderError(w, "pbcID is empty", http.StatusBadRequest)
			return
		}

		alias := r.URL.Query().Get("alias")
		if alias == "" {
			RenderError(w, "alias is empty", http.StatusBadRequest)
			return
		}

		device := r.URL.Query().Get("device")
		if device == "" {
			RenderError(w, "device is empty", http.StatusBadRequest)
			return
		}

		la := r.URL.Query().Get("logical_addr")
		if la == "" {
			RenderError(w, "logical_addr is empty", http.StatusBadRequest)
			return
		}

		logicalAddr, err := strconv.Atoi(la)
		if err != nil {
			RenderError(w, "invalid logical_addr: must be int 0 - 15", http.StatusBadRequest)
			return
		}

		physicalAddr := r.URL.Query().Get("physical_addr")
		if physicalAddr == "" {
			RenderError(w, "physical_addr is empty", http.StatusBadRequest)
			return
		}

		cec, err := NewCEC(pbcID, alias, device, physicalAddr, logicalAddr)
		if err != nil {
			s.Logger.Printf("error creating CEC struct: %v\n", err)
			RenderError(w, fmt.Sprintf("error creating CEC struct: %v", err), http.StatusBadRequest)
			return
		}

		err = s.DB.CECUpdate(cec)
		if err != nil {
			s.Logger.Printf("error updating CEC entry: %v\n", err)
			RenderError(w, fmt.Sprintf("error updating CEC entry: %v", err), http.StatusInternalServerError)
			return
		}

		http.Error(w, "", http.StatusOK)
	})
}

func (s *HTTPServer) CECDeleteHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pbcID := r.PathValue("pbcID")
		if pbcID == "" {
			RenderError(w, "pbcID is empty", http.StatusBadRequest)
			return
		}

		err := s.DB.CECDelete(pbcID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "", http.StatusNoContent)
				return
			}

			s.Logger.Printf("error deleting CEC entry: %v\n", err)
			RenderError(w, fmt.Sprintf("error deleting CEC entry: %v", err), http.StatusInternalServerError)
			return
		}

		http.Error(w, "", http.StatusNoContent)
	})
}

/*
CECPowerHandler returns a http.Handler that sends a CEC command to the provided playback client.

cec/{pbcID}/power/{cmd} where {cmd} is the command to send to the device.

# Playback Client ID

{pbcID} is the playback client ID which identifies the playback client the cec config is stored under.

# Command

{cmd} can be "on", "off", or "status":

	"on" or "off" returns a 200 and no body when the command was successfully executed.
	"status" returns a 200 and a body containing a json object with the power status of the device. {power: "on"}
*/
func (s *HTTPServer) CECPowerHandler(cmd string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pbcID := r.PathValue("pbcID")
		if pbcID == "" {
			RenderError(w, "pbcID is empty", http.StatusBadRequest)
			return
		}

		if cmd == "" {
			RenderError(w, "cmd is empty", http.StatusBadRequest)
			return
		}

		// Get the CEC config for the provided playback client ID.
		cec, err := s.DB.CECGet(pbcID)
		if err != nil {
			// Return No Content status of no record was found.
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "", http.StatusNoContent)
				return
			}

			// Return any other errors.
			s.Logger.Printf("error getting CEC entry: %v\n", err)
			RenderError(w, fmt.Sprintf("error getting CEC entry: %v", err), http.StatusInternalServerError)
			return
		}

		switch cmd {
		case "on":
			err := cec.PowerOn()
			if err != nil {
				s.Logger.Printf("error powering on device: %v\n", err)
				RenderError(w, fmt.Sprintf("error powering on device: %v", err), http.StatusInternalServerError)
				return
			}
		case "off":
			err := cec.PowerOff()
			if err != nil {
				s.Logger.Printf("error powering off device: %v\n", err)
				RenderError(w, fmt.Sprintf("error powering off device: %v", err), http.StatusInternalServerError)
				return
			}
		case "status":
			status, err := cec.PowerStatus()
			if err != nil {
				s.Logger.Printf("error getting power status: %v\n", err)
				RenderError(w, fmt.Sprintf("error getting power status: %v", err), http.StatusInternalServerError)
				return
			}

			s.Logger.Printf("power status (%s): %s\n", cec.Alias, status)
			if err := RenderJSON(w, http.StatusOK, map[string]string{"power": status}); err != nil {
				s.Logger.Printf("error rendering json: %v\n", err)
				RenderError(w, fmt.Sprintf("error rendering json: %v", err), http.StatusInternalServerError)
			}
		default:
			RenderError(w, fmt.Sprintf("invalid command: '%s'", cmd), http.StatusBadRequest)
			return
		}

		http.Error(w, "", http.StatusOK)
	})
}
