package server

import (
	"context"
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
	addr     string
	files    *file.Repository
	httpServ http.Server
	logger   *zap.SugaredLogger
}

func New(port int, files *file.Repository, logger *zap.SugaredLogger) *Server {
	s := &Server{
		addr:   fmt.Sprintf("0.0.0.0:%d", port),
		files:  files,
		logger: logger,
	}
	router := chi.NewMux()
	router.Post("/store", s.storePage)
	router.Get("/get", s.getPage)
	s.httpServ = http.Server{
		Addr:    s.addr,
		Handler: router,
	}
	return s
}

func (s *Server) Listen() error {
	err := s.httpServ.ListenAndServe()
	if err != nil {
		return errors.Wrapf(err, "listening on %s", s.addr)
	}
	return nil
}

func (s *Server) storePage(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		s.logger.Errorw("reading store payload", "msg", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	idStr := r.Header.Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		s.logger.Errorw("parsing file id", "msg", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := s.files.Store(id, data); err != nil {
		s.logger.Errorw("storing file", "id", id, "msg", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, err := w.Write([]byte("ok")); err != nil {
		s.logger.Errorw("writing success response", "id", id, "msg", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (s *Server) getPage(w http.ResponseWriter, r *http.Request) {
	idStr := r.Header.Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		s.logger.Errorw("parsing file id", "msg", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}
	data, err := s.files.Get(id)
	if err != nil {
		s.logger.Errorw("finding file", "id", id, "msg", err.Error())
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if _, err := w.Write(data); err != nil {
		s.logger.Errorw("writing file", "id", id, "msg", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServ.Shutdown(ctx)
}
