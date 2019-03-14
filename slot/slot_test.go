package slot_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/advanderveer/27067dd17/slot"
	"github.com/advanderveer/27067dd17/vrf"
	test "github.com/advanderveer/go-test"
)

func TestSyncronousNetwork(t *testing.T) {
	nrounds := uint64(10)
	nmembers := 5
	chains := make([]*slot.Chain, nmembers)
	pks := make([][]byte, nmembers)
	sks := make([]*[vrf.SecretKeySize]byte, nmembers)
	bw := &testbw{}

	//creat member data
	var err error
	for i := range chains {
		chains[i] = slot.NewChain()

		kdata := make([]byte, 33)
		binary.LittleEndian.PutUint64(kdata, uint64(i))
		pks[i], sks[i], err = vrf.GenerateKey(bytes.NewReader(kdata))
		test.Ok(t, err)
	}

	//run rounds
	for round := uint64(1); round < nrounds; round++ {
		var voters []*slot.Voter //just for this round
		tickets := make([]slot.Ticket, nmembers)

		//everyone draws a ticket for this round
		for j, c := range chains {
			// c.Progress(round) //progress to (next) round

			tickets[j], err = c.Draw(pks[j], sks[j], c.Tip(), round)
			test.Ok(t, err)
		}

		//setup voters for everyone that drew that role
		for j, ticket := range tickets {
			if !ticket.Vote {
				continue
			}

			voters = append(voters, slot.NewVoter(round, chains[j], ticket, pks[j]))
		}

		//let all proposers propose blocks to voters
		for j, ticket := range tickets {
			if !ticket.Propose {
				continue
			}

			b := slot.NewBlock(round, chains[j].Tip(), ticket.Data, ticket.Proof, pks[j])
			for _, n := range voters {

				//verify for notarization
				ok, err := n.Verify(b)
				test.Ok(t, err)
				test.Equals(t, true, ok)

				n.Propose(b)
			}
		}

		//append all voted blocks to all chains
		for _, n := range voters {
			votes := n.Vote(bw)
			for _, v := range votes {
				for _, c := range chains {

					//verify for appending
					ok, err := c.Verify(v)
					test.Ok(t, err)
					test.Equals(t, true, ok)

					//append notarized block to chain
					_, _ = c.Append(v.Block)

					//@TODO assert state after appending
				}
			}
		}

	}
}
