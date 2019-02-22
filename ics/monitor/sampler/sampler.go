package sampler

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	ErrNotValidFormat = errors.New("line is not a valid key value format")
	ErrNotEnoughData = errors.New("not enough data")
)

// Sampler will return a sample each time
type Sampler interface {
	Sample(time.Time) (*Sample, error)
}

type Variable interface {
	MakeVariable(float64, time.Time) *Sample
}

type ChannelSampler interface {
	Sample() <-chan *Sample
}

type VariableSampler interface {
	Sampler
	Variable
}

// Sample class
type Sample struct {
	Value     float64
	Time      time.Time
}

func NewSample() *Sample {
	return &Sample{}
}

func (s *Sample) NewDiff(last *Sample) *Sample {
	return &Sample{
		Value: s.Value - last.Value,
		Time: s.Time,
	}
}

func (s *Sample) NewRate(last *Sample) *Sample {
	return &Sample{
		Value: (s.Value - last.Value) / float64(s.Time.Sub(last.Time).Nanoseconds()),
		Time: s.Time,
	}
}

// Saturates negative values at zero and returns a uint64.
// Due to kernel bugs, some of the memory cgroup stats can be negative.
func parseUint(s string, base, bitSize int) (uint64, error) {
	value, err := strconv.ParseUint(s, base, bitSize)
	if err != nil {
		intValue, intErr := strconv.ParseInt(s, base, bitSize)
		// 1. Handle negative values greater than MinInt64 (and)
		// 2. Handle negative values lesser than MinInt64
		if intErr == nil && intValue < 0 {
			return 0, nil
		} else if intErr != nil && intErr.(*strconv.NumError).Err == strconv.ErrRange && intValue < 0 {
			return 0, nil
		}

		return value, err
	}

	return value, nil
}

// Parses a cgroup param and returns as name, value
//  i.e. "io_service_bytes 1234" will return as io_service_bytes, 1234
func getCgroupParamKeyValue(t string) (string, uint64, error) {
	parts := strings.Fields(t)
	switch len(parts) {
	case 2:
		value, err := parseUint(parts[1], 10, 64)
		if err != nil {
			return "", 0, fmt.Errorf("unable to convert param value (%q) to uint64: %v", parts[1], err)
		}

		return parts[0], value, nil
	default:
		return "", 0, ErrNotValidFormat
	}
}

// Gets a single uint64 value from the specified cgroup file.
func getCgroupParamUint(cgroupPath, cgroupFile string) (time.Time, uint64, error) {
	fileName := filepath.Join(cgroupPath, cgroupFile)
	ts, contents, err := ReadFile(fileName)
	if err != nil {
		return ts, 0, err
	}

	res, err := parseUint(strings.TrimSpace(string(contents)), 10, 64)
	if err != nil {
		return ts, res, fmt.Errorf("unable to parse %q as a uint from Cgroup file %q", string(contents), fileName)
	}
	return ts, res, nil
}

// Gets a string value from the specified cgroup file
func getCgroupParamString(cgroupPath, cgroupFile string) (time.Time, string, error) {
	ts, contents, err := ReadFile(filepath.Join(cgroupPath, cgroupFile))
	if err != nil {
		return ts, "", err
	}

	return ts, strings.TrimSpace(string(contents)), nil
}

// ReadFile reads the file named by filename and returns the timestamp and contents.
// A successful call returns err == nil, not err == EOF. Because ReadFile
// reads the whole file, it does not treat an EOF from Read as an error
// to be reported.
func ReadFile(filename string) (time.Time, []byte, error) {
	f, err := os.Open(filename)
	// For some dummy file, content is determinded on opening.
	ts := time.Now()
	if err != nil {
		return ts, nil, err
	}
	defer f.Close()
	// It's a good but not certain bet that FileInfo will tell us exactly how much to
	// read, so let's try it but be prepared for the answer to be wrong.
	var n int64 = bytes.MinRead

	if fi, err := f.Stat(); err == nil {
		// As initial capacity for readAll, use Size + a little extra in case Size
		// is zero, and to avoid another allocation after Read has filled the
		// buffer. The readAll call will read into its allocated internal buffer
		// cheaply. If the size was wrong, we'll either waste some space off the end
		// or reallocate as needed, but in the overwhelmingly common case we'll get
		// it just right.
		if size := fi.Size() + bytes.MinRead; size > n {
			n = size
		}
	}
	data, err := readAll(f, n)
	return ts, data, err
}

// readAll reads from r until an error or EOF and returns the data it read
// from the internal buffer allocated with a specified capacity.
func readAll(r io.Reader, capacity int64) (b []byte, err error) {
	var buf bytes.Buffer
	// If the buffer overflows, we will get bytes.ErrTooLarge.
	// Return that as an error. Any other panic remains.
	defer func() {
		e := recover()
		if e == nil {
			return
		}
		if panicErr, ok := e.(error); ok && panicErr == bytes.ErrTooLarge {
			err = panicErr
		} else {
			panic(e)
		}
	}()
	if int64(int(capacity)) == capacity {
		buf.Grow(int(capacity))
	}
	_, err = buf.ReadFrom(r)
	return buf.Bytes(), err
}
