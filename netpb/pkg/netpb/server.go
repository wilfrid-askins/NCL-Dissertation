package netpb

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"net"
)

func Serve(ctx context.Context, port int, logger *zap.SugaredLogger) (chan Connection, chan error, error) {
	newCons := make(chan net.Conn)
	cons := make(chan Connection)
	errs := make(chan error)

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to handle on port %d", port)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				close(newCons)
				return
			}
			newCons <- conn
		}
	}()

	go func() {
	loop:
		for {
			select {
			case conn := <-newCons:
				peer, err := handle(ctx, conn, logger)
				if err != nil {
					errs <- errors.Wrap(err, "failed to handle connection")
				}
				cons <- peer
			case <-ctx.Done():
				if err := listener.Close(); err != nil {
					errs <- err
				}
				close(cons)
				close(errs)
				break loop
			}
		}
	}()

	return cons, errs, nil
}
