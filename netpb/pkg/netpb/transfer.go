package netpb

import (
	"bytes"
	"compress/gzip"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
	"io/ioutil"
	"math"
	"sync"
)

const chunkSize = 10_000

type (
	Transfer struct {
		transfers  map[int64]transfer
		transferMx sync.RWMutex
		logger     *zap.SugaredLogger
	}

	transfer struct {
		ID                  int64
		ChunkNum, ChunkSize int64
		Data                *bytes.Buffer
	}
)

func NewTransfer(logger *zap.SugaredLogger) *Transfer {
	return &Transfer{
		transfers:  make(map[int64]transfer),
		transferMx: sync.RWMutex{},
		logger:     logger,
	}
}

func (tr *Transfer) New(req *PageTransfer) {
	data := make([]byte, 0, req.TotalSize)
	buffer := bytes.NewBuffer(data)
	page := transfer{
		ID:        req.Id,
		ChunkNum:  req.ChunkNum,
		ChunkSize: req.ChunkSize,
		Data:      buffer,
	}
	tr.transferMx.Lock()
	defer tr.transferMx.Unlock()
	tr.transfers[req.Id] = page
}

func (tr *Transfer) Push(chunk *PageChunk) (bool, []byte) {
	tr.transferMx.Lock()
	defer tr.transferMx.Unlock()
	page := tr.transfers[chunk.Id]
	page.Data.Write(chunk.Data)

	if chunk.Chunk == page.ChunkNum-1 {
		zr, err := gzip.NewReader(page.Data)
		if err != nil {
			tr.logger.Errorw("unzipping page", "err", err)
		}
		inflated, err := ioutil.ReadAll(zr)
		if err != nil {
			tr.logger.Errorw("decompressing page data", "err", err)
		}
		return true, inflated
	}

	return false, nil
}

func SendPage(reqID int64, data []byte, send chan proto.Message) {
	chunkNum := divRoundUp(len(data), chunkSize)
	send <- &PageTransfer{
		Id:        reqID,
		Found:     true,
		ChunkSize: int64(chunkSize),
		ChunkNum:  int64(chunkNum),
		TotalSize: int64(len(data)),
	}
	for i := 0; i < chunkNum; i++ {
		send <- &PageChunk{
			Id:    reqID,
			Chunk: int64(i),
			Data:  data[i*chunkSize : min((i+1)*chunkSize, len(data))],
		}
	}
}

func divRoundUp(x, y int) int {
	fx, fy := float64(x), float64(y)
	rx := math.Ceil(fx / fy)
	return int(rx)
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
