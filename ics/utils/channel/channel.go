package channel

type Channel interface {
	In() chan<- interface{}

	Out() <-chan interface{}

	Close()
}
