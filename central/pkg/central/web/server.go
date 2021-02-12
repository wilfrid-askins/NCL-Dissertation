package web

import (
	"fmt"
	"github.com/go-chi/chi"
	"github.com/gobuffalo/packr/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"wilfrid.me/archive-central/pkg/central/peer"
)

type Server struct {
	group        peer.Group
	storeForm    []byte
	registerForm []byte
	logger       *zap.SugaredLogger
}

func NewServer(group peer.Group, logger *zap.SugaredLogger) (*Server, error) {
	htmlPages := packr.New("HTML Box", "./html")
	storeForm, err := htmlPages.FindString("store.html")
	if err != nil {
		return nil, errors.Wrap(err, "loading store page")
	}
	registerForm, err := htmlPages.FindString("register.html")
	if err != nil {
		return nil, errors.Wrap(err, "loading register page")
	}
	return &Server{
		group:        group,
		storeForm:    []byte(storeForm),
		registerForm: []byte(registerForm),
		logger:       logger,
	}, nil
}

func (s *Server) GetRouter() *chi.Mux {
	router := chi.NewMux()
	router.Get("/api/page/{id}", s.GetPage)
	router.Get("/api/store", s.StorePage)
	router.Get("/store", s.staticPage(s.storeForm))
	router.Get("/register", s.staticPage(s.registerForm))
	return router
}

func (s *Server) GetPage(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		s.logger.Errorw("failed to get page", "msg", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	page, err := s.group.Get(r.Context(), id)
	if err != nil {
		s.logger.Errorw("failed to get page", "msg", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
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

	recID, err := s.group.Store(r.Context(), peerID, url)
	if err != nil {
		s.logger.Errorw("failed to store page", "msg", err.Error())
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
