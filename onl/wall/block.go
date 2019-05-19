package wall

import "crypto/sha256"

//IDLen is the id length
const IDLen = sha256.Size

//NilID is an empty id
var NilID = BID{}

//BID is the ID of a block
type BID [IDLen]byte
