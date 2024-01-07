package precog

import (
	"context"
	"fmt"
	"math/big"
	"sync"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	ethCore "github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/params"
	"github.com/sirupsen/logrus"

	"github.com/FeurJak/Prudence/pkg/precog/core/network/fetcher"
	"github.com/FeurJak/Prudence/pkg/precog/core/network/protocols/eth"

	"github.com/FeurJak/Prudence/pkg/geth_cli"

	"github.com/FeurJak/Prudence/pkg/precog/core/network/txpool"
	"github.com/FeurJak/Prudence/pkg/precog/core/network/txpool/legacypool"
)

var genesisBlockBytes = []byte{123, 34, 100, 105, 102, 102, 105, 99, 117, 108, 116, 121, 34, 58, 34, 48, 120, 52, 48, 48, 48, 48, 48, 48, 48, 48, 34, 44, 34, 101, 120, 116, 114, 97, 68, 97, 116, 97, 34, 58, 34, 48, 120, 49, 49, 98, 98, 101, 56, 100, 98, 52, 101, 51, 52, 55, 98, 52, 101, 56, 99, 57, 51, 55, 99, 49, 99, 56, 51, 55, 48, 101, 52, 98, 53, 101, 100, 51, 51, 97, 100, 98, 51, 100, 98, 54, 57, 99, 98, 100, 98, 55, 97, 51, 56, 101, 49, 101, 53, 48, 98, 49, 98, 56, 50, 102, 97, 34, 44, 34, 103, 97, 115, 76, 105, 109, 105, 116, 34, 58, 34, 48, 120, 49, 51, 56, 56, 34, 44, 34, 103, 97, 115, 85, 115, 101, 100, 34, 58, 34, 48, 120, 48, 34, 44, 34, 104, 97, 115, 104, 34, 58, 34, 48, 120, 100, 52, 101, 53, 54, 55, 52, 48, 102, 56, 55, 54, 97, 101, 102, 56, 99, 48, 49, 48, 98, 56, 54, 97, 52, 48, 100, 53, 102, 53, 54, 55, 52, 53, 97, 49, 49, 56, 100, 48, 57, 48, 54, 97, 51, 52, 101, 54, 57, 97, 101, 99, 56, 99, 48, 100, 98, 49, 99, 98, 56, 102, 97, 51, 34, 44, 34, 108, 111, 103, 115, 66, 108, 111, 111, 109, 34, 58, 34, 48, 120, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 34, 44, 34, 109, 105, 110, 101, 114, 34, 58, 34, 48, 120, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 34, 44, 34, 109, 105, 120, 72, 97, 115, 104, 34, 58, 34, 48, 120, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 34, 44, 34, 110, 111, 110, 99, 101, 34, 58, 34, 48, 120, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 52, 50, 34, 44, 34, 110, 117, 109, 98, 101, 114, 34, 58, 34, 48, 120, 48, 34, 44, 34, 112, 97, 114, 101, 110, 116, 72, 97, 115, 104, 34, 58, 34, 48, 120, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 34, 44, 34, 114, 101, 99, 101, 105, 112, 116, 115, 82, 111, 111, 116, 34, 58, 34, 48, 120, 53, 54, 101, 56, 49, 102, 49, 55, 49, 98, 99, 99, 53, 53, 97, 54, 102, 102, 56, 51, 52, 53, 101, 54, 57, 50, 99, 48, 102, 56, 54, 101, 53, 98, 52, 56, 101, 48, 49, 98, 57, 57, 54, 99, 97, 100, 99, 48, 48, 49, 54, 50, 50, 102, 98, 53, 101, 51, 54, 51, 98, 52, 50, 49, 34, 44, 34, 115, 104, 97, 51, 85, 110, 99, 108, 101, 115, 34, 58, 34, 48, 120, 49, 100, 99, 99, 52, 100, 101, 56, 100, 101, 99, 55, 53, 100, 55, 97, 97, 98, 56, 53, 98, 53, 54, 55, 98, 54, 99, 99, 100, 52, 49, 97, 100, 51, 49, 50, 52, 53, 49, 98, 57, 52, 56, 97, 55, 52, 49, 51, 102, 48, 97, 49, 52, 50, 102, 100, 52, 48, 100, 52, 57, 51, 52, 55, 34, 44, 34, 115, 105, 122, 101, 34, 58, 34, 48, 120, 50, 49, 99, 34, 44, 34, 115, 116, 97, 116, 101, 82, 111, 111, 116, 34, 58, 34, 48, 120, 100, 55, 102, 56, 57, 55, 52, 102, 98, 53, 97, 99, 55, 56, 100, 57, 97, 99, 48, 57, 57, 98, 57, 97, 100, 53, 48, 49, 56, 98, 101, 100, 99, 50, 99, 101, 48, 97, 55, 50, 100, 97, 100, 49, 56, 50, 55, 97, 49, 55, 48, 57, 100, 97, 51, 48, 53, 56, 48, 102, 48, 53, 52, 52, 34, 44, 34, 116, 105, 109, 101, 115, 116, 97, 109, 112, 34, 58, 34, 48, 120, 48, 34, 44, 34, 116, 111, 116, 97, 108, 68, 105, 102, 102, 105, 99, 117, 108, 116, 121, 34, 58, 34, 48, 120, 52, 48, 48, 48, 48, 48, 48, 48, 48, 34, 44, 34, 116, 114, 97, 110, 115, 97, 99, 116, 105, 111, 110, 115, 34, 58, 91, 93, 44, 34, 116, 114, 97, 110, 115, 97, 99, 116, 105, 111, 110, 115, 82, 111, 111, 116, 34, 58, 34, 48, 120, 53, 54, 101, 56, 49, 102, 49, 55, 49, 98, 99, 99, 53, 53, 97, 54, 102, 102, 56, 51, 52, 53, 101, 54, 57, 50, 99, 48, 102, 56, 54, 101, 53, 98, 52, 56, 101, 48, 49, 98, 57, 57, 54, 99, 97, 100, 99, 48, 48, 49, 54, 50, 50, 102, 98, 53, 101, 51, 54, 51, 98, 52, 50, 49, 34, 44, 34, 117, 110, 99, 108, 101, 115, 34, 58, 91, 93, 125}

