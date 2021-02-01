package web

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/gobuffalo/packr/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"wilfrid.me/netpb/pkg/archive"
)

type (
	PageProvider interface {
		Get(context.Context, int) ([]byte, error)
		Store(context.Context, int, string) (int, error)
	}

	DeveloperProvider interface {
		GetLast(context.Context, int) ([]archive.Snapshot, error)
		GetPeers(context.Context) ([]archive.Peer, error)
	}

	Server struct {
		pages        PageProvider
		storeForm    []byte
		registerForm []byte
		logger       *zap.SugaredLogger
	}
)

func NewServer(pages PageProvider, developer DeveloperProvider, logger *zap.SugaredLogger) (*Server, error) {
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
		pages:        pages,
		developer:    developer,
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
		return
	}

	page, err := s.pages.Get(r.Context(), id)
	if err != nil {
		s.logger.Errorw("failed to get page", "msg", err.Error())
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
	peerID := 1 // TODO: set peer id

	recID, err := s.pages.Store(r.Context(), peerID, url)
	if err != nil {
		s.logger.Errorw("failed to store page", "msg", err.Error())
		return
	}
	resp := []byte(fmt.Sprintf("%d", recID))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if _, err := w.Write(resp); err != nil {
		s.logger.Errorw("failed to write page data", "msg", err.Error())
	}
}

func (s *Server) GetReport(w http.ResponseWriter, r *http.Request) {
	type Resp struct {
		Pages []archive.Snapshot `json:"pages"`
		Peers []archive.Peer     `json:"peers"`
	}
	lastNum := r.URL.Query().Get("last")
	lastInt, err := strconv.Atoi(lastNum)
	if err != nil {
		s.logger.Errorw("failed to parse num", "err", err.Error())
		return
	}

	snapshots, err := s.developer.GetLast(r.Context(), lastInt)
	if err != nil {
		s.logger.Errorw("failed to query database", "err", err.Error())
		return
	}

	peers, err := s.developer.GetPeers(r.Context())
	if err != nil {
		s.logger.Errorw("failed to query database", "err", err.Error())
		return
	}

	resp := Resp{Pages: snapshots, Peers: peers}
	respData, err := json.Marshal(resp)
	if err != nil {
		s.logger.Errorw("failed to marshal response", "err", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if _, err := w.Write(respData); err != nil {
		s.logger.Errorw("failed to write response data", "msg", err.Error())
	}
}

func (s *Server) staticPage(page []byte) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write(page); err != nil {
			s.logger.Errorw("failed to write static page", "msg", err.Error())
		}
	}
}
