package proxy

import (
	"fmt"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/openfaas/faas/ics/logger"
	"github.com/openfaas/faas/ics/utils/channel"
	"github.com/openfaas/faas/ics/utils/channel/flash"
)

const REMOTE_TOTAL int = 2

type Server struct {
	Addr           string // TCP address to listen on
	Verbose        bool
	Debug          bool
	ServingFeed    <-chan interface{}
	ServedFeed     <-chan interface{}
	Throttle       chan bool

	mu             sync.RWMutex
	log            logger.ILogger
	remoteids      [REMOTE_TOTAL]int
	remotes        [REMOTE_TOTAL]*net.TCPAddr
	listener       *net.TCPListener
	activeConn     map[*forwardConnection]struct{}
	connid         uint64
	listening      chan struct{}
	done           chan struct{}
	servingFeed    channel.Channel
	servedFeed     channel.Channel
	remotePrimary  int
	remoteSecondary int

	started        time.Time      // Time started listening
	requested      int32          // Number of incoming requests.
	served         int32          // Number of served requests.
	serving        int32          // Accurate serving requests.
	sumResponse    int64          // Accumualated response time.
	// usage          uint64       // Accumulated serve time in nanoseconds.
	// updated        uint64       // Last updated duration from started.
}

type Stats struct {
	Requested      int32
	Served         int32
	Time           time.Time
}

// ErrServerClosed is returned by the Server after a call to Shutdown or Close.
var ErrServerClosed = errors.New("proxy: Server closed")

// ErrServerListening is returned by the Server if server is listening.
var ErrServerListening = errors.New("proxy: Server is listening")

func NewServer(port int, debug bool) *Server{
	srv := &Server{
		Addr:       fmt.Sprintf(":%d", port),
		Verbose:    false,
		Debug:      debug,
	}
	if (debug) {
		srv.log = &logger.ColorLogger{
			Verbose:     srv.Verbose,
			Level:       srv.getLoggerLevel(),
			Prefix:      "Proxy server ",
			Color:       true,
		}
	} else {
		srv.log = logger.NilLogger
	}
	srv.activeConn = make(map[*forwardConnection]struct{})
	srv.done = make(chan struct{})
	srv.servingFeed = flash.NewChannel()
	srv.ServingFeed = srv.servingFeed.Out()
	srv.servedFeed = flash.NewChannel()
	srv.ServedFeed = srv.servedFeed.Out()
	srv.Throttle = make(chan bool, 10)

	return srv
}

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
	srv.remotePrimary = 0
	srv.remoteSecondary = 1
	srv.listener, err = net.ListenTCP("tcp", laddr)
	if err == nil {
		srv.log.Info("ICS Listening on %s", srv.Addr)
	}
	srv.started = time.Now()
	return srv.listener, err
}

func (srv *Server) ListenAndProxy(id int, remoteAddr string, onProxy func(int)) error {
	// Start server
	listener, err := srv.Listen()
	if err != nil {
		return err
	}
	defer srv.Close()

	// Override remote address
	err = srv.setRemoteAddr(id, remoteAddr)
	if err != nil {
		return err
	}

	if onProxy != nil {
		go onProxy(id)
	}

	// throttling := false
	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			select {
			case <-srv.isDone():
				return ErrServerClosed
			default:
			}
			srv.log.Error("Failed to accept connection '%s'\n", err)
			continue
		}
		srv.connid++
		srv.log.Info("Connection accepted: %03d", srv.connid)

		go func(conn *net.TCPConn, connid uint64) {
			defer conn.Close()

			// select {
			// case <-srv.Throttle:
			// 	return
			// // case throttle := <-srv.Throttle:
			// // 	if throttle {
			// // 		buffconn.Close()
			// // 	}
			// // 	if throttling != throttle {
			// // 		if throttle {
			// // 			srv.log.Debug("start throttling...")
			// // 		} else {
			// // 			srv.log.Debug("end throttling.")
			// // 		}
			// // 		// throttling = throttle
			// // 	}
			// // 	continue
			// default:
			// 	// if throttling {
			// 	// 	srv.log.Debug("close connection.")
			// 	// 	buffconn.Close()
			// 	// 	continue
			// 	// }
			// }



			var fconn *forwardConnection
			_, addrs := srv.getRemoteAddr()
			fconn = newForwardConnection(conn, listener.Addr().(*net.TCPAddr), addrs)
			fconn.Debug = srv.Debug
			fconn.Nagles = true
			fconn.log = &logger.ColorLogger{
				Verbose:     srv.Verbose,
				Level:       srv.getLoggerLevel(),
				Prefix:      fmt.Sprintf("Connection #%03d ", connid),
				Color:       true,
			}
			fconn.Matcher = srv.packageMatcher

			fconn.forward(srv)
		}(conn, srv.connid)
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

	if !srv.doneLocked() {
		srv.servingFeed.Close()

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

	return nil
}

func (srv *Server) Stats() *Stats {
	return srv.RequestStats()
}

func (srv *Server) RequestStats() *Stats {
	return &Stats{
		Served: atomic.LoadInt32(&srv.served),
		Requested: atomic.LoadInt32(&srv.requested),
		Time: time.Now(),
	}
}

func (srv *Server) getLoggerLevel() int {
	logLevel := logger.LOG_LEVEL_INFO
	if srv.Debug {
		logLevel = logger.LOG_LEVEL_ALL
	}
	return logLevel
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
	srv.mu.Lock()
	defer srv.mu.Unlock()

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

func (srv *Server) doneLocked() bool {
	ch := srv.isDoneLocked()
	select {
	case <-ch:
		// Already closed. Don't close again.
		return true
	default:
		// Safe to close here. We're the only closer, guarded
		// by s.mu.
		close(ch)
		return false
	}
}

func (srv *Server) packageMatcher(fconn *forwardConnection, inbound bool, b []byte) {
	method := string(b[:4])
	switch method {
	case "HTTP":
		// Response
		srv.onServe(fconn)
	case "GET ":
		// Reqeust
		fallthrough
	case "POST":
		fallthrough
	case "PUT ":
		fallthrough
	case "DELE":
		srv.onRequest(fconn)
	}
}

func (srv *Server) onRequest(fconn *forwardConnection) int32 {
	requested := atomic.AddInt32(&srv.requested, 1)
	srv.servingFeed.In() <- requested
	// fconn.markRequest("")
	return requested
}

func (srv *Server) onServe(fconn *forwardConnection) int32 {
	served := atomic.AddInt32(&srv.served, 1)
	// srv.servedFeed.In() <- fconn.markResponse("")
	return served
}
