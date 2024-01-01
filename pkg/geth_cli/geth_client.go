package geth_cli

/*
	██████  ███████ ████████ ██   ██
	██      ██        ██     ██   ██
	██  ███ █████     ██     ███████
	██  ██  ██        ██     ██   ██
	██████  ███████   ██     ██   ██
*/

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/sirupsen/logrus"
)

// Client defines typed wrappers for the Ethereum RPC API.
type GethClient struct {
	connAddr string
	c        *rpc.Client
	mtx      *sync.RWMutex
	logger   logrus.Ext1FieldLogger
}

type EthInterface interface {
	RpcCl() *rpc.Client
	BlockNumber(ctx context.Context) (uint64, error)
	BlockByNumber(ctx context.Context, BlockNo *big.Int) (*types.Block, error)
	Call(ctx context.Context, method string, args ...interface{}) (raw json.RawMessage, err error)
	SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error)
	IsSync(ctx context.Context) (*ethereum.SyncProgress, error)

	SubscribeTo(ctx context.Context, namespace string, ch chan json.RawMessage, args ...interface{}) (ethereum.Subscription, error)
	GetLatestHeader(ctx context.Context) (*types.Header, error)
}

// Dial connects a client to the given URL.
func (ec *GethClient) dial() (err error) {
	return ec.dialWithContext(context.Background())
}

func (ec *GethClient) dialWithContext(ctx context.Context) (err error) {
	ec.c, err = rpc.DialContext(ctx, ec.connAddr)
	return err
}

func (ec *GethClient) connect() error {
	ec.logger.Infof("connecting to %s", ec.connAddr)
	ctx := context.Background()
	err := ec.dialWithContext(ctx)
	ec.mtx.Unlock()
	if err != nil {
		return err
	}
	block, err := ec.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("{connect} could not get blocknumber from EVM Client, conn: %s", ec.connAddr)
	}

	ec.logger.WithFields(logrus.Fields{"method": "connect()", "block": block}).Info("connected")
	return nil
}

// NewClient creates a client that uses the given RPC client.
func NewGethClient(connAddr string, logger logrus.Ext1FieldLogger) (EthInterface, error) {
	cl := &GethClient{logger: logger, connAddr: connAddr, mtx: new(sync.RWMutex)}
	cl.mtx.Lock()
	err := cl.connect()
	return cl, err
}

type rpcBlock struct {
	Number       string           `json:"number"           gencodec:"required"`
	Time         string           `json:"timestamp"        gencodec:"required"`
	Hash         common.Hash      `json:"hash"`
	Transactions []rpcTransaction `json:"transactions"`
	UncleHashes  []common.Hash    `json:"uncles"`
}

type txExtraInfo struct {
	BlockNumber *string         `json:"blockNumber,omitempty"`
	BlockHash   *common.Hash    `json:"blockHash,omitempty"`
	From        *common.Address `json:"from,omitempty"`
}

type rpcTransaction struct {
	tx *types.Transaction
	txExtraInfo
}

func (ec *GethClient) RpcCl() *rpc.Client {
	return ec.c
}

func (ec *GethClient) Call(ctx context.Context, method string, args ...interface{}) (raw json.RawMessage, err error) {
	err = ec.c.CallContext(ctx, &raw, method, args...)
	if err != nil {
		return nil, err
	} else if len(raw) == 0 {
		return nil, ethereum.NotFound
	}
	return raw, err
}

func (ec *GethClient) GetLatestHeader(ctx context.Context) (*types.Header, error) {
	var head *types.Header
	var body rpcBlock

	var raw json.RawMessage
	err := ec.c.CallContext(ctx, &raw, "eth_getBlockByNumber", "latest", true)
	if err != nil {
		ec.logger.WithFields(logrus.Fields{"method": "eth_getBlockByNumber", "error": err}).Error("GetLatestHeader")
		return nil, err
	} else if len(raw) == 0 {
		return nil, ethereum.NotFound
	}

	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}

	if head.UncleHash == types.EmptyUncleHash && len(body.UncleHashes) > 0 {
		err = fmt.Errorf("server returned non-empty uncle list but block header indicates no uncles")
	} else if head.UncleHash != types.EmptyUncleHash && len(body.UncleHashes) == 0 {
		err = fmt.Errorf("server returned empty uncle list but block header indicates uncles")
	} else if head.TxHash == types.EmptyRootHash && len(body.Transactions) > 0 {
		err = fmt.Errorf("server returned non-empty transaction list but block header indicates no transactions")
	} else if head.TxHash != types.EmptyRootHash && len(body.Transactions) == 0 {
		err = fmt.Errorf("server returned empty transaction list but block header indicates transactions")
	}

	if err != nil {
		ec.logger.WithFields(logrus.Fields{"method": "eth_getBlockByNumber", "error": err}).Error("GetLatestHeader")
		return nil, err
	}
	return head, err
}

