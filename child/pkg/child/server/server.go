// Package server is used to serve HTTP endpoints providing the storage of files for the child server.
package server

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"wilfrid.me/archive-child/pkg/child/file"
)

// Server stores the data of the web server.
type Server struct {
	addr     string
	files    *file.Repository
	httpServ http.Server
	logger   *zap.SugaredLogger
}

// New creates an instance of the web server.
func New(port int, files *file.Repository, logger *zap.SugaredLogger) *Server {
	s := &Server{
		addr:   fmt.Sprintf("0.0.0.0:%d", port),
		files:  files,
		logger: logger,
	}
	router := chi.NewMux()
	router.Put("/page/{id}", s.storePage)
	router.Get("/page/{id}", s.getPage)
	router.Get("/health", s.getHealth)
	s.httpServ = http.Server{
		Addr:    s.addr,
		Handler: router,
	}
	return s
}

// Listen will listen and serve on the required endpoints till shutdown is called.
func (s *Server) Listen() error {
	err := s.httpServ.ListenAndServe()
	if err != nil {
		return errors.Wrapf(err, "listening on %s", s.addr)
	}
	return nil
}

// storePage is used to handle storage requests.
func (s *Server) storePage(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		s.logger.Errorw("reading store payload", "msg", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	idStr := chi.URLParam(r, "id")
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

// getPage handles the retrieval of stored pages.
func (s *Server) getPage(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
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

// getHealth returns a report on the status of this peer.
func (s *Server) getHealth(w http.ResponseWriter, r *http.Request) {
	if _, err := w.Write([]byte("ok")); err != nil {
		s.logger.Errorw("writing response", "msg", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// Shutdown will end the serving of the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServ.Shutdown(ctx)
}