type StateConf struct {
	Remote        bool
	DbOpts        rawdb.OpenOptions
	TrieCacheConf *core.CacheConfig
}

type BackendConfig struct {
	GethConnAddr string
	NoP2P        bool
	State        StateConf
}
type Backend struct {
	// p2p stuff
	p2pServer *p2p.Server
	dnsdisc   enode.Iterator

	engine consensus.Engine

	// State DB
	db dbAPI

	// RPC interface to a GETH node
	gethCli geth_cli.EthInterface

	// Latest Head of the chain
	// We get this from a synced Geth Node
	lastHead    *types.Header
	lastHeadMtx sync.RWMutex

	genesisBlock *types.Block
	difficulty   *big.Int

	logger logrus.Ext1FieldLogger
	conf   *BackendConfig

	txPool    txPool
	txFetcher *fetcher.TxFetcher
	peers     *peerSet

	// for supporting eth subscriptions
	scope         event.SubscriptionScope
	chainHeadFeed event.Feed
}

func NewBackend(logger logrus.Ext1FieldLogger, conf *BackendConfig) (a API, err error) {
	b := &Backend{
		logger: logger,
		conf:   conf,
	}
	// init geth cli interface to a geth node
	if err = b.initGethCli(); err != nil {
		return nil, fmt.Errorf("failed to init geth client: %s", err)
	}

	if err = b.initStateDb(); err != nil {
		return nil, fmt.Errorf("failed to init stateDB: %s", err)
	}

	// P2P required, init txpool & p2p server.
	if !conf.NoP2P {
		b.peers = newPeerSet()
		if err = b.initProtos(); err != nil {
			return nil, fmt.Errorf("failed to init protocols: %s", err)
		}

		// only care about legacy txs for now
		// use default config for now
		legacyPool := legacypool.New(legacypool.DefaultConfig, b)
		b.txPool, err = txpool.New(new(big.Int).SetUint64(legacypool.DefaultConfig.PriceLimit), b, []txpool.SubPool{legacyPool})
		if err != nil {
			return nil, err
		}

		fetchTx := func(peer string, hashes []common.Hash) error {
			p := b.peers.peer(peer)
			if p == nil {
				return errors.New("unknown peer")
			}
			return p.RequestTxs(hashes)
		}
		addTxs := func(txs []*types.Transaction) []error {
			return b.txPool.Add(txs, false, false)
		}

		b.txFetcher = fetcher.NewTxFetcher(b.txPool.Has, addTxs, fetchTx, b.removePeer, b.logger)
	}

	b.engine, err = ethconfig.CreateConsensusEngine(params.MainnetChainConfig, b.db.EthDB())
	if err != nil {
		return nil, err
	}

	go b.Run()

	return NewAPI(b), nil
}

func (b *Backend) updateLatestHeader(ctx context.Context) error {
	// Sub to the latest header from geth node
	headc := make(chan *types.Header)
	sub, err := b.gethCli.SubscribeNewHead(ctx, headc)
	if err != nil {
		return err
	}
	for {
		select {
		case head := <-headc:
			b.lastHeadMtx.Lock()
			b.logger.WithFields(logrus.Fields{
				"Hash":    head.Hash(),
				"Number":  head.Number,
				"peers":   b.peers.count(),
				"mempool": len(b.txPool.Pending(true)),
			}).Info("Updated Head")
			b.lastHead = head
			b.lastHeadMtx.Unlock()

			go func(hash common.Hash, number uint64) {
				latestBlock := b.GetBlock(hash, number)
				if latestBlock != nil {
					b.chainHeadFeed.Send(ethCore.ChainHeadEvent{Block: latestBlock})
				}
			}(head.Hash(), head.Number.Uint64())
		case err := <-sub.Err():
			b.logger.WithFields(logrus.Fields{
				"err": err,
			}).Error("failed newMemPoolLogs")
			sub.Unsubscribe()
			return err
		}
	}
}

func (b *Backend) Run() error {
	errc := make(chan error)

	if !b.conf.NoP2P {
		if err := b.p2pServer.Start(); err != nil {
			return err
		}

		b.logger.Info("Started P2P Server")

		// start tx fetcher
		b.txFetcher.Start()
		b.logger.Info("Started txFetcher")
	}

	// continuously update latest header
	go func() {
		errc <- b.updateLatestHeader(context.Background())
	}()

	for err := range errc {
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *Backend) CurrentBlock() *types.Header {
	defer b.lastHeadMtx.RUnlock()
	b.lastHeadMtx.RLock()
	head := *b.lastHead
	return &head
}

func (b *Backend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.scope.Track(b.chainHeadFeed.Subscribe(ch))
}

func (b *Backend) Genesis() *types.Block {
	return b.genesisBlock
}

func (b *Backend) TxPool() eth.TxPool {
	return b.txPool
}

// return default mainnet config
func (b *Backend) Config() *params.ChainConfig {
	return params.MainnetChainConfig
}

func (b *Backend) Logger() logrus.Ext1FieldLogger {
	return b.logger
}

func (b *Backend) GetDificulty() *big.Int {
	return b.difficulty
}

func (b *Backend) Engine() consensus.Engine {
	return b.engine
}
