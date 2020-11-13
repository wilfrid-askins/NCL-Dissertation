package web

import (
	"github.com/go-chi/chi"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

type (
	PageProvider interface {
		Get(int) ([]byte, error)
		Store(int, string) error
	}

	Server struct {
		pages  PageProvider
		logger *zap.SugaredLogger
	}
)

func NewServer(pages PageProvider, logger *zap.SugaredLogger) *Server {
	return &Server{
		pages:  pages,
		logger: logger,
	}
}

func (s *Server) GetRouter() *chi.Mux {
	router := chi.NewMux()
	router.Get("/page/{id}", s.GetPage)
	return router
}

func (s *Server) GetPage(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		s.logger.Errorw("failed to get page", "msg", err.Error())
		return
	}

	page, err := s.pages.Get(id)
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

func (s *Server) Store(w http.ResponseWriter, r *http.Request) {
	url := chi.URLParam(r, "url")
	peerID := 0

	if err := s.pages.Store(peerID, url); err != nil {
		s.logger.Errorw("failed to store page", "msg", err.Error())
		return
	}
	resp := []byte("true")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if _, err := w.Write(resp); err != nil {
		s.logger.Errorw("failed to write page data", "msg", err.Error())
	}
}
