package ytqueuer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/felixge/httpsnoop"
)

type HTTPServer struct {
	Addr        string
	Logger      *log.Logger
	Queue       *Queue
	TLSCertFile string
	TLSKeyFile  string

	Handler http.Handler
	// Mux saves the http.ServeMux instance. This provides easier access to the
	// mux without having to enforce a ref type on HTTPServer.Handler everytime.
	// We can now use HTTPServer.Mux.Handle() instead of HTTPServer.Handler.(*http.ServeMux).Handle().
	Mux *http.ServeMux
}

func NewHTTPServer(logger *log.Logger, addr string, port int, certFile string, keyFile string, q *Queue) HTTPServer {
	mux := http.NewServeMux()
	return HTTPServer{
		Addr:        net.JoinHostPort(addr, strconv.Itoa(port)),
		Logger:      logger,
		Queue:       q,
		TLSCertFile: certFile,
		TLSKeyFile:  keyFile,
		Handler:     mux, Mux: mux,
	}
}

func (s *HTTPServer) Start(ctx context.Context, timeoutSec int) error {
	httpServer := &http.Server{
		Addr:    s.Addr,
		Handler: s.Handler,
	}

	// Start the server.
	srvErr := make(chan error)
	go func() {
		s.Logger.Printf("http server listening on %s\n", httpServer.Addr)
		// err := httpServer.ListenAndServe()
		err := httpServer.ListenAndServeTLS(s.TLSCertFile, s.TLSKeyFile)
		if err != nil {
			if err == http.ErrServerClosed {
				s.Logger.Printf("server closed")
				close(srvErr)
			} else {
				srvErr <- err
			}
		}
	}()

	// Create a wait group to handle a graceful shutdown.
	var wg sync.WaitGroup
	wg.Add(1)
	wgErr := make(chan error)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		shutdownCtx := context.Background()
		shutdownCtx, cancel := context.WithTimeout(
			shutdownCtx,
			time.Duration(timeoutSec)*time.Second,
		)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			wgErr <- fmt.Errorf("http server shutdown error: %w", err)
		}
	}()
	wg.Wait()

	select {
	case err := <-srvErr:
		if err != nil {
			return err
		}
	case err := <-wgErr:
		if err != nil {
			return err
		}
	}

	return nil
}

type ReqMetrics struct {
	ClientIP     string        `json:"client_ip"`
	RequestTime  time.Time     `json:"request_time"`
	Method       string        `json:"method"`
	URI          string        `json:"uri"`
	ResponseCode int           `json:"response_code"`
	ResponseSize int64         `json:"response_size"`
	Referer      string        `json:"referer"`
	UserAgent    string        `json:"user_agent"`
	Duration     time.Duration `json:"duration"`
}

func NewReqMetrics(r *http.Request) ReqMetrics {
	return ReqMetrics{
		ClientIP:    ClientIP(r),
		RequestTime: time.Now(),
		Method:      r.Method,
		URI:         r.RequestURI,
		Referer:     r.Referer(),
		UserAgent:   r.UserAgent(),
	}
}

func ClientIP(r *http.Request) string {
	// Headers are not case sensitive. Initial caps for readability.
	if x := r.Header.Get("X-Real-IP"); x != "" {
		return x
	}
	if x := r.Header.Get("X-Forwarded-For"); x != "" {
		// The first IP in the list should be the client IP.
		return strings.Split(x, ", ")[0]
	}

	return remoteAddr(r)
}

// remoteAddr returns the remote address from the request without the port.
func remoteAddr(r *http.Request) string {
	addr := r.RemoteAddr
	if strings.Contains(addr, ":") {
		addr, _, _ := net.SplitHostPort(addr)
		return addr
	}

	return addr
}

func LoggerMiddleware(logger *log.Logger) func(http.Handler) http.Handler {
	accessLogger := log.New(logger.Writer(), "ytqueuer-access: ", 0)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				// logger.Debugf("request: %s %s\n", r.Method, r.URL.Path)
				rm := NewReqMetrics(r)
				m := httpsnoop.CaptureMetrics(next, w, r)
				rm.ResponseCode = m.Code
				rm.ResponseSize = m.Written
				rm.Duration = m.Duration

				// Add request metrics to the global metrics.
				// RecordRequest(rm.ResponseCode, rm.Duration)
				log, err := json.Marshal(rm)
				if err != nil {
					logger.Printf("LoggerMiddleware: failed to marshal request metrics: %v\n", err)
					return
				}

				// Don't fill the logs with clients trying to get the next video on
				// an empty queue.
				if rm.URI == "/queue/next" && rm.ResponseCode == http.StatusNoContent {
					return
				}

				// Log the request metrics.
				accessLogger.Print(string(log))
			})
	}
}
