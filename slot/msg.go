package slot

//MsgType tags messages
type MsgType uint

const (
	//MsgTypeUnkown is a message that is unkown
	MsgTypeUnkown MsgType = iota

	//MsgTypeVote is a block voting message
	MsgTypeVote

	//MsgTypeProposal is a block proposal message
	MsgTypeProposal
)

//Msg holds messages holds passed around between members
type Msg struct {
	Proposal *Block
	Vote     *Vote
}

//Type returns the message type
func (m *Msg) Type() MsgType {
	switch true {
	case m.Proposal != nil:
		return MsgTypeProposal
	case m.Vote != nil:
		return MsgTypeVote
	default:
		return MsgTypeUnkown
	}
}
