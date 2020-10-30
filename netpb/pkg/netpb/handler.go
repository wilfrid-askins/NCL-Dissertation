package netpb

import
(
	"encoding/binary"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io"
	"net"
)

type Output func(Payload)

func handle(conn net.Conn, logger *zap.SugaredLogger, events *Emitter) error {
	defer func() {
		if err := conn.Close(); err != nil {
			logger.Errorw("failed to close connection", "msg", err.Error())
		}
	}()

	for {
		lenB := make([]byte, 8)
		if _, err := conn.Read(lenB); err != nil {
			if err == io.EOF {
				break
			}
			return errors.Wrap(err, "reading length bytes")
		}
		length, _ := binary.Varint(lenB)

		payload := make([]byte, length)
		if _, err := conn.Read(payload); err != nil {
			return errors.Wrap(err, "reading payload bytes")
		}
		pbp := Payload{}
		if err := proto.Unmarshal(payload, &pbp); err != nil {
			return errors.Wrap(err, "unmarshalling payload message")
		}

		realMsg, err := unwrap(pbp)
		if err != nil {
			return errors.Wrap(err, "unwrapping proto")
		}
		events.Emit(int(pbp.Type), PacketEvent{
			Conn: conn,
			Message: realMsg,
		})
	}
	return nil
}
