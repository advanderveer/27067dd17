package slot

//MsgType tags messages
type MsgType uint

const (
	//MsgTypeUnkown is a message that is unkown
	MsgTypeUnkown MsgType = iota

	//MsgTypeNotarized is a block notarization message
	MsgTypeNotarized

	//MsgTypeProposal is a block proposal message
	MsgTypeProposal
)

//Msg holds messages holds passed around between members
type Msg struct {
	Proposal  *Block
	Notarized *Block
}

//Type returns the message type
func (m *Msg) Type() MsgType {
	switch true {
	case m.Proposal != nil:
		return MsgTypeProposal
	case m.Notarized != nil:
		return MsgTypeNotarized
	default:
		return MsgTypeUnkown
	}
}
