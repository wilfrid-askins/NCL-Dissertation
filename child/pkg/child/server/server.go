package server

import (
	"bytes"
	"compress/gzip"
	"context"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"wilfrid.me/archive-child/pkg/child/file"
	"wilfrid.me/netpb/pkg/netpb"
)

type Server struct {
	port      int
	files     *file.Repository
	transfers *netpb.Transfer
	logger    *zap.SugaredLogger
}

func New(port int, files *file.Repository, logger *zap.SugaredLogger) *Server {
	return &Server{
		port:      port,
		files:     files,
		transfers: netpb.NewTransfer(logger),
		logger:    logger,
	}
}

func (s *Server) Listen(ctx context.Context) error {
	ctx, cfn := context.WithCancel(ctx)
	defer cfn()

	cons, errs, err := netpb.Serve(ctx, s.port, s.logger)
	if err != nil {
		return errors.Wrap(err, "serving on port")
	}

	for {
		select {
		case con := <-cons:
			go s.handle(ctx, con)
		case err := <-errs:
			if err != nil {
				s.logger.Errorw("connection error", "err", err.Error())
			}
		case <-ctx.Done():
			break
		}
	}
}

func (s *Server) handle(ctx context.Context, con netpb.Connection) {
	for {
		select {
		case p := <-con.Rec:
			switch p.(type) {
			case *netpb.PageTransfer:
				trans := p.(*netpb.PageTransfer)
				s.transfers.New(trans)
			case *netpb.PageChunk:
				chunk := p.(*netpb.PageChunk)
				done, data := s.transfers.Push(chunk)
				if !done {
					continue
				}
				if err := s.files.Store(int(chunk.Id), data); err != nil {
					s.logger.Errorw("failed to store page in file system", "msg", err.Error())
				}
			case *netpb.PageRequest:
				req := p.(*netpb.PageRequest)
				data, err := s.files.Get(int(req.Id))
				if err != nil {
					s.logger.Errorw("failed to read file", "msg", err.Error())
					continue
				}

				var comp bytes.Buffer
				zw := gzip.NewWriter(&comp)
				if _, err := zw.Write(data); err != nil {
					s.logger.Errorw("failed to compress file", "msg", err.Error())
					continue
				}
				if err := zw.Close(); err != nil {
					s.logger.Errorw("failed to close compression", "msg", err.Error())
					continue
				}
				if err := netpb.SendPage(req.Id, comp.Bytes(), con.Write); err != nil {
					s.logger.Errorw("failed to send page", "msg", err.Error())
					continue
				}
			default:
				s.logger.Error("unknown packet type")
			}
		case err := <-con.Errs:
			s.logger.Errorw("handler error", "err", err.Error())
		case <-ctx.Done():
			if err := con.Close(); err != nil {
				s.logger.Errorw("closing peer con", "msg", err.Error())
			}
			break
		}
	}
}
