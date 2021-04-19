package peer

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha512"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
	"wilfrid.me/archive-central/pkg/central/core"
	"wilfrid.me/archive-central/pkg/central/record"
)

const (
	cacheTime      = time.Hour
	mirrorAttempts = 4
	mirrorTime     = time.Minute * 2
)

// Network describes a peer group which communicates over a network.
type Network struct {
	records      record.Repository
	cache        *redis.Client
	requestGroup singleflight.Group
	logger       *zap.SugaredLogger
	mirrors      int
}

// NewNetwork returns a network instance which uses the specified records repository and cache.
func NewNetwork(records record.Repository, cache *redis.Client, mirrors int, logger *zap.SugaredLogger) *Network {
	return &Network{
		records: records,
		cache:   cache,
		mirrors: mirrors,
		logger:  logger,
	}
}

// Get returns the file of the page with the specified id.
func (nw *Network) Get(ctx context.Context, id int) ([]byte, error) {
	// check the cache for this page
	//key := fmt.Sprintf("snapshot-%d", id)
	//sctx, cfn := context.WithTimeout(ctx, time.Second*2)
	//defer cfn()
	//cached, err := nw.cache.Get(sctx, key).Bytes()
	//if err != nil && err != redis.Nil {
	//	nw.logger.Errorw("failed to query cache", "msg", err.Error())
	//}
	//if err != redis.Nil {
	//	return cached, nil
	//}

	nw.logger.Infow("getting record", "id", id)
	snap, err := nw.records.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "getting snapshot record")
	}
	mirrors, err := nw.records.GetMirrors(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "getting mirror records")
	}

	// make an array of all peers who have this page
	var peers []core.Peer
	peers = append(peers, snap.Peer)
	peers = append(peers, mirrors...)

	nw.logger.Infow("messaging peers", "snap", snap)
	data, err, _ := nw.requestGroup.Do(fmt.Sprintf("network-get-%d", id), func() (interface{}, error) {
		return nw.getWithFallback(ctx, snap, peers)
	})
	if err != nil {
		return nil, errors.Wrap(err, "messaging peers")
	}
	bdata := (data).([]byte)

	//if err := nw.cache.Set(ctx, key, bdata, cacheTime).Err(); err != nil {
	//	nw.logger.Errorw("failed to set cache", "snap", snap, "msg", err.Error())
	//}
	return bdata, nil
}

// Store will query a url and store its content in the specified peer.
func (nw *Network) Store(ctx context.Context, peerID int, url string) (int, error) {
	// find peer to store in
	peer, err := nw.records.GetPeer(ctx, peerID)
	if err != nil {
		return 0, errors.Wrap(err, "getting peer record")
	}

	// download url to memory
	respData, err, _ := nw.requestGroup.Do(fmt.Sprintf("http-get-%s", url), func() (interface{}, error) {
		resp, err := http.Get(url)
		if err != nil {
			return 0, errors.Wrap(err, "downloading url")
		}
		return ioutil.ReadAll(resp.Body)
	})
	if err != nil {
		return 0, errors.Wrap(err, "reading url data")
	}
	data := respData.([]byte)

	checkSum, err := getHash(data)
	if err != nil {
		return 0, errors.Wrap(err, "calculating hash")
	}

	// add page record
	recID, err := nw.records.Add(ctx, core.Snapshot{
		Sum:  checkSum,
		URL:  url,
		Time: time.Now(),
		Peer: core.Peer{
			ID: peerID,
		},
	})
	if err != nil {
		return 0, errors.Wrap(err, "creating new page record")
	}

	if nw.mirrors > 0 {
		go func() {
			nctx, cfn := context.WithTimeout(context.Background(), mirrorTime)
			defer cfn()
			if err := nw.sendToMirrors(nctx, int(recID), peerID, data); err != nil {
				nw.logger.Errorw("sending data to mirrors", "id", recID, "err", err.Error())
			}
		}()
	}

	if err := sendTo(ctx, peer.Address, int(recID), data); err != nil {
		return 0, errors.Wrap(err, "sending to main peer")
	}

	return int(recID), nil
}

