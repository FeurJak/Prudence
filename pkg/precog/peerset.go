package precog

/*
	Taken from eth / peerset.go
*/

import (
	"errors"
	"sync"

	"github.com/Prudence/pkg/precog/core/network/protocols/eth"

	"github.com/ethereum/go-ethereum/eth/protocols/snap"
)

// ethPeerInfo represents a short summary of the `eth` sub-protocol metadata known
// about a connected peer.
type ethPeerInfo struct {
	Version uint `json:"version"` // Ethereum protocol version negotiated
}

// ethPeer is a wrapper around eth.Peer to maintain a few extra metadata.
type ethPeer struct {
	*eth.Peer
}

// info gathers and returns some `eth` protocol metadata known about a peer.
func (p *ethPeer) info() *ethPeerInfo {
	return &ethPeerInfo{
		Version: p.Version(),
	}
}

var (
	// errPeerSetClosed is returned if a peer is attempted to be added or removed
	// from the peer set after it has been terminated.
	errPeerSetClosed = errors.New("peerset closed")

	// errPeerAlreadyRegistered is returned if a peer is attempted to be added
	// to the peer set, but one with the same id already exists.
	errPeerAlreadyRegistered = errors.New("peer already registered")

	// errPeerNotRegistered is returned if a peer is attempted to be removed from
	// a peer set, but no peer with the given id exists.
	errPeerNotRegistered = errors.New("peer not registered")

	// errSnapWithoutEth is returned if a peer attempts to connect only on the
	// snap protocol without advertising the eth main protocol.
	errSnapWithoutEth = errors.New("peer connected on snap without compatible eth support")
)

// peerSet represents the collection of active peers currently participating in
// the `eth` protocol without 'snap' extension
type peerSet struct {
	peers map[string]*ethPeer // Peers connected on the `eth` protocol

	lock   sync.RWMutex
	closed bool
}

// newPeerSet creates a new peer set to track the active participants.
func newPeerSet() *peerSet {
	return &peerSet{
		peers: make(map[string]*ethPeer),
	}
}

// registerPeer injects a new `eth` peer into the working set, or returns an error
// if the peer is already known.
func (ps *peerSet) registerPeer(peer *eth.Peer, ext *snap.Peer) error {
	// Start tracking the new peer
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if ps.closed {
		return errPeerSetClosed
	}
	id := peer.ID()
	if _, ok := ps.peers[id]; ok {
		return errPeerAlreadyRegistered
	}
	eth := &ethPeer{
		Peer: peer,
	}
	ps.peers[id] = eth
	return nil
}

// unregisterPeer removes a remote peer from the active set, disabling any further
// actions to/from that particular entity.
func (ps *peerSet) unregisterPeer(id string) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	_, ok := ps.peers[id]
	if !ok {
		return errPeerNotRegistered
	}
	delete(ps.peers, id)
	return nil
}

// peer retrieves the registered peer with the given id.
func (ps *peerSet) peer(id string) *ethPeer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return ps.peers[id]
}

func (ps *peerSet) count() int {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return len(ps.peers)
}
