package channel

type Out interface {
	Out() <-chan interface{}

	Pipe(chan<- interface{})

	StopPipe()
}

type In interface {
	In() chan<- interface{}
}

type Channel interface {
	In

	Out

	Close()
}
