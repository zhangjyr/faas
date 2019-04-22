package model

import (
	"testing"
	"time"
	"math/rand"
	"runtime"
)

func NewLightStatsN(n int) *LightStats {
	stats := NewLightStats(1)
	for i := 1; i <= n; i++  {
		stats.Add(float64(i))
	}
	return stats
}

func NewLightStatsGON(n int) *LightStats {
	stats := NewLightStats(1)
	for i := 1; i <= n; i++  {
		go stats.Add(float64(i))
	}
	return stats
}

func NewLightStatsCHN(n int, slice int) (*LightStats, chan int, []time.Time) {
	stats := NewLightStats(1)
	ch := make(chan int, n)
	times := make([]time.Time, n + 1)
	for i := 1; i <= slice; i++  {
		go func(i int) {
			<-ch
			j := i
			for ; j <= n; j += slice {
				stats.Add(float64(j))
			}
			times[j - slice] = time.Now()
		}(i)
	}
	return stats, ch, times
}

func NewSumN(n int, start int) Sumer {
	stats := NewSum()
	for i := start; i < n + start; i++  {
		stats.Add(float64(i))
	}
	return stats
}

func NewSumN2(n int, start int) Sumer {
	stats := NewSum()
	for i := start; i < n + start; i++  {
		stats.Add(float64(i) * float64(i))
	}
	return stats
}

func TestValidity(t *testing.T) {
	n := 10
	stats := NewLightStatsN(n)
	time.Sleep(1 * time.Second)
	if stats.N() != int64(n) {
		t.Logf("Wrong n on sequencial adding, want: %v, got: %v", n, stats.N())
		t.Fail()
	}

	sum := float64((n + 1) * n / 2)
	if stats.Sum() != sum {
		t.Logf("Wrong sum on sequencial adding, want: %v, got: %v", sum, stats.Sum())
		t.Fail()
	}

	mean := sum / float64(n)
	if stats.Mean() != mean {
		t.Logf("Wrong mean on sequencial adding, want: %v, got: %v", mean, stats.Mean())
		t.Fail()
	}

	stats = NewLightStatsGON(n)
	time.Sleep(1 * time.Second)
	if stats.N() != int64(n) {
		t.Logf("Wrong n on concurrent adding, want: %v, got: %v", n, stats.N())
		t.Fail()
	}

	sum = float64((n + 1) * n / 2)
	if stats.Sum() != sum {
		t.Logf("Wrong sum on concurrent adding, want: %v, got: %v", sum, stats.Sum())
		t.Fail()
	}

	mean = sum / float64(n)
	if stats.Mean() != mean {
		t.Logf("Wrong mean on concurrent adding, want: %v, got: %v", mean, stats.Mean())
		t.Fail()
	}
}

func TestOverhead(t *testing.T) {
	n := 20
	slice := runtime.NumGoroutine() - 1
	if slice < 1 {
		slice = 1
	}
	t.Logf("Generating stats with %d routines", slice)

	stats, ch, times := NewLightStatsCHN(n, slice)
	timeout, full, failed, blocked := 0, 0, 0, 0
	stats.OnBlock = func() {
		blocked++
	}
	stats.OnSwap = func(reason int, cnt int) {
		if reason == SwapOnFull {
			full++
		} else if cnt > 0 {
			timeout++
		}
	}
	stats.OnFailToSwap = func(reason int) {
		failed++
	}

	start := time.Now()
	go close(ch)
	<-ch
	times[0] = time.Now()
	// for i := 1; i <= n; i++ {
	// 	ch <- i
	// }
	// cost := time.Since(start)
	time.Sleep(1 * time.Second)
	cost := times[0].Sub(start).Seconds()
	t.Logf("Testing overhead (Closing blocked channel): %f s", cost)
	for i := n - slice; i <= n; i++ {
		end := times[i].Sub(start).Seconds()
		if end > cost {
			cost = end
		}
	}
	if stats.N() != int64(n) {
		t.Logf("Wrong n, want: %v, got: %v", n, stats.N())
		t.Fail()
	}

	sum := float64((n + 1) * n / 2)
	if stats.Sum() != sum {
		t.Logf("Wrong sum, want: %v, got: %v", sum, stats.Sum())
		t.Fail()
	}

	mean := sum / float64(n)
	if stats.Mean() != mean {
		t.Logf("Wrong mean, want: %v, got: %v", mean, stats.Mean())
		t.Fail()
	}

	rand.Seed(time.Now().Unix())
	seed := rand.Intn(10000)
	t.Logf("Simple sum start from %d", seed)
	start2 := time.Now()
	sumer := NewSumN(n, seed)
	sumer2 := NewSumN2(n, seed)
	cost2 := time.Since(start2)

	sum = sumer.Sum() + sumer2.Sum()

	t.Logf("Overhead %f s, compare with simple sum of %f s", cost, cost2.Seconds())
	t.Logf("Swap on timeout: %d, on full %d. Failed %d. Blocked %d", timeout, full, failed, blocked)
}
