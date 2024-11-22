package core

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/coder/websocket"
	"github.com/libp2p/go-yamux"
	"golang.org/x/sync/errgroup"
)

type RelayServer struct {
	sync.RWMutex
	portNext uint16
	agents   map[uint16]*yamux.Session
	// Logf controls where logs are sent.
	Logf func(f string, v ...interface{})
}

// set next port and return it
// todo find unused port randomly
func (s *RelayServer) nextPort() uint16 {
	s.Lock()
	defer s.Unlock()
	s.portNext = s.portNext + 1
	return s.portNext
}

func (s *RelayServer) InitPort(p uint16) {
	s.Lock()
	defer s.Unlock()
	s.portNext = p
}

// / listen for websocket connection from agents
// / transform it to net.Conn
// / yamux over new net.Conn
func (s *RelayServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("Got HTTP request from (%s) with Host: (%s):  %s", r.RemoteAddr, r.Host, r.URL.String())
	if r.Header.Get("Upgrade") != "websocket" {
		w.Header().Set("Location", "https://www.microsoft.com/")
		w.WriteHeader(http.StatusFound) // Use 302 status code for redirect
		// fmt.Fprintf(w, "OK")
		return
	}
	// todo authentication via bearer jwt
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols:       nil,
		InsecureSkipVerify: true,
		OriginPatterns: []string{
			"*",
		},
		CompressionMode:      0,
		CompressionThreshold: 0,
	})
	if err != nil {
		log.Printf("Error upgrading to socket (%s):  %v", r.RemoteAddr, err)
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	defer c.CloseNow()

	websocketTunnelConn := websocket.NetConn(context.Background(), c, websocket.MessageBinary)

	//Add connection to yamux
	session, err := yamux.Client(websocketTunnelConn, nil)
	if err != nil {
		log.Printf("Error creating client in yamux for (%s): %v", r.RemoteAddr, err)
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	port := s.nextPort()

	ctx := r.Context()

	s.listenForClientOnPort(ctx, port, session)

	c.Close(websocket.StatusNormalClosure, "")
	s.Logf("agent websocket to port %d closed", port)
}

// bind on unique port per agent
// listen for connection for clients and pipe data via yamux stream
func (s *RelayServer) listenForClientOnPort(context context.Context, port uint16, session *yamux.Session) {
	s.Logf("listen for client on port: %d", port)

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		s.Logf("Error listening on port %d: %v", port, err)
		return
	}
	for {
		select {
		case <-context.Done():
			s.Logf("agent tunnel closed")
			break
		default:
			// do not block
		}

		conn, err := l.Accept()
		if err != nil {
			s.Logf("Error accepting on port %d: %v", port, err)
			break
		}

		s.Logf("accept new conn from client on port %d", port)

		stream, err := session.Open()
		if err != nil {
			s.Logf("Error opening stream on port %d: %v", port, err)
			break
		}

		// as this is a bidirectional proxy if one of our connections closes we need to clean up its "sister" connection
		eg, ctx := errgroup.WithContext(context)
		go func() {
			s.Logf("start proxy")
			eg.Go(proxy(ctx, stream, conn))
			eg.Go(proxy(ctx, conn, stream))

			_ = eg.Wait() // waiting here to avoid closing the connections prematurely
			s.Logf("proxy between conn and stream done")
			// notify the other end this yamux stream is closed
			stream.Close()
		}()
	}

	s.Logf("stop proxy on port %d", port)
}
