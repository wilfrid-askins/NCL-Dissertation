package netpb

import (
	"context"
	"encoding/binary"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io"
	"net"
	"sync"
)

const (
	EventPacket = iota
	EventError
)

type Connection struct {
	conn     net.Conn
	Rec      chan Event
	closed   bool
	closedMx sync.Mutex
	logger   *zap.SugaredLogger
}

type Event struct {
	Type   int
	Packet proto.Message
	Error  error
}

func Connect(ctx context.Context, addr string, logger *zap.SugaredLogger) (Connection, error) {
	dial := net.Dialer{}
	conn, err := dial.DialContext(ctx, "tcp", addr)
	if err != nil {
		return Connection{}, errors.Wrapf(err, "dialing address %s", addr)
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
	c.closedMx.Lock()
	defer c.closedMx.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	//if err := c.conn.Close(); err != nil {
	//	c.logger.Errorw("closing connection", "msg", err.Error())
	//}
	_ = c.conn.Close()
}

func handle(ctx context.Context, conn net.Conn, logger *zap.SugaredLogger) (Connection, error) {
	peer := Connection{
		conn:   conn,
		Rec:    make(chan Event),
		logger: logger,
	}
	go func() {
		for {
			lenB := make([]byte, 8)
			if _, err := conn.Read(lenB); err != nil {
				if err == io.EOF {
					peer.Close()
					break
				}
				peer.Rec <- Event{
					Type:  EventError,
					Error: errors.Wrap(err, "reading length bytes"),
				}
				continue
			}
			length, _ := binary.Uvarint(lenB)
			payload := make([]byte, length)
			if _, err := conn.Read(payload); err != nil {
				peer.Rec <- Event{
					Type:  EventError,
					Error: errors.Wrap(err, "reading payload bytes"),
				}
				continue
			}
			pbp := Payload{}
			if err := proto.Unmarshal(payload, &pbp); err != nil {
				peer.Rec <- Event{
					Type:  EventError,
					Error: errors.Wrap(err, "unmarshalling payload message"),
				}
				continue
			}
			realMsg, err := unwrap(&pbp)
			if err != nil {
				peer.Rec <- Event{
					Type:  EventError,
					Error: errors.Wrap(err, "unwrapping proto"),
				}
				continue
			}
			peer.Rec <- Event{
				Type:   EventPacket,
				Packet: realMsg,
			}
		}
		close(peer.Rec)
	}()

	go func() {
		<-ctx.Done()
		peer.Close()
	}()

	return peer, nil
}
