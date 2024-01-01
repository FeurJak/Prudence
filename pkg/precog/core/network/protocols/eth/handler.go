package eth

import (
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/dnsdisc"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/params"
	"github.com/sirupsen/logrus"
)

/*
p2p defaults from node / defaults.go
node initiates p2p server at node / node.go
server:        &p2p.Server{Config: conf.P2P},
*/
var DefaultP2pConf = p2p.Config{
	ListenAddr:  ":30303",
	MaxPeers:    10000,
	NAT:         nat.Any(),
	DiscoveryV4: false, // v4 execution layer
	DiscoveryV5: false, // consensus layer
	NoDiscovery: false, // disable discovery
}

const dnsPrefix = "enrtree://AKA3AM6LPBYEUDMVNU3BSVQJ5AD45Y7YPOHJLEF6W26QOE4VTUDPE@"

// taken from eth/backend.go
func InitDNS() (ethDialCandidates enode.Iterator, err error) {
	dnsclient := dnsdisc.NewClient(dnsdisc.Config{})
	ethDiscoveryURLs := []string{dnsPrefix + "all" + "." + "mainnet" + ".ethdisco.net"} // mainnet URL taken from params / bootnodes.go
	ethDialCandidates, err = dnsclient.NewIterator(ethDiscoveryURLs...)
	if err != nil {
		return nil, err
	}
	return ethDialCandidates, nil
}

func InitP2PServer(conf *p2p.Config) (p2pServer *p2p.Server, err error) {
	// run default conf if nil
	if conf == nil {
		conf = &DefaultP2pConf
		conf.PrivateKey, err = crypto.GenerateKey()
		if err != nil {
			return nil, err
		}
	}
	p2pServer = &p2p.Server{Config: *conf}
	return p2pServer, nil
}

// Handler is a callback to invoke from an outside runner after the boilerplate
// exchanges have passed.
type Handler func(peer *Peer) error

// Backend defines the data retrieval methods to serve remote requests and the
// callback methods to invoke on remote deliveries.
type Backend interface {
	CurrentBlock() *types.Header // Get latest Header
	Genesis() *types.Block       // Get Genesis Block

	// TxPool retrieves the transaction pool object to serve data.
	TxPool() TxPool

	// RunPeer is invoked when a peer joins on the `eth` protocol. The handler
	// should do any peer maintenance work, handshakes and validations. If all
	RunPeer(peer *Peer, handler Handler) error

	// PeerInfo retrieves all known `eth` information about a peer.
	PeerInfo(id enode.ID) interface{}

	// Handle is a callback to be invoked when a data packet is received from
	// the remote peer. Only packets not consumed by the protocol handler will
	// be forwarded to the backend.
	Handle(peer *Peer, packet Packet) error

	Logger() logrus.Ext1FieldLogger
	Config() *params.ChainConfig
	GetDificulty() *big.Int
}

// TxPool defines the methods needed by the protocol handler to serve transactions.
type TxPool interface {
	// Get retrieves the transaction from the local txpool with the given hash.
	Get(hash common.Hash) *types.Transaction
}

// taken from eth / protocols / eth / handler.go
// MakeProtocols constructs the P2P protocol definitions for `eth`.
func MakeProtocols(backend Backend, network uint64, dnsdisc enode.Iterator) []p2p.Protocol {
	protocols := make([]p2p.Protocol, 0, len(ProtocolVersions))

	for _, version := range ProtocolVersions {
		version := version // Closure

		protocols = append(protocols, p2p.Protocol{
			Name:    ProtocolName,
			Version: version,
			Length:  protocolLengths[version],
			Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
				peer := NewPeer(version, p, rw, backend.TxPool(), backend.Logger())
				defer peer.Close()

				return backend.RunPeer(peer, func(peer *Peer) error {
					return Handle(backend, peer)
				})
			},
			NodeInfo: func() interface{} {
				return nodeInfo(backend, network)
			},
			PeerInfo: func(id enode.ID) interface{} {
				return backend.PeerInfo(id)
			},
			Attributes:     []enr.Entry{CreateENR(backend)},
			DialCandidates: dnsdisc,
		})
	}
	return protocols
}

