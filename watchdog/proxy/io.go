package proxy

import (
	"errors"
	"io"
)

var ErrWriteToAny = errors.New("MultiWriter: Fail to write to any writer.")

type multiret struct {
	from   int
    n      int
    err    error
}

type multiReader struct {
	readers   []io.Reader
	readings  []bool
}

func (mr *multiReader) readFrom(idx int, p []byte, done chan multiret) {
	ret := multiret{
		from: idx,
	}
	ret.n, ret.err = mr.readers[idx].Read(p)
	done <-ret
}

// func (mr *multiReader) Read(p []byte) (n int, err error) {
// 	done := make(chan multiret)
// 	for i, _ := range mr.readers {
// 		if mr.readings[i] == false {
// 			go mr.readFrom(i, p, done)
// 			mr.readings[i] = true
// 		}
// 	}
//
// 	ret := <- done
// 	mr.readings[ret.from] = false
// 	return ret.n, ret.err
// }
func (mr *multiReader) Read(p []byte) (n int, err error) {
	if len(mr.readers) == 1 {
		return mr.readers[0].Read(p)
	}

	dones := make([]chan multiret, len(mr.readers))
	dps := make([][]byte, len(mr.readers))
	for i, _ := range mr.readers {
		dp := make([]byte, len(p))
		done := make(chan multiret)
		go mr.readFrom(i, dp, done)
		dones[i] = done
		dps[i] = dp
	}

	// Only one reader may return result, EOF or err for others
	for i, done := range dones{
		ret := <-done
		n, err = ret.n, ret.err
		if err == nil {
			mr.readers = mr.readers[i:i+1]
			copy(p, dps[i])
			break
		}
	}
	// All err
	if len(mr.readers) > 0 {
		copy(p, dps[len(mr.readers) - 1])
	}

	return n, err
}


// MultiReader returns a Reader that read from one of
// the provided input readers. Once any inputs have returned EOF or error,
// Read will return EOR or err
func MultiReader(readers ...io.Reader) io.Reader {
	r := make([]io.Reader, len(readers))
	copy(r, readers)
	return &multiReader{
		readers: r,
		readings: make([]bool, len(readers)),
	}
}

type multiWriter struct {
	writers   []io.Writer
}

func (mw *multiWriter) writeTo(idx int, p []byte, done chan multiret) {
	ret := multiret{
		from: idx,
	}
	ret.n, ret.err = mw.writers[idx].Write(p)
	done <-ret
}

func (mw *multiWriter) Write(p []byte) (n int, err error) {
	if len(mw.writers) == 1 {
		return mw.writers[0].Write(p)
	}

	dones := make([]chan multiret, len(mw.writers))
	for i, _ := range mw.writers {
		done := make(chan multiret)
		go mw.writeTo(i, p, done)
		dones[i] = done
	}

	succeeded := len(mw.writers)
	for i, done := range dones{
		ret := <-done
		// Short write
		if ret.err == nil && ret.n != len(p) {
			ret.err = io.ErrShortWrite
		}

		// Flag error
		if ret.err == nil {
			n, err = ret.n, ret.err
		} else {
			succeeded -= 1
			mw.writers[i] = nil
		}
	}
	if succeeded == 0 {
		return 0, ErrWriteToAny
	} else if succeeded < len(mw.writers) {
		writers := make([]io.Writer, 0, len(mw.writers) - succeeded)
		for _, writer := range mw.writers {
			if writer != nil {
				writers = append(writers, writer)
			}
		}
		mw.writers = writers
	}

	return n, err
}

// MultiWriter creates a writer that duplicates its writes to all the
// provided writers.
//
// Each write is written to each listed writer, in parallel.
// If a listed writer returns an error, only the first error get returned.
func MultiWriter(writers ...io.Writer) io.Writer {
	w := make([]io.Writer, len(writers))
	copy(w, writers)
	return &multiWriter{w}
}
