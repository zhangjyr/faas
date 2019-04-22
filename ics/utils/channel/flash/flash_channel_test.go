package flash

import (
	"testing"
	"time"
)

func TestOut(t *testing.T) {
	channel := NewChannel()

	out := 0
	wait := make(chan struct{})
	go func() {
		val := <- channel.Out()
		out = val.(int)
		close(wait)
	}()

	channel.In() <- 1

	<-wait
	channel.Close()
	if out != 1 {
		t.Logf("Wrong out from channel, want: %v, got: %v", 1, out)
		t.Fail()
	}
}

func TestNonBlock(t *testing.T) {
	channel := NewChannel()

	timer := time.NewTimer(100 * time.Millisecond)
	nonblock := false
	go func() {
		channel.In() <- 1
		channel.In() <- 2
		channel.In() <- 3

		nonblock = true
	}()

	<- timer.C
	timer.Stop()
	channel.Close()
	if !nonblock {
		t.Logf("Channel blocked on no listenner")
		t.Fail()
	}
}

func TestOutAfterSkip(t *testing.T) {
	channel := NewChannel()
	channel.In() <- 1

	out := 0
	wait := make(chan struct{})
	go func() {
		val := <- channel.Out()
		out = val.(int)
		close(wait)
	}()

	time.Sleep(100 * time.Millisecond) // Neccessary to avoid out/wait deadlock

	channel.In() <- 2
	<-wait
	channel.Close()
	if out != 2 {
		t.Logf("Wrong out from channel, want: %v, got: %v", 2, out)
		t.Fail()
	}
}
