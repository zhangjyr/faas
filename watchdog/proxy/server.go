package proxy

import (
	"fmt"
	"errors"
	"log"
	"net"
	"sync"

	"github.com/openfaas/faas/watchdog/logger"
)

const REMOTE_PRIMARY int = 0
const REMOTE_SECONDARY int = 1
const REMOTE_MAX int = 2

type Server struct {
	Addr           string // TCP address to listen on
	Verbose        bool
	Debug          bool

	mu             sync.Mutex
	remotes        [REMOTE_MAX]*net.TCPAddr
	listener       *net.TCPListener
	activeConn     map[*forwardConnection]struct{}
	connid         uint64
	listening      chan struct{}
	done           chan struct{}
}

// ErrServerClosed is returned by the Server after a call to Shutdown or Close.
var ErrServerClosed = errors.New("proxy: Server closed")

// ErrServerListening is returned by the Server if server is listening.
var ErrServerListening = errors.New("proxy: Server is listening")

func (srv *Server) Listen() (*net.TCPListener, error) {
	// Parse host address
	laddr, err := net.ResolveTCPAddr("tcp", srv.Addr)
	if err != nil {
		return nil, err
	}

	srv.mu.Lock()
	defer srv.mu.Unlock()

	if srv.isListeningLocked() {
		err = ErrServerListening
	} else {
		srv.listener, err = net.ListenTCP("tcp", laddr)
	}
	return srv.listener, err
}

func (srv *Server) ListenAndProxy(remoteAddr string) error {
	// Override remote address
	err := srv.setRemoteAddr(remoteAddr)
	if err != nil {
		return err
	}

	// Start server
	listener, err := srv.Listen()
	if err != nil {
		return err
	}
	defer srv.Close()

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			select {
			case <-srv.isDone():
				return ErrServerClosed
			default:
			}
			log.Printf("Failed to accept connection '%s'\n", err)
			continue
		}
		srv.connid++

		logLevel := logger.LOG_LEVEL_INFO
		if srv.Debug {
			logLevel = logger.LOG_LEVEL_ALL
		}

		var fconn *forwardConnection
		fconn = newForwardConnection(conn, listener.Addr().(*net.TCPAddr), srv.getRemoteAddr())
		fconn.Debug = srv.Debug
		fconn.Nagles = true
		fconn.Log = &logger.ColorLogger{
			Verbose:     srv.Verbose,
			Level:       logLevel,
			Prefix:      fmt.Sprintf("Connection #%03d ", srv.connid),
			Color:       true,
		}

		go func() {
			defer  conn.Close()

			fconn.forward(srv)
		}()
	}
}

func (srv *Server) IsListening() bool {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	return srv.isListeningLocked()
}

func (srv *Server) Switch(remoteAddr string) error {
	return srv.setRemoteAddr(remoteAddr)
}

func (srv *Server) Close() error {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	srv.doneLocked()

	var err error
	if srv.isListeningLocked() {
		err = srv.listener.Close()
	}

	for fconn := range srv.activeConn {
		fconn.close()
		delete(srv.activeConn, fconn)
	}
	
	return err
}

func (srv *Server) getRemoteAddr() *net.TCPAddr {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	return srv.remotes[REMOTE_PRIMARY];
}

func (srv *Server) setRemoteAddr(remoteAddr string) error {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	raddr, err := net.ResolveTCPAddr("tcp", remoteAddr)
	if err != nil {
		return err
	}

	srv.remotes[REMOTE_PRIMARY] = raddr
	return nil
}

func (srv *Server) trackConn(fconn *forwardConnection, add bool) {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	if srv.activeConn == nil {
		srv.activeConn = make(map[*forwardConnection]struct{})
	}
	if add {
		srv.activeConn[fconn] = struct{}{}
	} else {
		delete(srv.activeConn, fconn)
	}
}

func (srv *Server) isListeningLocked() bool {
	return srv.listener != nil
}

func (srv *Server) isDone() <-chan struct{} {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	return srv.isDoneLocked()
}

func (srv *Server) isDoneLocked() chan struct{} {
	if srv.done == nil {
		srv.done = make(chan struct{})
	}
	return srv.done
}

func (srv *Server) doneLocked() {
	ch := srv.isDoneLocked()
	select {
	case <-ch:
		// Already closed. Don't close again.
	default:
		// Safe to close here. We're the only closer, guarded
		// by s.mu.
		close(ch)
	}
}
