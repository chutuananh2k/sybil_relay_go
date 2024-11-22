package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sybil_relay_go/core"
	"time"
)

var listenAddr string

func main() {
	log.SetFlags(0)

	flag.StringVar(&listenAddr, "l", ":8080", "listen address")
	flag.Parse()

	if listenAddr == "" {
		flag.Usage()
		return
	}

	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

// run starts a http.Server for the passed in address
// with all requests handled by RelayServer.
func run() error {
	l, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	log.Printf("listening on ws://%v", l.Addr())

	relayServer := &core.RelayServer{Logf: log.Printf}
	// avoid privilege port
	relayServer.InitPort(3000)

	s := &http.Server{
		Handler:      relayServer,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
	}

	errc := make(chan error, 1)

	go func() {
		errc <- s.Serve(l)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	select {
	case err := <-errc:
		log.Printf("failed to serve: %v", err)
	case sig := <-sigs:
		log.Printf("terminating: %v", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	return s.Shutdown(ctx)
}
