package main

import (
	"context"
	"log"
	"os"

	ytqueuer "github.com/chadeldridge/yt-queuer"
)

func run(logger *log.Logger, ctx context.Context, addr string, port int, q *ytqueuer.Queue) error {
	// queue := ytqueuer.NewQueue()
	server := ytqueuer.NewHTTPServer(logger, addr, port, q)
	server.AddRoutes()

	return server.Start(ctx, 10)
}

func main() {
	ctx := context.Background()
	logger := log.New(os.Stdout, "ytqueuer: ", log.LstdFlags)

	// addr := "172.19.120.11"
	addr := ""
	port := 8080

	// Create a new empty queue.
	q := ytqueuer.NewQueue()

	logger.Printf("queue size: %d\n", len(q.Videos))
	logger.Printf("queue: %+v\n", q)

	if err := run(logger, ctx, addr, port, &q); err != nil {
		log.Printf("error: %v\n", err)
		os.Exit(1)
	}
}
