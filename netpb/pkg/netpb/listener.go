package netpb

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"net"
	"strconv"
)

type Listener struct {
	Events *Emitter
	port int
	logger *zap.SugaredLogger
}

type PacketEvent struct {
	Conn net.Conn
	Message proto.Message
}

func NewListener(port int, logger *zap.SugaredLogger) *Listener {
	return &Listener{
		Events: NewEmitter(),
		port: port,
		logger: logger,
	}
}

func (l Listener) Listen() error {
	listener, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(l.port))
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
