package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	var (
		addr = flag.String("addr", ":8080", "HTTP address to bind to")
		ttl  = flag.Duration("ttl", 60*time.Second, "Time to wait before shutdown is enforced")
	)
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile | log.LUTC)
	log.SetOutput(os.Stdout)

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}

	// Create a web server
	mux := http.DefaultServeMux
	mux.HandleFunc("/", indexHandler)
	srv := &http.Server{
		Handler: mux,
	}

	log.Printf("Starting server at %s", *addr)

	idleConnsClosed := make(chan struct{})
	errCh := make(chan error, 1)

	go func() {
		errCh <- srv.Serve(lis)
	}()

	go func() {
		defer close(idleConnsClosed)

		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
		for {
			switch sig := <-ch; sig {
			case syscall.SIGINT, syscall.SIGTERM:
				log.Printf("Received signal %v. Starting shutdown.", sig)
				ctx, cancel := context.WithTimeout(context.Background(), *ttl)
				defer cancel()
				errCh <- srv.Shutdown(ctx)
				return
			case syscall.SIGHUP:
				log.Printf("Received signal %v.", sig)
			default:
				log.Printf("Received signal %v.", sig)
			}
		}
	}()

	// ErrServerClosed is returned after the HTTP server closed. That's not an error.
	if err := <-errCh; err != nil && err != http.ErrServerClosed {
		log.Printf("Exiting: %v", err)
	}

	<-idleConnsClosed

	log.Print("Exiting.")
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(15 * time.Second)
	fmt.Fprintf(w, "Hello world. The time here is %s.\n", time.Now())
}
