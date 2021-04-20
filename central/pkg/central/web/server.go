// Package web provides functionality for the central web server, including the front-end interface and backend API.
package web

import (
	_ "embed"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"go.uber.org/zap"
	"wilfrid.me/archive-central/pkg/central/core"
	"wilfrid.me/archive-central/pkg/central/peer"
	"wilfrid.me/archive-central/pkg/central/record"
)

//go:embed pages/store.html
var storeForm []byte

//go:embed pages/register.html
var registerForm []byte

// Server stores the data sources required by the web server.
type Server struct {
	group   peer.Group
	records record.Repository
	logger  *zap.SugaredLogger
}

// NewServer returns a new web server instance.
func NewServer(group peer.Group, records record.Repository, logger *zap.SugaredLogger) *Server {
	return &Server{
		group:   group,
		records: records,
		logger:  logger,
	}
}

// GetRouter returns a chi web router will all web server routes stored.
func (s *Server) GetRouter() *chi.Mux {
	router := chi.NewMux()
	router.Get("/api/page/{id}", s.GetPage)
	router.Get("/api/store", s.StorePage)
	router.Get("/api/register", s.RegisterPeer)
	router.Get("/store", s.staticPage(storeForm))
	router.Get("/register", s.staticPage(registerForm))
	return router
}

// GetPage handles route /api/page/{id} by returning the page of that ID.
func (s *Server) GetPage(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		s.logger.Errorw("failed to parse page id", "msg", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s.logger.Infow("get", "id", id)
	page, err := s.group.Get(r.Context(), id)
	if err != nil {
		s.logger.Errorw("failed to get page", "msg", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s.logger.Infow("writing page", "id", id, "len", len(page))
	w.Header().Set("Content-Type", "text/html")
	if _, err := w.Write(page); err != nil {
		s.logger.Errorw("failed to write page data", "msg", err.Error())
	}
}

// StorePage handles route /api/store.
func (s *Server) StorePage(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	peer := r.URL.Query().Get("peer")
	peerID, err := strconv.Atoi(peer)
	if err != nil {
		s.logger.Errorw("failed to parse peerID", "peer", peerID, "msg", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	s.logger.Infow("store", "url", url, "peer", peerID)

	recID, err := s.group.Store(r.Context(), peerID, url)
	if err != nil {
		s.logger.Errorw("failed to store page", "url", url, "peer", peerID, "msg", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp := []byte(fmt.Sprintf("%d", recID))

	s.logger.Infow("store successful", "url", url, "peer", peerID)
	w.Header().Set("Content-Type", "text/plain")
	if _, err := w.Write(resp); err != nil {
		s.logger.Errorw("failed to write page data", "peer", peerID, "msg", err.Error())
	}
}

// RegisterPeer handles /api/register (the peer registry backend).
func (s *Server) RegisterPeer(w http.ResponseWriter, r *http.Request) {
	address := r.URL.Query().Get("address")
	s.logger.Infow("register", "address", address)

	recID, err := s.records.AddPeer(r.Context(), core.Peer{
		Address: address,
	})
	if err != nil {
		s.logger.Errorw("failed to register peer", "address", address, "msg", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp := []byte(fmt.Sprintf("%d", recID))

	w.Header().Set("Content-Type", "text/plain")
	if _, err := w.Write(resp); err != nil {
		s.logger.Errorw("failed to write register response", "msg", err.Error())
	}
}

// staticPage handles all pages which are just unchanging HTML content.
func (s *Server) staticPage(page []byte) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write(page); err != nil {
			s.logger.Errorw("failed to write static page", "msg", err.Error())
		}
	}
}
