package netpb

import (
	"bytes"
	"compress/gzip"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io/ioutil"
	"math"
	"sync"
)

const chunkSize = 1400

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

func SendPage(reqID int64, pageData []byte, send func(proto.Message) error) error {
	var comp bytes.Buffer
	zw := gzip.NewWriter(&comp)
	if _, err := zw.Write(pageData); err != nil {
		return errors.Wrap(err, "compressing data")
	}
	if err := zw.Close(); err != nil {
		return errors.Wrap(err, "closing compression writer")
	}
	data := comp.Bytes()

	chunkNum := divRoundUp(len(data), chunkSize)
	if err := send(&PageTransfer{
		Id:        reqID,
		Found:     true,
		ChunkSize: int64(chunkSize),
		ChunkNum:  int64(chunkNum),
		TotalSize: int64(len(data)),
	}); err != nil {
		return errors.Wrap(err, "sending page transfer req")
	}
	for i := 0; i < chunkNum; i++ {
		if err := send(&PageChunk{
			Id:    reqID,
			Chunk: int64(i),
			Data:  data[i*chunkSize : min((i+1)*chunkSize, len(data))],
		}); err != nil {
			return errors.Wrap(err, "sending page chunk")
		}
	}
	return nil
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