func (ec *GethClient) getBlock(ctx context.Context, method string, args ...interface{}) (*types.Block, error) {

	var raw json.RawMessage
	err := ec.c.CallContext(ctx, &raw, method, args...)
	if err != nil {
		return nil, err
	} else if len(raw) == 0 {
		return nil, ethereum.NotFound
	}
	// Decode header and transactions.
	var head *types.Header
	var body rpcBlock

	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	// Quick-verify transaction and uncle lists. This mostly helps with debugging the server.
	if head.UncleHash == types.EmptyUncleHash && len(body.UncleHashes) > 0 {
		return nil, fmt.Errorf("server returned non-empty uncle list but block header indicates no uncles")
	}
	if head.UncleHash != types.EmptyUncleHash && len(body.UncleHashes) == 0 {
		return nil, fmt.Errorf("server returned empty uncle list but block header indicates uncles")
	}
	if head.TxHash == types.EmptyRootHash && len(body.Transactions) > 0 {
		return nil, fmt.Errorf("server returned non-empty transaction list but block header indicates no transactions")
	}
	if head.TxHash != types.EmptyRootHash && len(body.Transactions) == 0 {
		return nil, fmt.Errorf("server returned empty transaction list but block header indicates transactions")
	}
	// Load uncles because they are not included in the block response.
	var uncles []*types.Header
	if len(body.UncleHashes) > 0 {
		uncles = make([]*types.Header, len(body.UncleHashes))
		reqs := make([]rpc.BatchElem, len(body.UncleHashes))
		for i := range reqs {
			reqs[i] = rpc.BatchElem{
				Method: "eth_getUncleByBlockHashAndIndex",
				Args:   []interface{}{body.Hash, hexutil.EncodeUint64(uint64(i))},
				Result: &uncles[i],
			}
		}
		if err := ec.c.BatchCallContext(ctx, reqs); err != nil {
			return nil, err
		}
		for i := range reqs {
			if reqs[i].Error != nil {
				return nil, reqs[i].Error
			}
			if uncles[i] == nil {
				return nil, fmt.Errorf("got null header for uncle %d of block %x", i, body.Hash[:])
			}
		}
	}
	// Fill the sender cache of transactions in the block.
	txs := make([]*types.Transaction, len(body.Transactions))
	for i, tx := range body.Transactions {
		if tx.From != nil {
			setSenderFromServer(tx.tx, *tx.From, body.Hash)
		}
		txs[i] = tx.tx
	}
	return types.NewBlockWithHeader(head).WithBody(txs, uncles), nil
}

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	pending := big.NewInt(-1)
	if number.Cmp(pending) == 0 {
		return "pending"
	}
	return hexutil.EncodeBig(number)
}

func (ec *GethClient) BlockByNumber(ctx context.Context, BlockNo *big.Int) (*types.Block, error) {
	block, err := ec.getBlock(ctx, "eth_getBlockByNumber", toBlockNumArg(BlockNo), true)
	if err != nil {
		ec.logger.WithFields(logrus.Fields{"method": "eth_getBlockByNumber", "error": err}).Error("BlockByNumber")
		return nil, err
	}
	return block, err
}

// BlockNumber returns the most recent block number
func (ec *GethClient) BlockNumber(ctx context.Context) (uint64, error) {
	defer ec.mtx.RUnlock()
	ec.mtx.RLock()

	var result hexutil.Uint64
	err := ec.c.CallContext(ctx, &result, "eth_blockNumber")
	if err != nil {
		ec.logger.WithFields(logrus.Fields{"method": "eth_blockNumber", "error": err}).Error("BlockNumber")
		return 0, err
	}
	return uint64(result), err
}

func (ec *GethClient) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	sub, err := ec.c.EthSubscribe(ctx, ch, "newHeads")
	if err != nil {
		ec.logger.WithFields(logrus.Fields{"subscription": "newHeads", "error": err}).Error("SubscribeNewHead")
		return nil, err
	}
	return sub, nil
}

// SVE stuff

