package bcasthttp

import (
	"encoding/gob"
	"io"
	"log"
	"net/http"
	"sync"
	"sync/atomic"

	bcast "github.com/advanderveer/27067dd17/exp/btree/bcast"
)

// Handler handles http requests
type Handler struct {
	logs   *log.Logger
	closed uint32

	// push side
	reads chan *bcast.Msg
	wg    sync.WaitGroup
}

// NewHandler creates the request handler
func NewHandler(logw io.Writer, readn int) (h *Handler) {
	h = &Handler{
		logs:  log.New(logw, "bcast/handler: ", 0),
		reads: make(chan *bcast.Msg, readn),
	}

	return
}

// Read the next message that was send to the server
func (h *Handler) Read() (msg *bcast.Msg, err error) {
	msg = <-h.reads
	if msg == nil {
		return nil, io.EOF
	}

	return
}

func (h *Handler) handlePush(w http.ResponseWriter, r *http.Request) {
	h.wg.Add(1)
	defer h.wg.Done()

	dec := gob.NewDecoder(r.Body)
	for {
		msg := new(bcast.Msg)
		err := dec.Decode(msg)

		if err == io.EOF {
			return //request ended
		} else if err != nil {
			h.logs.Printf("[DEBUG] failed to decode incoming message data: %v", err)
			http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
			return //decoding fail, stop the connection
		}

		h.reads <- msg
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.TLS == nil || !r.ProtoAtLeast(2, 0) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return //must be at least 2.0 and tls
	}

	if atomic.LoadUint32(&h.closed) > 0 {
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return //closed, service unavailable
	}

	switch r.URL.Path {
	case "/push":
		h.handlePush(w, r)
	default:
		http.NotFound(w, r)
	}
}

// Close the handler
func (h *Handler) Close() (err error) {
	atomic.StoreUint32(&h.closed, 1) //mark as closed
	h.wg.Wait()                      //wait for open pushes
	close(h.reads)                   //close reads
	return nil
}
