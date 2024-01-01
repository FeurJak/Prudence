/*
	Taken from eth / protocols / eth / discovery.go
*/

package eth

import (
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

// enrEntry is the ENR entry which advertises `eth` protocol on the discovery.
type enrEntry struct {
	ForkID forkid.ID // Fork identifier per EIP-2124

	// Ignore additional fields (for forward compatibility).
	Rest []rlp.RawValue `rlp:"tail"`
}

// ENRKey implements enr.Entry.
func (e enrEntry) ENRKey() string {
	return "eth"
}

// UpdateENR updates the nodes ENR value
func UpdateENR(backend Backend, ln *enode.LocalNode) {
	ln.Set(CreateENR(backend))
}

// CreateENR constructs an `eth` ENR entry based on the current state of the chain.
func CreateENR(backend Backend) *enrEntry {
	head := backend.CurrentBlock()
	return &enrEntry{
		ForkID: forkid.NewID(params.MainnetChainConfig, backend.Genesis(), head.Number.Uint64(), head.Time),
	}
}
