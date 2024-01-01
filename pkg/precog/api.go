package precog

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

type api struct {
	b *Backend
}

func NewAPI(b *Backend) API {
	return &api{
		b: b,
	}
}

func (a *api) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return a.b.SubscribeChainHeadEvent(ch)
}

func (a *api) SubToPendingTx(ch chan<- core.NewTxsEvent) event.Subscription {
	return a.b.txPool.SubscribeTransactions(ch, false)
}

func (a *api) CurrentStateAndHeader() (stateDb *state.StateDB, header *types.Header, err error) {
	header = a.b.CurrentBlock()
	stateDb, err = a.b.StateAt(header.Root)
	return
}

func (a *api) GetEVM() {

}

// OverrideAccount indicates the overriding fields of account during the execution
// of a message call.
// Note, state and stateDiff can't be specified at the same time. If state is
// set, message execution will only use the data in the given state. Otherwise
// if statDiff is set, all diff will be applied first and then execute the call
// message.
type OverrideAccount struct {
	Nonce     *hexutil.Uint64              `json:"nonce"`
	Code      *hexutil.Bytes               `json:"code"`
	Balance   **hexutil.Big                `json:"balance"`
	State     *map[common.Hash]common.Hash `json:"state"`
	StateDiff *map[common.Hash]common.Hash `json:"stateDiff"`
}

// StateOverride is the collection of overridden accounts.
type StateOverride map[common.Address]OverrideAccount

// Apply overrides the fields of specified accounts into the given state.
func (diff *StateOverride) Apply(state *state.StateDB) error {
	if diff == nil {
		return nil
	}
	for addr, account := range *diff {
		// Override account nonce.
		if account.Nonce != nil {
			state.SetNonce(addr, uint64(*account.Nonce))
		}
		// Override account(contract) code.
		if account.Code != nil {
			state.SetCode(addr, *account.Code)
		}
		// Override account balance.
		if account.Balance != nil {
			state.SetBalance(addr, (*big.Int)(*account.Balance))
		}
		if account.State != nil && account.StateDiff != nil {
			return fmt.Errorf("account %s has both 'state' and 'stateDiff'", addr.Hex())
		}
		// Replace entire state if caller requires.
		if account.State != nil {
			state.SetStorage(addr, *account.State)
		}
		// Apply state diff into specified accounts.
		if account.StateDiff != nil {
			for key, value := range *account.StateDiff {
				state.SetState(addr, key, value)
			}
		}
	}
	// Now finalize the changes. Finalize is normally performed between transactions.
	// By using finalize, the overrides are semantically behaving as
	// if they were created in a transaction just before the tracing occur.
	state.Finalise(false)
	return nil
}

/*
To-DO:
  - allow to choose custom EVM impls
  - add timeout ctx

func (a *API) Call(ctx context.Context, txMsg ethereum.CallMsg, blockHash common.Hash, blockNumber uint64, satetOverride *StateOverride) (*pCore.ExecutionResult, error) {

	// get Header
	header := a.b.GetHeader(blockHash, blockNumber)
	if header == nil {
		return nil, fmt.Errorf("header not found")
	}

	// get state
	state, err := a.b.StateAt(header.Root)
	if err != nil {
		return nil, err
	}

	if err := satetOverride.Apply(state); err != nil {
		return nil, err
	}

	msg := &pCore.Message{
		From:              txMsg.From,
		To:                txMsg.To,
		Value:             txMsg.Value,
		GasLimit:          txMsg.Gas,
		GasPrice:          txMsg.GasPrice,
		GasFeeCap:         txMsg.GasFeeCap,
		GasTipCap:         txMsg.GasTipCap,
		Data:              txMsg.Data,
		AccessList:        txMsg.AccessList,
		SkipAccountChecks: true,
	}

	// Get a new instance of the EVM.
	evm := vm_default.NewEVM(
		vm_default.NewEVMBlockContext(header, a.b, nil), //blockCtx
		vm_default.NewEVMTxContext(msg),                 //txCtx
		state,                                           //stateDb
		params.MainnetChainConfig,
		vm_default.Config{})

	// Wait for the context to be done and cancel the evm. Even if the
	// EVM has finished, cancelling may be done (repeatedly)
	go func() {
		<-ctx.Done()
		evm.Cancel()
	}()

	gp := new(pCore.GasPool).AddGas(math.MaxUint64)
	result, err := pCore.ApplyMessage(evm, msg, gp)
	if err := state.Error(); err != nil {
		return nil, err
	}

	if err != nil {
		return result, fmt.Errorf("err: %w (supplied gas %d)", err, msg.GasLimit)
	}
	return result, nil
}

*/
