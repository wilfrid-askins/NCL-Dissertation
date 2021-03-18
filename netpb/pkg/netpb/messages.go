package netpb

import (
	"encoding/binary"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"net"
	"reflect"
)

var (
	messages = []proto.Message{
		&PageRequest{},
		&PageTransfer{},
		&PageChunk{},
	}

	messageIDs = make(map[reflect.Type]int)
)

func init() {
	for ind, msg := range messages {
		messageIDs[reflect.TypeOf(msg)] = ind
	}
}

func ID(msg proto.Message) int {
	return messageIDs[reflect.TypeOf(msg)]
}

func Write(conn net.Conn, msg proto.Message) error {
	wrapper, err := wrap(msg)
	if err != nil {
		return errors.Wrap(err, "wrapping message")
	}

	data, err := proto.Marshal(&wrapper)
	if err != nil {
		return errors.Wrap(err, "marshalling protobuf msg")
	}

	length := len(data)
	lengthData := make([]byte, 8)
	binary.PutUvarint(lengthData, uint64(length))
	if _, err := conn.Write(lengthData); err != nil {
		return errors.Wrap(err, "writing to connection")
	}

	if _, err := conn.Write(data); err != nil {
		return errors.Wrap(err, "writing to connection")
	}
	return nil
}

func wrap(msg proto.Message) (Payload, error) {
	typeID, pres := messageIDs[reflect.TypeOf(msg)]
	if !pres {
		return Payload{}, errors.New("unknown message type")
	}

	msgData, err := proto.Marshal(msg)
	if err != nil {
		return Payload{}, errors.Wrap(err, "marshalling proto")
	}
	return Payload{
		Type: int64(typeID),
		Data: msgData,
	}, nil
}

func unwrap(p *Payload) (proto.Message, error) {
	var res proto.Message
	switch p.Type {
	case 0:
		res = &PageRequest{}
	case 1:
		res = &PageTransfer{}
	case 2:
		res = &PageChunk{}
	default:
		return nil, errors.New("unknown message type")
	}
	if err := proto.Unmarshal(p.Data, res); err != nil {
		return nil, errors.Wrap(err, "unmarshalling proto")
	}
	return res, nil
}
