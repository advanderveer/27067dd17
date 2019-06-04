package slot

import "encoding/hex"

//PKString can be configured before initialization to turn logging of primary keys
//into readable names for logging and debugging purposes. By default it will just
//hex encode the first 6 bytes and panic if the pk is shorter
var PKString = DefaultPKString

//DefaultPKString is not used but can be used to reset PKString
var DefaultPKString = func(pk []byte) string {
	return hex.EncodeToString(pk[:4])
}

//BlockName is used for debugging to identify certain blocks
var BlockName = DefaultBlockName

//DefaultBlockName is not used but can be used to reset PKString
var DefaultBlockName = func(id ID) string {
	return hex.EncodeToString(id[:4])
}