// sendToMirrors will pick a random selection of peers and send them a page file to store.
func (nw *Network) sendToMirrors(ctx context.Context, recID, exID int, data []byte) error {
	peers, err := nw.records.GetPeers(ctx, nw.mirrors*mirrorAttempts)
	if err != nil {
		return errors.Wrap(err, "getting list of peers")
	}
	needed := nw.mirrors
	for needed > 0 && len(peers) > 0 {
		peer := peers[0]
		peers = peers[1:]
		// skip main peer as this page will already be stored
		if peer.ID == exID {
			continue
		}

		if err := sendTo(ctx, peer.Address, recID, data); err != nil {
			nw.logger.Errorw("sending to mirror", "msg", err.Error())
			continue
		}

		if _, err := nw.records.AddMirror(ctx, peer.ID, recID); err != nil {
			nw.logger.Errorw("adding mirror record", "msg", err.Error())
			continue
		}
		needed--
	}
	if needed > 0 {
		return errors.New("could not find enough available peers")
	}
	return nil
}

// getWithFallback will return a page file from one of the peers supplied.
func (nw *Network) getWithFallback(ctx context.Context, snap core.Snapshot, peers []core.Peer) ([]byte, error) {
	for _, peer := range peers {
		data, err := nw.getFrom(ctx, snap, peer.Address)
		if err != nil {
			nw.logger.Warnw("couldn't get from peer", "snap", snap, "addr", peer.Address, "msg", err.Error())
			continue
		}
		return data, nil
	}
	return nil, errors.New("none available")
}

// getFrom will return a page file from a single peer with the address supplied.
func (nw *Network) getFrom(ctx context.Context, snap core.Snapshot, address string) ([]byte, error) {
	nw.logger.Infow("getting file from peer", "snap", snap, "addr", address)

	target := fmt.Sprintf("http://%s/page/%d", address, snap.ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, errors.Wrap(err, "making request")
	}

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "sending request")
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("file not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("request not successful")
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "making request")
	}

	result, err := decompress(data)
	if err != nil {
		return nil, errors.Wrap(err, "decompressing file")
	}

	checkSum, err := getHash(result)
	if err != nil {
		return nil, errors.Wrap(err, "calculating hash")
	}
	if snap.Sum != checkSum {
		return nil, errors.Errorf("wrong hash, want: %s, got: %s", snap.Sum, checkSum)
	}

	nw.logger.Infow("got file from peer", "snap", snap, "addr", address)
	return result, nil
}

// sendTo will store a page file in the peer at the given address.
func sendTo(ctx context.Context, address string, id int, file []byte) error {
	data, err := compress(file)
	if err != nil {
		return errors.Wrap(err, "compressing file")
	}
	target := fmt.Sprintf("http://%s/page/%d", address, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, target, bytes.NewReader(data))
	if err != nil {
		return errors.Wrap(err, "making request")
	}
	req.Header.Set("id", strconv.Itoa(id))

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "sending request")
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New("file not accepted")
	}
	return nil
}

func compress(file []byte) ([]byte, error) {
	var comp bytes.Buffer
	zw := gzip.NewWriter(&comp)
	if _, err := zw.Write(file); err != nil {
		return nil, errors.Wrap(err, "compressing data")
	}
	if err := zw.Close(); err != nil {
		return nil, errors.Wrap(err, "closing compression writer")
	}
	return comp.Bytes(), nil
}

func decompress(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, errors.Wrap(err, "creating reader")
	}
	res, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "reading data")
	}
	return res, nil
}

func getHash(data []byte) (string, error) {
	hash := sha512.New512_256()
	if _, err := hash.Write(data); err != nil {
		return "", errors.Wrap(err, "writing hash")
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
