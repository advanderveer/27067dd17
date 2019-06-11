package bcast

import (
	"encoding/gob"
	"io"
	"net/http"
	"sync"
)

type multiWriter struct {
	writers []io.WriteCloser
	mu      sync.RWMutex
}

func (t *multiWriter) Close() (err error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, w := range t.writers {
		err = w.Close()
	}

	return
}

func (t *multiWriter) Add(w io.WriteCloser) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.writers = append(t.writers, w)
}

// @TODO how to deal with error handling, it now breaks of writing to the
// rest of the connected clients if one fails. thats not what we want
func (t *multiWriter) Write(p []byte) (n int, err error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, w := range t.writers {
		n, err = w.Write(p)
		if err != nil {
			return
		}
		if n != len(p) {
			err = io.ErrShortWrite
			return
		}
	}

	return len(p), nil
}

// Handler handles incoming http requests
type Handler struct {
	mw  *multiWriter
	rc  io.ReadCloser
	wc  io.WriteCloser
	enc *gob.Encoder

	ingress chan *Msg
}

// NewHandler inits a new handler
func NewHandler() (h *Handler) {
	h = &Handler{
		mw:      &multiWriter{},
		ingress: make(chan *Msg),
	}

	h.enc = gob.NewEncoder(h.mw)
	h.rc, h.wc = io.Pipe()
	return
}

func (h *Handler) Write(msg *Msg) (err error) {
	return h.enc.Encode(msg)
}

func (h *Handler) Read() (msg *Msg, err error) {
	msg = <-h.ingress
	return
}

// Close the open communcation channels with the peer
func (h *Handler) Close() (err error) {
	close(h.ingress)

	err = h.mw.Close()
	if err != nil {
		return err
	}

	err = h.wc.Close()
	if err != nil {
		return err
	}

	err = h.rc.Close()
	if err != nil {
		return err
	}

	return
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/push":
		h.handlePush(w, r)
	case "/pull":
		h.handlePull(w, r)
	default:
		http.NotFound(w, r)
	}
}

type flushWriter struct{ http.ResponseWriter }

func (fw *flushWriter) Write(p []byte) (n int, err error) {
	n, err = fw.ResponseWriter.Write(p)
	fw.ResponseWriter.(http.Flusher).Flush()
	return
}

func (h *Handler) handlePull(w http.ResponseWriter, r *http.Request) {
	left := w.(http.CloseNotifier).CloseNotify()
	// tick := time.NewTicker(1 * time.Second)
	// defer tick.Stop()
	_ = left //@TODO handle client leaving early

	pr, pw := io.Pipe()

	//@TODO how does flush work?
	w.(http.Flusher).Flush()

	h.mw.Add(pw)

	_, err := io.Copy(&flushWriter{w}, pr)
	if err != nil {
		panic(err) //@TODO handle error
	}
}

func (h *Handler) handlePush(w http.ResponseWriter, r *http.Request) {
	dec := gob.NewDecoder(r.Body)

	msg := new(Msg)
	err := dec.Decode(msg)
	if err != nil {
		panic(err) //@TODO handle errors
	}

	h.ingress <- msg
	_ = msg
}
