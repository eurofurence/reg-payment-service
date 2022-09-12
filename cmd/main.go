package main

import (
	"github.com/eurofurence/reg-payment-service/internal/server"

	"time"
	"os/signal"
	"syscall"
	"context"
	"log"
	"os"
)

func main() {
	// TODO start implementing your service here

	ctx, cancel := context.WithCancel(context.Background())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sig
		cancel()
		log.Println("Stopping services now")

		tCtx, tcancel := context.WithTimeout(context.Background(), time.Second*5)
		defer tcancel()

		if err := server.Shutdown(tCtx); err != nil {
			log.Fatal("Couldn't shutdown server gracefully")
		}
	}()

	handler := server.Create()
	server.Serve(ctx, handler)
}
