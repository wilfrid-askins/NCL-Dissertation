package netpb

import (
	"context"
	"encoding/binary"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io"
	"net"
)

type Connection struct {
	conn   net.Conn
	Rec    chan proto.Message
	Errs   chan error
	closed bool
}

func Connect(ctx context.Context, addr string, logger *zap.SugaredLogger) (Connection, error) {
	dail := net.Dialer{}
	conn, err := dail.DialContext(ctx, "tcp", addr)
	if err != nil {
		return Connection{}, errors.Wrap(err, "dialing address")
	}
	return handle(ctx, conn, logger)
}

func (c *Connection) Write(msg proto.Message) error {
	if err := Write(c.conn, msg); err != nil {
		return errors.Wrap(err, "writing msg")
	}
	return nil
}

func (c *Connection) Close() {
	c.closed = true
}

func handle(ctx context.Context, conn net.Conn, logger *zap.SugaredLogger) (Connection, error) {
	peer := Connection{
		conn: conn,
		Rec:  make(chan proto.Message),
		Errs: make(chan error),
	}
	go func() {
		defer func() {
			if err := conn.Close(); err != nil {
				logger.Errorw("closing connection", "msg", err.Error())
			}
			close(peer.Rec)
			close(peer.Errs)
		}()
		for !peer.closed {
			lenB := make([]byte, 8)
			if _, err := conn.Read(lenB); err != nil {
				if err == io.EOF {
					break
				}
				peer.Errs <- errors.Wrap(err, "reading length bytes")
				continue
			}
			length, _ := binary.Uvarint(lenB)
			payload := make([]byte, length)
			if _, err := conn.Read(payload); err != nil {
				peer.Errs <- errors.Wrap(err, "reading payload bytes")
				continue
			}
			pbp := Payload{}
			if err := proto.Unmarshal(payload, &pbp); err != nil {
				peer.Errs <- errors.Wrap(err, "unmarshalling payload message")
				continue
			}
			realMsg, err := unwrap(&pbp)
			if err != nil {
				peer.Errs <- errors.Wrap(err, "unwrapping proto")
				continue
			}
			peer.Rec <- realMsg
		}
	}()

	go func() {
		<-ctx.Done()
		peer.Close()
	}()

	return peer, nil
}
