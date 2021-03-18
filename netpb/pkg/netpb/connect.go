package netpb

import (
	"context"
	"encoding/binary"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"io"
	"net"
)

type Connection struct {
	Send chan proto.Message
	Rec  chan proto.Message
	Errs chan error
}

func (c *Connection) Close() {
	c.Send <- nil
}

func Connect(ctx context.Context, addr string) (Connection, error) {
	dail := net.Dialer{}
	conn, err := dail.DialContext(ctx, "tcp", addr)
	if err != nil {
		return Connection{}, errors.Wrap(err, "dialing address")
	}
	return handle(ctx, conn)
}

func handle(ctx context.Context, conn net.Conn) (Connection, error) {
	peer := Connection{
		Send: make(chan proto.Message),
		Rec:  make(chan proto.Message),
		Errs: make(chan error),
	}
	go func() {
		defer func() {
			if err := conn.Close(); err != nil {
				peer.Errs <- err
			}
		}()
		for {
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
	loop:
		for {
			select {
			case msg := <-peer.Send:
				if msg == nil {
					close(peer.Rec)
					close(peer.Errs)
					break loop
				}
				if err := Write(conn, msg); err != nil {
					peer.Errs <- err
				}
			case <-ctx.Done():
				close(peer.Rec)
				close(peer.Errs)
				break loop
			}
		}
	}()

	return peer, nil
}
