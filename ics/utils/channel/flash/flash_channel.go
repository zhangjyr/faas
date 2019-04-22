package flash

type channel struct {
	in      chan interface{}
	out     chan interface{}
}

func NewChannel() *channel {
	c := &channel{
		in: make(chan interface{}),
		out: make(chan interface{}),
	}
	go c.channel()
	return c
}

func (c *channel) channel() {
	for i := range c.in {
		select {
		case c.out <- i:
		default:
			// if out is not consumed, consume it and move on
		}
	}
	close(c.out)
}

func (c *channel) In() chan<- interface{} {
	return c.in
}

func (c *channel) Out() <-chan interface{} {
	return c.out
}

func (c *channel) Close() {
	close(c.in)
}
