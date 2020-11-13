package netpb

import (
	"bytes"
	"compress/gzip"
	"go.uber.org/zap"
	"io/ioutil"
	"sync"
)

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
			tr.logger.Errorw("unzipping page chunk", "err", err)
		}
		inflated, err := ioutil.ReadAll(zr)
		if err != nil {
			tr.logger.Errorw("decompressing chunk", "err", err)
		}
		return true, inflated
	}

	return false, nil
}