// args[0] is the subscription name
func (ec *GethClient) SubscribeTo(ctx context.Context, namespace string, ch chan json.RawMessage, args ...interface{}) (ethereum.Subscription, error) {
	sub, err := ec.c.Subscribe(ctx, namespace, ch, args...)
	if err != nil {
		ec.logger.WithFields(logrus.Fields{"subscription": args[0].(string), "error": err}).Error("SubscribeTo")
		return nil, err
	}
	return sub, nil
}

// server. It is stored in the transaction's sender address cache to avoid an additional
// request in TransactionSender.
type senderFromServer struct {
	addr      common.Address
	blockhash common.Hash
}

var errNotCached = errors.New("sender not cached")

func setSenderFromServer(tx *types.Transaction, addr common.Address, block common.Hash) {
	// Use types.Sender for side-effect to store our signer into the cache.
	types.Sender(&senderFromServer{addr, block}, tx)
}

func (s *senderFromServer) Equal(other types.Signer) bool {
	os, ok := other.(*senderFromServer)
	return ok && os.blockhash == s.blockhash
}

func (s *senderFromServer) Sender(tx *types.Transaction) (common.Address, error) {
	if s.addr == (common.Address{}) {
		return common.Address{}, errNotCached
	}
	return s.addr, nil
}

func (s *senderFromServer) ChainID() *big.Int {
	panic("can't sign with senderFromServer")
}
func (s *senderFromServer) Hash(tx *types.Transaction) common.Hash {
	panic("can't sign with senderFromServer")
}
func (s *senderFromServer) SignatureValues(tx *types.Transaction, sig []byte) (R, S, V *big.Int, err error) {
	panic("can't sign with senderFromServer")
}

// rpcProgress is a copy of SyncProgress with hex-encoded fields.
type rpcProgress struct {
	StartingBlock hexutil.Uint64
	CurrentBlock  hexutil.Uint64
	HighestBlock  hexutil.Uint64

	PulledStates hexutil.Uint64
	KnownStates  hexutil.Uint64

	SyncedAccounts      hexutil.Uint64
	SyncedAccountBytes  hexutil.Uint64
	SyncedBytecodes     hexutil.Uint64
	SyncedBytecodeBytes hexutil.Uint64
	SyncedStorage       hexutil.Uint64
	SyncedStorageBytes  hexutil.Uint64
	HealedTrienodes     hexutil.Uint64
	HealedTrienodeBytes hexutil.Uint64
	HealedBytecodes     hexutil.Uint64
	HealedBytecodeBytes hexutil.Uint64
	HealingTrienodes    hexutil.Uint64
	HealingBytecode     hexutil.Uint64
}

func (p *rpcProgress) toSyncProgress() *ethereum.SyncProgress {
	if p == nil {
		return nil
	}
	return &ethereum.SyncProgress{
		StartingBlock:       uint64(p.StartingBlock),
		CurrentBlock:        uint64(p.CurrentBlock),
		HighestBlock:        uint64(p.HighestBlock),
		PulledStates:        uint64(p.PulledStates),
		KnownStates:         uint64(p.KnownStates),
		SyncedAccounts:      uint64(p.SyncedAccounts),
		SyncedAccountBytes:  uint64(p.SyncedAccountBytes),
		SyncedBytecodes:     uint64(p.SyncedBytecodes),
		SyncedBytecodeBytes: uint64(p.SyncedBytecodeBytes),
		SyncedStorage:       uint64(p.SyncedStorage),
		SyncedStorageBytes:  uint64(p.SyncedStorageBytes),
		HealedTrienodes:     uint64(p.HealedTrienodes),
		HealedTrienodeBytes: uint64(p.HealedTrienodeBytes),
		HealedBytecodes:     uint64(p.HealedBytecodes),
		HealedBytecodeBytes: uint64(p.HealedBytecodeBytes),
		HealingTrienodes:    uint64(p.HealingTrienodes),
		HealingBytecode:     uint64(p.HealingBytecode),
	}
}

func (ec *GethClient) IsSync(ctx context.Context) (*ethereum.SyncProgress, error) {
	var raw json.RawMessage
	if err := ec.c.CallContext(ctx, &raw, "eth_syncing"); err != nil {
		return nil, err
	}
	// Handle the possible response types
	var syncing bool
	if err := json.Unmarshal(raw, &syncing); err == nil {
		return nil, nil // Not syncing (always false)
	}
	var p *rpcProgress
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	return p.toSyncProgress(), nil
}
