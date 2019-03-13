package slot

type MsgType uint

const (
	//MsgTypeUnkown is a message that is unkown
	MsgTypeUnkown MsgType = iota

	//MsgTypeNotarized is a block notarization message
	MsgTypeNotarized

	//MsgTypePropose is a block proposal message
	MsgTypePropose
)

//Msg holds messages holds passed around between members
type Msg struct {
	Propose   *Block
	Notarized *Block
}

//Type returns the message type
func (m *Msg) Type() MsgType {
	switch true {
	case m.Propose != nil:
		return MsgTypePropose
	case m.Notarized != nil:
		return MsgTypeNotarized
	default:
		return MsgTypeUnkown
	}
}
