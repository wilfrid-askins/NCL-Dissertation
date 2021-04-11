package server

import (
	"fmt"
	"github.com/go-chi/chi"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"strconv"
	"wilfrid.me/archive-child/pkg/child/file"
)

type Server struct {
	port      int
	files     *file.Repository
	logger    *zap.SugaredLogger
}

func New(port int, files *file.Repository, logger *zap.SugaredLogger) *Server {
	return &Server{
		port:      port,
		files:     files,
		logger:    logger,
	}
}

func (s *Server) Listen() error {
	router := chi.NewMux()
	router.Post("/store", s.storePage)
	router.Get("/get", s.getPage)

	addr := fmt.Sprintf("0.0.0.0:%d", s.port)
	err := http.ListenAndServe(addr, router)
	if err != nil {
		return errors.Wrapf(err, "listening on %s", addr)
	}
	return nil
}

func (s *Server) storePage(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		s.logger.Errorw("reading store payload", "msg", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		s.logger.Errorw("parsing file id", "msg", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}
	if err := s.files.Store(id, data); err != nil {
		s.logger.Errorw("parsing file id", "msg", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}
	if _, err := w.Write([]byte("ok")); err != nil {
		s.logger.Errorw("writing success response", "msg", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) getPage(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		s.logger.Errorw("parsing file id", "msg", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}

	data, err := s.files.Get(id)
	if err != nil {
		s.logger.Errorw("finding file", "msg", err.Error())
		w.WriteHeader(http.StatusNotFound)
	}

	if _, err := w.Write(data); err != nil {
		s.logger.Errorw("writing file", "msg", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}
}
