package peer

import (
	"bytes"
	"compress/gzip"
	"go.uber.org/zap"
	"wilfrid.me/archive-child/pkg/child/file"
	"wilfrid.me/netpb/pkg/netpb"
)

type Node struct {
	events    *netpb.Emitter
	files     *file.Repository
	transfers *netpb.Transfer
	logger    *zap.SugaredLogger
}

func New(events *netpb.Emitter, files *file.Repository, logger *zap.SugaredLogger) *Node {
	node := &Node{
		events:    events,
		files:     files,
		transfers: netpb.NewTransfer(logger),
		logger:    logger,
	}
	node.subscribe()
	return node
}

func (no *Node) subscribe() {
	no.events.On(netpb.ID(&netpb.PageRequest{}), func(i interface{}) {
		p := i.(netpb.PacketEvent)
		req := p.Message.(*netpb.PageRequest)

		data, err := no.files.Get(int(req.Id))
		if err != nil {
			no.logger.Errorw("failed to read file", "msg", err.Error())
			return
		}

		var comp bytes.Buffer
		zw := gzip.NewWriter(&comp)

		if _, err := zw.Write(data); err != nil {
			no.logger.Errorw("failed to compress file", "msg", err.Error())
			return
		}
		if err := zw.Close(); err != nil {
			no.logger.Errorw("failed to close compression", "msg", err.Error())
			return
		}

		// read file with that Id
		// split into chunks
		// send page response packet
		// send chunks in page chunk packets

		sendData := comp.Bytes()
		chunkSize := int64(len(sendData))
		chunkNum := int64(1)
		totalSize := chunkSize

		if err := netpb.Write(p.Conn, &netpb.PageTransfer{
			Id:        req.Id,
			Found:     true,
			ChunkSize: chunkSize,
			ChunkNum:  chunkNum,
			TotalSize: totalSize,
		}); err != nil {
			no.logger.Errorw("failed to send page response", "msg", err.Error())
		}

		if err := netpb.Write(p.Conn, &netpb.PageChunk{
			Id:    req.Id,
			Chunk: 0,
			Data:  sendData,
		}); err != nil {
			no.logger.Errorw("failed to send page response", "msg", err.Error())
		}
	})

	no.events.On(netpb.ID(&netpb.PageTransfer{}), func(i interface{}) {
		p := i.(netpb.PacketEvent)
		req := p.Message.(*netpb.PageTransfer)
		no.transfers.New(req)
	})

	no.events.On(netpb.ID(&netpb.PageChunk{}), func(i interface{}) {
		p := i.(netpb.PacketEvent)
		chunk := p.Message.(*netpb.PageChunk)
		done, result := no.transfers.Push(chunk)
		if done {
			if err := no.files.Store(int(chunk.Id), result); err != nil {
				no.logger.Errorw("failed to store page in file system", "msg", err.Error())
			}
		}
	})
}
