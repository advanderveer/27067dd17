package ripple

import (
	"crypto/sha256"
	"fmt"
	"io"
	"strings"
)

type TxID [sha256.Size]byte

type Msg struct {
	P *ProposalMsg
}

func (msg *Msg) String() string {
	switch true {
	case msg.P != nil:
		txs := []string{}
		for vote := range msg.P.Votes {
			txs = append(txs, fmt.Sprintf("%.3x", vote))
		}

		return fmt.Sprintf("Msg{P: %s}", strings.Join(txs, ","))
	default:
		return "Msg{}"
	}
}

type ProposalMsg struct {
	Votes TxSet
}

//TxSet represents a set of transactions
type TxSet map[TxID]struct{}

//Txs creates a new set of transactions of a
func Txs(txs ...[]byte) (s TxSet) {
	s = make(TxSet)
	for _, tx := range txs {
		s[sha256.Sum256(tx)] = struct{}{}
	}

	return
}

// Round handles messages for a certain round
type Round struct {
	candidates TxSet
	ballot     map[TxID]uint64
	broadcast  Broadcast
}

// NewRound sets up a new round
func NewRound(bc Broadcast, txs TxSet) (r *Round) {
	r = &Round{broadcast: bc, candidates: txs, ballot: map[TxID]uint64{}}

	return
}

// Start handling proposals
func (r *Round) Start() {

	//write our candidate as votes to the broadcast right awawy
	err := r.broadcast.Write(&Msg{P: &ProposalMsg{Votes: r.candidates}})
	if err != nil {
		panic("failed to write initial candidates: " + err.Error())
	}

	go func() {
		for {
			curr := &Msg{}

			err := r.broadcast.Read(curr)
			if err == io.EOF {
				panic("broadcast closed")
			} else if err != nil {
				panic("failed to read broadcast: " + err.Error())
			}

			err = r.Handle(curr)
			if err != nil {
				panic("failed to handle broadcast messsage: " + err.Error())
			}
		}
	}()
}

// Handle a single message in the round
func (r *Round) Handle(msg *Msg) (err error) {
	if msg.P == nil {
		return nil //no proposal in message
	}

	//1) verify that the proposer was allowed to propose:
	//  are the proposals shipped with this valid proposals
	//  does merging those proposals result in a higher threshold
	//  is his draw from that high enough

	// if applying the proposal increases 0

	//2) merge proposals, check if a threshold has been reached
	for vote := range msg.P.Votes {
		r.ballot[vote]++
	}

	//3) if so, package up the proposals that got us to the threshold
	// and formulate a vrf. use VRF to check if we are allowed to propose

	//4) if so send out a proposal with our proof

	return nil
}
