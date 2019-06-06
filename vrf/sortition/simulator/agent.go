package simulator

type BlockReceiver interface {
	ReceiveBlock(b *Block) (err error)
}

type Agent struct {
	incoming chan *Block
}

func NewAgent() (a *Agent) {
	a = &Agent{incoming: make(chan *Block)}

	go func() {
		for inb := range a.incoming {

			//@TODO if already seen a certain block: discard
			//@TODO if a specific user already sent a block at this theight: discard
			//@TODO if a block with a better prio was already seen at this height: discard
			//@TODO verify block prio, if not valid: discard
			//@TODO verify proof of work, if not valid: discard

			//@TODO if previous block doesn't exist. Store out-of-order

			//else: call receive on three random other agents
			_ = inb

		}
	}()

	return
}

// ReceiveBlock will asynchronously ingest the given block. It may it is not
// guaranteed to be accepted
func (a *Agent) ReceiveBlock(b *Block) (err error) {
	a.incoming <- b
	return
}

// DiscoverAgent will allow this agent to send blocks to another agent
func (a *Agent) DiscoverAgent(br BlockReceiver) {
	return
}
