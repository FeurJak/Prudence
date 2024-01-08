package precog

import (
	"github.com/FeurJak/Prudence/pkg/precog/core/network/txpool"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
)

type dbAPI interface {
	GetHead() (head common.Hash, err error)
	GetHeaderNumber(hash common.Hash) (number *uint64, err error)
	GetBlock(hash common.Hash, number uint64) (block *types.Block, err error)
	GetLatestBlock() (block *types.Block, err error)
	StateAt(root common.Hash) (*state.StateDB, error)
	EthDB() ethdb.Database
	GetBlockByNumber(number uint64) (block *types.Block, err error)
	GetReceiptsByHash(hash common.Hash) types.Receipts
	GetReceiptsByNumber(number uint64) (receipts types.Receipts, err error)
	GetLatestBlockNumber() (number *uint64, err error)
}

// txPool defines the methods needed from a transaction pool implementation to
// support all the operations needed by the Ethereum chain protocols.
type txPool interface {
	// Has returns an indicator whether txpool has a transaction
	// cached with the given hash.
	Has(hash common.Hash) bool

	// Get retrieves the transaction from local txpool with given
	// tx hash.
	Get(hash common.Hash) *types.Transaction

	// Add should add the given transactions to the pool.
	Add(txs []*types.Transaction, local bool, sync bool) []error

	// Pending should return pending transactions.
	// The slice should be modifiable by the caller.
	Pending(enforceTips bool) map[common.Address][]*txpool.LazyTransaction

	// SubscribeTransactions subscribes to new transaction events. The subscriber
	// can decide whether to receive notifications only for newly seen transactions
	// or also for reorged out ones.
	SubscribeTransactions(ch chan<- core.NewTxsEvent, reorgs bool) event.Subscription
}

type API interface {
	Engine() consensus.Engine
	SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription
	SubToPendingTx(ch chan<- core.NewTxsEvent) event.Subscription
	CurrentStateAndHeader() (stateDb *state.StateDB, header *types.Header, err error)
	GetHeader(common.Hash, uint64) *types.Header
	GetBlockByNumber(number uint64) (block *types.Block)
	GetReceiptsByHash(hash common.Hash) types.Receipts
	GetReceiptsByNumber(number uint64) (receipts types.Receipts, err error)
	GetLatestBlockNumber() (number *uint64, err error)
}
