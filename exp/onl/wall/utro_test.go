package wall

import (
	"crypto/rand"
	"testing"

	"github.com/advanderveer/go-test"
)

func TestUTROCRUD(t *testing.T) {
	id1 := NewIdentity([]byte{0x01}, rand.Reader)
	utro := NewUTRO()
	tr1 := NewTr().Send(100, id1, 0, false).Sign(id1)

	//shouldn't exist yet
	_, ok := utro.Get(Ref(tr1.ID, 0))
	test.Equals(t, false, ok)

	//put in
	utro.Put(Ref(tr1.ID, 0), tr1.Outputs[0])

	_, ok = utro.Get(Ref(tr1.ID, 0))
	test.Equals(t, true, ok)

	//delete
	utro.Del(Ref(tr1.ID, 0))

	_, ok = utro.Get(Ref(tr1.ID, 0))
	test.Equals(t, false, ok)
}

func TestTotalDeposited(t *testing.T) {
	depositTTL := uint64(300)

	id1 := NewIdentity([]byte{0x01}, rand.Reader)
	tr1 := NewTr().
		Send(10, id1, 200, false). //invalid: not a deposit
		Send(15, id1, 300, true).  //valid
		Send(27, id1, 299, true).  //valid
		Send(30, id1, 100, true).  //invalid: unlocks on round 100
		Send(40, id1, 99, true).   //invalid: unlocks before 100
		Send(15, id1, 401, true).  //invalid: locks too long
		Sign(id1)

	utro := NewUTRO()
	utro.Put(Ref(tr1.ID, 0), tr1.Outputs[0])
	utro.Put(Ref(tr1.ID, 1), tr1.Outputs[1])
	utro.Put(Ref(tr1.ID, 2), tr1.Outputs[2])
	utro.Put(Ref(tr1.ID, 3), tr1.Outputs[3])
	utro.Put(Ref(tr1.ID, 4), tr1.Outputs[4])
	utro.Put(Ref(tr1.ID, 5), tr1.Outputs[5])

	total := utro.Deposited(100, depositTTL)
	test.Equals(t, uint64(42), total)
}
