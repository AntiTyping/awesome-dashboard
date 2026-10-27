package server

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/AntiTyping/awesome-dashboard/internal/store"
)

type broker struct {
	mu      sync.RWMutex
	clients map[chan struct{}]struct{}
}

func newBroker() *broker {
	return &broker{clients: make(map[chan struct{}]struct{})}
}

func (b *broker) subscribe() chan struct{} {
	ch := make(chan struct{}, 1)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *broker) unsubscribe(ch chan struct{}) {
	b.mu.Lock()
	delete(b.clients, ch)
	b.mu.Unlock()
}

// broadcast never blocks: a client with a pending event already has a refresh queued.
func (b *broker) broadcast() {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

type Server struct {
	store  *store.Store
	broker *broker
}

func New(st *store.Store) *Server {
	return &Server{store: st, broker: newBroker()}
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/items", s.handleItems)
	mux.HandleFunc("GET /api/events", s.handleEvents)
	mux.HandleFunc("POST /api/notify", s.handleNotify)
}

func (s *Server) handleItems(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.GetAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	// Subscribe before flushing headers so a client that sees the response is already registered.
	ch := s.broker.subscribe()
	defer s.broker.unsubscribe(ch)

	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	flusher.Flush()

	for {
		select {
		case <-ch:
			if _, err := w.Write([]byte("data: refresh\n\n")); err != nil {
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) handleNotify(w http.ResponseWriter, r *http.Request) {
	s.broker.broadcast()
	w.WriteHeader(http.StatusNoContent)
}
