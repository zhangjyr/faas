package proxy

import (
	"fmt"
	"errors"
	"log"
	"net"
	"sync"

	"github.com/openfaas/faas/watchdog/logger"
)

const REMOTE_TOTAL int = 2

type Server struct {
	Addr           string // TCP address to listen on
	Verbose        bool
	Debug          bool

	mu             sync.RWMutex
	remoteids      [REMOTE_TOTAL]int
	remotes        [REMOTE_TOTAL]*net.TCPAddr
	listener       *net.TCPListener
	activeConn     map[*forwardConnection]struct{}
	connid         uint64
	listening      chan struct{}
	done           chan struct{}
	remotePrimary  int
	remoteSecondary int
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
		return srv.listener, err
	}

	// Variable intialization and listen
	srv.activeConn = make(map[*forwardConnection]struct{})
	srv.done = make(chan struct{})
	srv.remotePrimary = 0
	srv.remoteSecondary = 1
	srv.listener, err = net.ListenTCP("tcp", laddr)
	return srv.listener, err
}

func (srv *Server) ListenAndProxy(id int, remoteAddr string, onProxy func(int)) error {
	// Override remote address
	err := srv.setRemoteAddr(id, remoteAddr)
	if err != nil {
		return err
	}

	// Start server
	listener, err := srv.Listen()
	if err != nil {
		return err
	}
	defer srv.Close()
	if onProxy != nil {
		go onProxy(id)
	}

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
		_, addrs := srv.getRemoteAddr()
		fconn = newForwardConnection(conn, listener.Addr().(*net.TCPAddr), addrs)
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
	srv.mu.RLock()
	defer srv.mu.RUnlock()
	return srv.isListeningLocked()
}

/**
 * Forward request to another address
 */
func (srv *Server) Swap(id int, remoteAddr string) error {
	return srv.setRemoteAddr(id, remoteAddr)
}

/**
 * Add secondary address to forward list.
 */
func (srv *Server) Share(id int, remoteAddr string) error {
	return srv.setShareAddr(id, remoteAddr)
}

/**
 * Stop forwarding request to secondary address.
 */
func (srv *Server) Unshare() {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	srv.remoteids[srv.remoteSecondary] = 0
	srv.remotes[srv.remoteSecondary] = nil
}

/**
 * Stop forwarding request to primary address and promote secondary address to primary.
 */
func (srv *Server) Promote() {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	srv.remoteids[srv.remotePrimary] = 0
	srv.remotes[srv.remotePrimary] = nil
	temp := srv.remotePrimary
	srv.remotePrimary = srv.remoteSecondary
	srv.remoteSecondary = temp
}

func (srv *Server) Primary() int {
	ids, _ := srv.getRemoteAddr()
	return ids[0]
}

func (srv *Server) Secondary() int {
	ids, _ := srv.getRemoteAddr()
	if len(ids) > 1 {
		return ids[1]
	} else {
		return 0
	}
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

func (srv *Server) getRemoteAddr() ([]int, []*net.TCPAddr) {
	srv.mu.RLock()
	defer srv.mu.RUnlock()

	if srv.remotes[srv.remoteSecondary] == nil {
		return []int{srv.remoteids[srv.remotePrimary]}, []*net.TCPAddr{srv.remotes[srv.remotePrimary]}
	} else {
		return []int{srv.remoteids[srv.remotePrimary], srv.remoteids[srv.remoteSecondary]},
			[]*net.TCPAddr{srv.remotes[srv.remotePrimary], srv.remotes[srv.remoteSecondary]}
	}
}

func (srv *Server) setRemoteAddr(id int, remoteAddr string) error {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	raddr, err := net.ResolveTCPAddr("tcp", remoteAddr)
	if err != nil {
		return err
	}

	srv.remoteids[srv.remotePrimary] = id
	srv.remotes[srv.remotePrimary] = raddr
	return nil
}

func (srv *Server) setShareAddr(id int, remoteAddr string) error {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	raddr, err := net.ResolveTCPAddr("tcp", remoteAddr)
	if err != nil {
		return err
	}

	srv.remoteids[srv.remoteSecondary] = id
	srv.remotes[srv.remoteSecondary] = raddr
	return nil
}

func (srv *Server) trackConn(fconn *forwardConnection, add bool) {
	srv.mu.RLock()
	defer srv.mu.RUnlock()

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
	srv.mu.RLock()
	defer srv.mu.RUnlock()
	return srv.isDoneLocked()
}

func (srv *Server) isDoneLocked() chan struct{} {
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
