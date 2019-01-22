/**
 * MIT License
 * Copyright © 2014 Jaime Pillora dev@jpillora.com
 *
 * Permission is hereby granted, free of charge, to any person obtaining a
 * copy of this software and associated documentation files (the 'Software'),
 * to deal in the Software without restriction, including without limitation
 * the rights to use, copy, modify, merge, publish, distribute, sublicense,
 * and/or sell copies of the Software, and to permit persons to whom the
 * Software is furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED 'AS IS', WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
 * FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS
 * IN THE SOFTWARE.
 */

/**
 * Refactor by Tianium jzhang33@gmu.edu
 */

package proxy

import (
	"io"
	"net"
	"sync"

	"github.com/openfaas/faas/watchdog/logger"
)

// forwardConnection - Manages a forwardConnection connection, piping data between local and remote.
type forwardConnection struct {
	sentBytes     uint64
	receivedBytes uint64
	laddr         *net.TCPAddr
	raddrs        []*net.TCPAddr
	lconn         io.ReadWriteCloser
	rconns        []io.ReadWriteCloser
	closed        chan bool
	traceFormat   string
	mu            sync.Mutex

	Matcher  func([]byte)
	Replacer func([]byte) []byte

	// Settings
	Nagles    bool
	Log       logger.ILogger
	Debug     bool
	Binary    bool
}

// New - Create a new forwardConnection instance. Takes over local connection passed in,
// and closes it when finished.
func newForwardConnection(lconn *net.TCPConn, laddr, raddrs []*net.TCPAddr) *forwardConnection {
	return &forwardConnection{
		lconn:  lconn,
		laddr:  laddr,
		raddrs:  raddrs,
		closed: make(chan bool),
		Log:    &logger.NilLogger{},
	}
}

type setNoDelayer interface {
	SetNoDelay(bool) error
}

// Start - open connection to remote and start forwardConnectioning data.
func (fconn *forwardConnection) forward(srv *Server) {
	var err error
	// Connect to remotes
	for i, raddr := range fconn.raddrs {
		fconn.rconns[i], err = net.DialTCP("tcp", nil, raddr)
		if err != nil {
			fconn.Log.Warn("Remote connection failed: %s", err)
			return
		}
		defer fconn.rconns[i].Close()
	}
	srv.trackConn(fconn, true)

	// Nagles?
	if fconn.Nagles {
		if conn, ok := fconn.lconn.(setNoDelayer); ok {
			conn.SetNoDelay(true)
		}
		for i, rconn := range fconn.rconns {
			if conn, ok := rconn.(setNoDelayer); ok {
				conn.SetNoDelay(true)
			}
		}
	}

	// Display both ends
	for i, raddr := range fconn.raddrs {
		fconn.Log.Info("Opened %s >>> %s", fconn.laddr.String(), raddr.String())
	}

	// Reset format for trace
	if fconn.Binary {
		fconn.traceFormat = "%x"
	} else {
		fconn.traceFormat = "%s"
	}

	// Bidirectional copy
	rwriter := fconn.rconns[0];
	rreader := fconn.rconns[0];
	if len(fconns.rconns) > 1 {
		rwriter = io.MultiWriter(fconn.rconns...)
		rreader = io.MultiReader(fconn.rconns...)
	}
	go fconn.pipe(fconn.lconn, rwriter)
	go fconn.pipe(rreader, fconn.lconn)

	// Wait for close...
	<-fconn.closed
	fconn.Log.Info("Closed (%d bytes sent, %d bytes recieved)", fconn.sentBytes, fconn.receivedBytes)
	srv.trackConn(fconn, false)
}

func (fconn *forwardConnection) close() {
	fconn.mu.Lock()
	defer fconn.mu.Unlock()

	select {
	case <-fconn.closed:
		// Already closed. Don't close again.
	default:
		close(fconn.closed)
	}
}

func (fconn *forwardConnection) isClosed() <-chan bool {
	fconn.mu.Lock()
	defer fconn.mu.Unlock()
	return fconn.closed
}

func (fconn *forwardConnection) err(s string, err error) {
	if err != io.EOF {
		fconn.Log.Warn(s, err)
	} else {
		fconn.Log.Debug(s, err)
	}

	fconn.close()
}

func (fconn *forwardConnection) pipe(src io.Reader, dst io.Writer) {
	islocal := src == fconn.lconn

	//directional copy (64k buffer)
	buff := make([]byte, 0xffff)
	for {
		n, readErr := src.Read(buff)
		if readErr != nil {
			select {
			case <-fconn.isClosed():
				// Stop on closing
				return
			default:
			}
			if readErr == io.EOF && n > 0 {
				// Pass down to to transfer rest bytes
			} else if islocal {
				fconn.err("Inbound read failed \"%s\"", readErr)
				return
			} else {
				fconn.err("Outbound read failed \"%s\"", readErr)
				return
			}
		}
		b := buff[:n]

		//execute match
		if fconn.Matcher != nil {
			fconn.Matcher(b)
		}

		//execute replace
		if fconn.Replacer != nil {
			b = fconn.Replacer(b)
		}

		//show output
		fconn.trace(islocal, b, n)

		//write out result
		n, writeErr := dst.Write(b)
		if writeErr != nil {
			if islocal {
				fconn.err("Inbound write failed \"%s\"", writeErr)
			} else {
				fconn.err("Outbound write failed \"%s\"", writeErr)
			}
			return
		}
		if islocal {
			fconn.sentBytes += uint64(n)
		} else {
			fconn.receivedBytes += uint64(n)
		}

		if (readErr != nil) {
			fconn.close()
			return
		}
	}
}

func (fconn *forwardConnection) trace(islocal bool, bytes []byte, len int) {
	if !fconn.Debug {
		return
	}

	if islocal {
		fconn.Log.Debug(">>> %d bytes sent", len)
		fconn.Log.Trace(fconn.traceFormat, bytes)
	} else {
		fconn.Log.Debug("<<< %d bytes recieved", len)
		fconn.Log.Trace(fconn.traceFormat, bytes)
	}
}
