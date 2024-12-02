package application

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// RECHECK: Remove this later if it is still not used. Moved cookie management to client side due to issues with the cookie being set on the server side.

const (
	COOKIE_NAME    = "playback_client"
	COOKIE_EXPIRES = 365 * 24 * time.Hour
)

var ErrExpiredCookie = fmt.Errorf("cookie has expired")

type PlaybackClientCookie struct {
	Value   string // Bearer token used to lookup JWT from AuthCache.
	Path    string
	Expires time.Time // Cookie expiration time.
}

func NewPlaybackClientCookie(pbc PlaybackClient) (PlaybackClientCookie, error) {
	if pbc.ID == "" {
		return PlaybackClientCookie{}, fmt.Errorf("NewPlaybackClientCookie: bearer - %s", ErrParamEmpty)
	}

	pbcData, err := json.Marshal(pbc)
	if err != nil {
		return PlaybackClientCookie{}, fmt.Errorf("NewPlaybackClientCookie: %w", err)
	}

	expires := time.Now().Add(COOKIE_EXPIRES)
	return PlaybackClientCookie{
		Value:   string(pbcData),
		Path:    "",
		Expires: expires,
	}, nil
}

func NewPlaybackClientCookieFromCookie(cookie *http.Cookie) (PlaybackClientCookie, error) {
	if cookie == nil {
		return PlaybackClientCookie{}, fmt.Errorf("NewPlaybackClientCookieFromCookie: cookie %s", ErrParamEmpty)
	}

	s := PlaybackClientCookie{
		Value: cookie.Value,
	}

	return s, s.Validate()
}

func GetPlaybackClientCookie(r *http.Request) (PlaybackClientCookie, error) {
	cookie, err := r.Cookie(COOKIE_NAME)
	if err != nil {
		return PlaybackClientCookie{}, err
	}

	// Create, validate, and return a new PlaybackClientCookie.
	return NewPlaybackClientCookieFromCookie(cookie)
}

func (s PlaybackClientCookie) PlaybackClient() (PlaybackClient, error) {
	if s.Value == "" {
		return PlaybackClient{}, fmt.Errorf("PlaybackClientCookie.PlaybackClient: value - %s", ErrParamEmpty)
	}

	var pbc PlaybackClient
	err := json.Unmarshal([]byte(s.Value), &pbc)
	if err != nil {
		return PlaybackClient{}, fmt.Errorf("PlaybackClientCookie.PlaybackClient: %w", err)
	}

	return pbc, nil
}

func (s PlaybackClientCookie) Validate() error {
	if s.Value == "" {
		return fmt.Errorf("PlaybackClientCookie.Validate: value - %s", ErrParamEmpty)
	}

	return nil
}

func (s PlaybackClientCookie) Write(w http.ResponseWriter) error {
	err := s.Validate()
	if err != nil {
		return fmt.Errorf("PlaybackClientCookie.Write: %w", err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     COOKIE_NAME,
		Value:    s.Value,
		Path:     s.Path,
		Expires:  s.Expires,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	})

	return nil
}
