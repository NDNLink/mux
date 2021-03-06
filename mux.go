package mux

import (
	"sync"
	"time"

	"github.com/NDNLink/lpm"
	"github.com/NDNLink/ndn"
)

// Mux routes an interest to the handler with the longest matching prefix.
type Mux struct {
	routeMatcher
	mu sync.RWMutex
	Handler
}

// New creates a new mux.
func New() *Mux {
	mux := new(Mux)
	mux.Handler = HandlerFunc(func(w ndn.Sender, i *ndn.Interest) error {
		var h Handler
		mux.mu.RLock()
		val, ok := mux.Match(i.Name.Components)
		if ok {
			h = val
		}
		mux.mu.RUnlock()

		if h != nil {
			return h.ServeNDN(w, i)
		}
		return nil
	})
	return mux
}

// Use adds middleware that will be used when ServeNDN is invoked.
func (mux *Mux) Use(m Middleware) {
	mux.Handler = m(mux.Handler)
}

// Handle adds Handler after additional route-specific middleware is applied.
func (mux *Mux) Handle(name string, h Handler, mw ...Middleware) {
	for _, m := range mw {
		h = m(h)
	}
	mux.mu.Lock()
	mux.routeMatcher.Update(lpm.NewComponents(name), h)
	mux.mu.Unlock()
}

// HandleFunc adds HandlerFunc like Handle.
func (mux *Mux) HandleFunc(name string, h HandlerFunc, mw ...Middleware) {
	mux.Handle(name, h, mw...)
}

// Register registers mux prefixes to nfd.
func (mux *Mux) Register(w ndn.Sender, key ndn.Key) error {
	var names [][]lpm.Component
	mux.mu.Lock()
	mux.routeMatcher.Visit(func(name []lpm.Component, v Handler) (Handler, bool) {
		cpy := make([]lpm.Component, len(name))
		copy(cpy, name)
		names = append(names, cpy)
		return v, false
	})
	mux.mu.Unlock()

	for _, name := range names {
		err := ndn.SendControl(w, "rib", "register", &ndn.Parameters{
			Name: ndn.Name{Components: name},
		}, key)
		if err != nil {
			return err
		}
	}
	return nil
}

// Run invokes Register periodically, and serves each incoming interest in a separate goroutine.
func (mux *Mux) Run(w ndn.Sender, ch <-chan *ndn.Interest, key ndn.Key) {
	mux.Register(w, key)
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case i, ok := <-ch:
			if !ok {
				return
			}
			go mux.ServeNDN(w, i)
		case <-ticker.C:
			go mux.Register(w, key)
		}
	}
}
