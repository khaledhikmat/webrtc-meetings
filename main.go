package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/joho/godotenv"

	"github.com/khaledhikmat/webrtc-meetings/server"
	"github.com/khaledhikmat/webrtc-meetings/service/meeting"
)

func main() {
	rootCanx := context.Background()
	canxCtx, cancel := signal.NotifyContext(rootCanx, os.Interrupt)

	// Load env vars
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	// Create and inject service layer
	server.MeetingService = meeting.NewService()
	defer server.MeetingService.Finalize()

	port := os.Getenv("HTTP_PORT")
	args := os.Args[1:]
	if len(args) > 0 {
		port = args[0]
	}

	defer func() {
		cancel()
	}()

	// Launch the http server
	httpServerErr := make(chan error, 1)
	go func() {
		httpServerErr <- server.Run(canxCtx, port)
	}()

	// Wait until server exits or context is cancelled
	for {
		select {
		case err := <-httpServerErr:
			fmt.Println("http server error", err)
			return
		case <-canxCtx.Done():
			fmt.Println("application cancelled...")
			cancel()
			// Wait until downstream processors are done
			fmt.Println("wait for 5 seconds until downstream processors are cancelled...")
			time.Sleep(5 * time.Second)
			return
		}
	}
}
