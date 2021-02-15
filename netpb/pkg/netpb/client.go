package netpb

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"net"
)

type Client struct {
	addr      string
	conn      net.Conn
	Connected bool
	Events    *Emitter
	logger    *zap.SugaredLogger
}

func NewClient(addr string, logger *zap.SugaredLogger) *Client {
	return &Client{
		addr:   addr,
		Events: NewEmitter(),
		logger: logger,
	}
}

func (c *Client) Connect() error {
	conn, err := net.Dial("tcp", c.addr)
	if err != nil {
		return errors.Wrap(err, "dialing address")
	}
	c.conn = conn
	c.Connected = true
	err = handle(conn, c.logger, c.Events)
	if err != nil {
		c.logger.Errorw("failed to handle on connection", "msg", err.Error())
	}
	return nil
}

func (c *Client) Send(msg proto.Message) error {
	return Write(c.conn, msg)
}
