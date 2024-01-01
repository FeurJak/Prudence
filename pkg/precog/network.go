package precog

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/Prudence/pkg/geth_cli"

	"github.com/Prudence/pkg/precog/core/network/protocols/eth"

	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/params"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (b *Backend) initGethCli() (err error) {
	if b.gethCli, err = geth_cli.NewGethClient(b.conf.GethConnAddr, b.logger); err != nil {
		return err
	}

	// wait for cli to sync
	for {
		if prog, err := b.gethCli.IsSync(context.Background()); err != nil {
			return fmt.Errorf("failed to get sync progress: %s", err)
		} else {
			if prog == nil {
				break
			}
			time.Sleep(15 * time.Second)
			b.logger.WithFields(logrus.Fields{
				"CurrentBlock": prog.CurrentBlock,
				"HighestBlock": prog.HighestBlock,
				"PulledStates": prog.PulledStates,
				"KnownStates":  prog.KnownStates,
			}).Info("remote client is syncing")
		}
	}
	return nil
}

func (b *Backend) initProtos() (err error) {

	// generate genesis header
	var genhead *types.Header
	if err = json.Unmarshal(genesisBlockBytes, &genhead); err != nil {
		return fmt.Errorf("failed to generate genesis block: %s", err)
	}

	b.genesisBlock = types.NewBlock(genhead, nil, nil, nil, nil)
	b.logger.WithFields(logrus.Fields{
		"genHash":   b.genesisBlock.Hash(),
		"genTime":   b.genesisBlock.Time(),
		"stateRoot": b.lastHead.Root,
		"blockNo":   b.lastHead.Number,
	}).Info("Prepared Genesis Block")

	b.difficulty, _ = big.NewInt(0).SetString("58750003716598352816469", 10)

	// Init P2P Server with default config
	pk, err := crypto.GenerateKey()
	if err != nil {
		return err
	}

	if b.p2pServer, err = eth.InitP2PServer(&p2p.Config{
		Name:        "precog",
		ListenAddr:  ":30304",
		MaxPeers:    10000,
		NAT:         nat.Any(),
		DiscoveryV4: false, // v4 execution layer
		DiscoveryV5: false, // consensus layer
		NoDiscovery: false, // disable discovery
		PrivateKey:  pk,
	}); err != nil {
		return err
	}

	// Init DNS
	if b.dnsdisc, err = eth.InitDNS(); err != nil {
		return err
	}

	// Init Protocols
	protos := eth.MakeProtocols(b, 1, b.dnsdisc)

	// Add protocols to p2p server
	b.p2pServer.Protocols = append([]p2p.Protocol{}, protos...)
	return err
}

func (b *Backend) Handle(peer *eth.Peer, packet eth.Packet) error {
	//peerInfo := peer.Info()
	switch packet := packet.(type) {
	case *eth.TransactionsPacket:
		txs := *packet
		for _, tx := range txs {
			if tx.Type() == types.BlobTxType {
				return errors.New("disallowed broadcast blob transaction")
			}
		}
		/*
			b.logger.WithFields(logrus.Fields{
				"enode": peerInfo.Enode,
				"enr":   peerInfo.ENR,
				"txs":   len(txs),
			}).Infof("Got TransactionsPacket packet")
		*/
		b.txFetcher.Enqueue(peer.ID(), txs, false)
	case *eth.PooledTransactionsResponse:
		txs := *packet
		/*
			b.logger.WithFields(logrus.Fields{
				"enode": peerInfo.Enode,
				"enr":   peerInfo.ENR,
				"txs":   len(txs),
			}).Infof("Got PooledTransactionsResponse packet")
		*/
		return b.txFetcher.Enqueue(peer.ID(), txs, true)
	case *eth.NewPooledTransactionHashesPacket68:
		/*
			b.logger.WithFields(logrus.Fields{
				"enode": peerInfo.Enode,
				"enr":   peerInfo.ENR,
				"txs":   len(packet.Hashes),
			}).Infof("Got NewPooledTransactionHashesPacket68 packet")
		*/
		return b.txFetcher.Notify(peer.ID(), packet.Types, packet.Sizes, packet.Hashes)
	case *eth.NewPooledTransactionHashesPacket67:
		txs := *packet
		/*
			b.logger.WithFields(logrus.Fields{
				"enode": peerInfo.Enode,
				"enr":   peerInfo.ENR,
				"txs":   len(txs),
			}).Infof("Got NewPooledTransactionHashesPacket68 packet")
		*/
		return b.txFetcher.Notify(peer.ID(), nil, nil, txs)
	default:
		return fmt.Errorf("unexpected eth packet type: %T", packet)
	}
	return nil
}

func (b *Backend) PeerInfo(id enode.ID) interface{} {
	return nil
}

func (b *Backend) runPeer(peer *eth.Peer, handler eth.Handler) error {
	// perform handshake
	latesthead := b.CurrentBlock()
	rndmForkId := forkid.NewID(params.MainnetChainConfig, b.genesisBlock, latesthead.Number.Uint64(), latesthead.Time)

	err := peer.Handshake(1, b.difficulty, latesthead.Hash(), b.genesisBlock.Hash(), rndmForkId, nil)
	if err != nil {
		/*
			peerInfo := peer.Info()
			b.logger.WithFields(logrus.Fields{
				"p":     peer.ID(),
				"name":  peerInfo.Name,
				"enode": peerInfo.Enode,
				"enr":   peerInfo.ENR,
				"error": err,
			}).Error("peer.Handshake() failed")
		*/
		return err
	}

	// add peer
	b.peers.registerPeer(peer, nil)
	return handler(peer)
}

func (b *Backend) RunPeer(peer *eth.Peer, handler eth.Handler) error {
	return b.runPeer(peer, handler)
}

// removePeer requests disconnection of a peer.
func (b *Backend) removePeer(id string) {
	peer := b.peers.peer(id)
	if peer != nil {
		peer.Peer.Disconnect(p2p.DiscUselessPeer)
	}
}
