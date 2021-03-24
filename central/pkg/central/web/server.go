package web

import (
	_ "embed"
	"fmt"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"wilfrid.me/archive-central/pkg/central/peer"
)

//go:embed pages/store.html
var storeForm []byte

//go:embed pages/register.html
var registerForm []byte

type Server struct {
	group  peer.Group
	logger *zap.SugaredLogger
}

func NewServer(group peer.Group, logger *zap.SugaredLogger) *Server {
	return &Server{
		group:  group,
		logger: logger,
	}
}

func (s *Server) GetRouter() *chi.Mux {
	router := chi.NewMux()
	router.Get("/api/page/{id}", s.GetPage)
	router.Get("/api/store", s.StorePage)
	router.Get("/store", s.staticPage(storeForm))
	router.Get("/register", s.staticPage(registerForm))
	return router
}

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

	s.logger.Infow("writing page", "id", id)
	w.Header().Set("Content-Type", "text/pages")
	w.WriteHeader(http.StatusAccepted)
	if _, err := w.Write(page); err != nil {
		s.logger.Errorw("failed to write page data", "msg", err.Error())
	}
}

func (s *Server) StorePage(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	peer := r.URL.Query().Get("peer")
	peerID, err := strconv.Atoi(peer)
	if err != nil {
		s.logger.Errorw("failed to parse peerID", "msg", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	s.logger.Infow("store", "url", url)

	recID, err := s.group.Store(r.Context(), peerID, url)
	if err != nil {
		s.logger.Errorw("failed to store page", "url", url, "msg", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp := []byte(fmt.Sprintf("%d", recID))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if _, err := w.Write(resp); err != nil {
		s.logger.Errorw("failed to write page data", "msg", err.Error())
	}
}

func (s *Server) staticPage(page []byte) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write(page); err != nil {
			s.logger.Errorw("failed to write static page", "msg", err.Error())
		}
	}
}
