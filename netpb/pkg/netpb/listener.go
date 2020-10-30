package netpb

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"net"
)

type Listener struct {
	Events *Emitter
	logger *zap.SugaredLogger
}

type PacketEvent struct {
	Conn net.Conn
	Message proto.Message
}

func NewListener(logger *zap.SugaredLogger) *Listener {
	return &Listener{
		Events: NewEmitter(),
		logger: logger,
	}
}

func (l Listener) Listen() error {
	listener, err := net.Listen("tcp", "127.0.0.1:25565")
	if err != nil {
		return errors.Wrap(err, "failed to handle on port")
	}

	// handle
	for {
		// accept handler
		conn, err := listener.Accept()
		l.logger.Info("new connection")
		if err != nil {
			return errors.Wrap(err, "failed to accept netpb")
		}
		go func() {
			err := handle(conn, l.logger, l.Events)
			if err != nil {
				l.logger.Errorw("failed to handle on connection", "msg", err.Error())
			}
		}()
	}
}
