package ytqueuer

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type HTTPServer struct {
	Addr   string
	Logger *log.Logger
	Queue  *Queue

	Handler http.Handler
	// Mux saves the http.ServeMux instance. This provides easier access to the
	// mux without having to enforce a ref type on HTTPServer.Handler everytime.
	// We can now use HTTPServer.Mux.Handle() instead of HTTPServer.Handler.(*http.ServeMux).Handle().
	Mux *http.ServeMux
}

func NewHTTPServer(logger *log.Logger, addr string, port int, q *Queue) HTTPServer {
	mux := http.NewServeMux()
	return HTTPServer{
		Addr:    net.JoinHostPort(addr, strconv.Itoa(port)),
		Logger:  logger,
		Queue:   q,
		Handler: mux, Mux: mux,
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
		err := httpServer.ListenAndServe()
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