// NodeInfo represents a short summary of the `eth` sub-protocol metadata
// known about the host peer.
type NodeInfo struct {
	Network    uint64              `json:"network"`    // Ethereum network ID (1=Mainnet, Goerli=5)
	Difficulty *big.Int            `json:"difficulty"` // Total difficulty of the host's blockchain
	Genesis    common.Hash         `json:"genesis"`    // SHA3 hash of the host's genesis block
	Config     *params.ChainConfig `json:"config"`     // Chain configuration for the fork rules
	Head       common.Hash         `json:"head"`       // Hex hash of the host's best owned block
}

// nodeInfo retrieves some `eth` protocol metadata about the running host node.
func nodeInfo(backend Backend, network uint64) *NodeInfo {
	head := backend.CurrentBlock()
	hash := head.Hash()

	return &NodeInfo{
		Network:    network,
		Difficulty: backend.GetDificulty(),
		Genesis:    backend.Genesis().Hash(),
		Config:     backend.Config(),
		Head:       hash,
	}
}

// Handle is invoked whenever an `eth` connection is made that successfully passes
// the protocol handshake. This method will keep processing messages until the
// connection is torn down.
func Handle(backend Backend, peer *Peer) error {
	for {
		if err := handleMessage(backend, peer); err != nil {
			peer.logger.WithFields(logrus.Fields{"error": err}).Error("Message handling failed in `eth`")
			return err
		}
	}
}

type msgHandler func(backend Backend, msg Decoder, peer *Peer) error
type Decoder interface {
	Decode(val interface{}) error
	Time() time.Time
}

var eth67 = map[uint64]msgHandler{
	NewBlockHashesMsg:             handleNewBlockhashes,
	NewBlockMsg:                   handleNewBlock,
	TransactionsMsg:               handleTransactions,
	NewPooledTransactionHashesMsg: handleNewPooledTransactionHashes67,
	GetBlockHeadersMsg:            handleGetBlockHeaders,
	BlockHeadersMsg:               handleBlockHeaders,
	GetBlockBodiesMsg:             handleGetBlockBodies,
	BlockBodiesMsg:                handleBlockBodies,
	GetReceiptsMsg:                handleGetReceipts,
	ReceiptsMsg:                   handleReceipts,
	GetPooledTransactionsMsg:      handleGetPooledTransactions,
	PooledTransactionsMsg:         handlePooledTransactions,
}

var eth68 = map[uint64]msgHandler{
	NewBlockHashesMsg:             handleNewBlockhashes,
	NewBlockMsg:                   handleNewBlock,
	TransactionsMsg:               handleTransactions,
	NewPooledTransactionHashesMsg: handleNewPooledTransactionHashes68,
	GetBlockHeadersMsg:            handleGetBlockHeaders,
	BlockHeadersMsg:               handleBlockHeaders,
	GetBlockBodiesMsg:             handleGetBlockBodies,
	BlockBodiesMsg:                handleBlockBodies,
	GetReceiptsMsg:                handleGetReceipts,
	ReceiptsMsg:                   handleReceipts,
	GetPooledTransactionsMsg:      handleGetPooledTransactions,
	PooledTransactionsMsg:         handlePooledTransactions,
}

// handleMessage is invoked whenever an inbound message is received from a remote
// peer. The remote connection is torn down upon returning any error.
func handleMessage(backend Backend, peer *Peer) error {
	// Read the next message from the remote peer, and ensure it's fully consumed
	msg, err := peer.rw.ReadMsg()
	if err != nil {
		return err
	}

	if msg.Size > maxMessageSize {
		return fmt.Errorf("%w: %v > %v", errMsgTooLarge, msg.Size, maxMessageSize)
	}
	defer msg.Discard()
	var handlers = eth67
	if peer.Version() >= ETH68 {
		handlers = eth68
	}
	if handler := handlers[msg.Code]; handler != nil {
		return handler(backend, msg, peer)
	}
	return fmt.Errorf("%w: %v", errInvalidMsgCode, msg.Code)
}
