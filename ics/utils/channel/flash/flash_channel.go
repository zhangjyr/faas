package flash

type channel struct {
	in      chan interface{}
	out     chan interface{}
	pipe    chan<- interface{}	// Being initialized with out, it can be overrided.
}

func NewChannel() *channel {
	c := &channel{
		in: make(chan interface{}),
		out: make(chan interface{}),
	}
	go c.channel()
	return c
}

func NewBufferChannel(size int) *channel {
	c := &channel{
		in: make(chan interface{}),
		out: make(chan interface{}, size),
	}
	go c.channel()
	return c
}

func (c *channel) In() chan<- interface{} {
	return c.in
}

func (c *channel) Out() <-chan interface{} {
	return c.out
}

func (c *channel) Pipe(pipeout chan<- interface{}) {
	c.pipe = pipeout
}

func (c *channel) StopPipe() {
	c.pipe = c.out
}

func (c *channel) Close() {
	close(c.in)
}

func (c *channel) channel() {
	c.pipe = c.out
	for i := range c.in {
		select {
		case c.pipe <- i:
		default:
			// if out is not consumed, consume it and move on
		}
	}
	close(c.out)
}
