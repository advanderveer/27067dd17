package wall

// State provides and interface to the application state as encoded in
// one specific ancestory of a block
type State interface {
	ReadTr(id TrID) (tr *Tr, err error)
	CurrRound() (r uint64)
	HasBeenSpend(outtr TrID, outi uint64) (ok bool)
}
