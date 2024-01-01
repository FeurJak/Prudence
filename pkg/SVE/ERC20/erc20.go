/*

          _____                    _____                    _____
         /\    \                  /\    \                  /\    \
        /::\    \                /::\____\                /::\    \
       /::::\    \              /:::/    /               /::::\    \
      /::::::\    \            /:::/    /               /::::::\    \
     /:::/\:::\    \          /:::/    /               /:::/\:::\    \
    /:::/__\:::\    \        /:::/____/               /:::/__\:::\    \
    \:::\   \:::\    \       |::|    |               /::::\   \:::\    \
  ___\:::\   \:::\    \      |::|    |     _____    /::::::\   \:::\    \
 /\   \:::\   \:::\    \     |::|    |    /\    \  /:::/\:::\   \:::\    \
/::\   \:::\   \:::\____\    |::|    |   /::\____\/:::/__\:::\   \:::\____\
\:::\   \:::\   \::/    /    |::|    |  /:::/    /\:::\   \:::\   \::/    /
 \:::\   \:::\   \/____/     |::|    | /:::/    /  \:::\   \:::\   \/____/
  \:::\   \:::\    \         |::|____|/:::/    /    \:::\   \:::\    \
   \:::\   \:::\____\        |:::::::::::/    /      \:::\   \:::\____\
    \:::\  /:::/    /        \::::::::::/____/        \:::\   \::/    /
     \:::\/:::/    /          ~~~~~~~~~~               \:::\   \/____/
      \::::::/    /                                     \:::\    \
       \::::/    /                                       \:::\____\
        \::/    /                                         \::/    /
         \/____/                                           \/____/

	- Sustainable Virtual Economies (SVE).
*/

package ERC20

import (
	"fmt"
	"math/big"
	"strings"

	sveCommon "github.com/Prudence/pkg/SVE/common"
	"github.com/Prudence/pkg/abi"
	vm "github.com/Prudence/pkg/vms/sve_vm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"
)

type execRef struct {
	poolBalancePc   uint64
	walletBalancePC uint64
	allowancePC     uint64
	tokenBalancePC  uint64
}

type debugCfg struct {
	TargetOpCodes map[string]bool // allows you to target multiple opcodes
	StackLen      int64
	MemOffset     int64
	MemSize       int64
	LastPcsLen    int
}

type checkPoint interface {
	StackValueAt(index int) *uint256.Int
}

type checkPoints interface {
	Len() int
	LastVal() interface{}
	ValueAt(index int) interface{}
}

type debugRes interface {
	CheckPoints() interface{}
}

type evmInterpreter interface {
	LiteRun(contract interface{}, input []byte, readOnly bool) (ret []byte, d interface{}, err error)
	ScanFor(contract interface{}, input []byte, readOnly bool, value *uint256.Int) (loc uint64, res []byte, err error)
	ScanForMultiple(contract interface{}, input []byte, readOnly bool, values map[string]bool) (loc map[string][]uint64, res []byte, err error)
	RunWithOverride(contract interface{}, input []byte, readOnly bool, overrides interface{}, dCfg interface{}) (ret []byte, dRes interface{}, err error)
}

type stackOverride struct {
	OverWith map[uint64]*uint256.Int
}

type contractCode interface {
	SetCallerAddress(callerAddr common.Address)
}

type erc20Token struct {
	meth          map[string]abi.Method
	tokenAddr     common.Address
	callerAddr    common.Address
	tokenContract contractCode
	evm           evm
	interpreter   evmInterpreter
	meta          *ERC20Meta
	reference     *execRef
}

func newErc20Token(tokenAddr common.Address, callerAddr common.Address, evm evm) (*erc20Token, error) {
	token := &erc20Token{
		tokenAddr:   tokenAddr,
		callerAddr:  callerAddr,
		evm:         evm,
		interpreter: evm.Interpreter().(evmInterpreter),
		meta:        &ERC20Meta{Addr: tokenAddr},
	}

	// Fetch Contract Code
	// Potential to TTL cache this.
	// Fetching code from state is expensive.
	addrCopy := tokenAddr
	tokenCode := evm.GetCode(tokenAddr)
	if tokenCode == nil {
		return nil, fmt.Errorf("token doesn't exist, no code at addr %s", tokenAddr.Hex())
	}

	// prep context for contract calls.
	token.tokenContract = evm.NewContract(callerAddr, addrCopy, big.NewInt(0), 100000, tokenCode).(contractCode)

	// Prepare Method interface for contract
	// Potential to cache this.
	// Prepping ABI is expensive.
	token.meth = make(map[string]abi.Method)
	if err := sveCommon.CookMeth(token.meth); err != nil {
		return nil, fmt.Errorf("failed to pack erc20 methods %s", err.Error())
	}

	return token, nil
}

func (t *erc20Token) Meta() *ERC20Meta {
	return t.meta
}

func (t *erc20Token) SetCallerAddress(callerAddr common.Address) {
	t.callerAddr = callerAddr
	t.tokenContract.SetCallerAddress(callerAddr)
}

func (t *erc20Token) name() string {
	if input, err := sveCommon.EncodeMethIn(t.meth, "name", nil); err == nil {
		if ret, _, err := t.interpreter.LiteRun(t.tokenContract, input, false); err == nil {
			if out, err := sveCommon.DecodeMethOut(t.meth, "name", ret); err == nil {
				return out[0].(string)
			}
		}
	}
	return ""
}

func (t *erc20Token) symbol() string {
	if input, err := sveCommon.EncodeMethIn(t.meth, "symbol", nil); err == nil {
		if ret, _, err := t.interpreter.LiteRun(t.tokenContract, input, false); err == nil {
			if out, err := sveCommon.DecodeMethOut(t.meth, "symbol", ret); err == nil {
				return out[0].(string)
			}
		}
	}
	return ""
}

func (t *erc20Token) decimals() uint8 {
	if input, err := sveCommon.EncodeMethIn(t.meth, "decimals", nil); err == nil {
		if ret, _, err := t.interpreter.LiteRun(t.tokenContract, input, false); err == nil {
			if out, err := sveCommon.DecodeMethOut(t.meth, "decimals", ret); err == nil {
				return out[0].(uint8)
			}
		}
	}
	return 0
}

func (t *erc20Token) totalSupply() *big.Int {
	if input, err := sveCommon.EncodeMethIn(t.meth, "totalSupply", nil); err == nil {
		if ret, _, err := t.interpreter.LiteRun(t.tokenContract, input, false); err == nil {
			if out, err := sveCommon.DecodeMethOut(t.meth, "totalSupply", ret); err == nil {
				return out[0].(*big.Int)
			}
		}
	}
	return nil
}

func (t *erc20Token) owner() common.Address {
	if input, err := sveCommon.EncodeMethIn(t.meth, "owner", nil); err == nil {
		if ret, _, err := t.interpreter.LiteRun(t.tokenContract, input, false); err == nil {
			if out, err := sveCommon.DecodeMethOut(t.meth, "owner", ret); err == nil {
				return out[0].(common.Address)
			}
		}
	}

	if input, err := sveCommon.EncodeMethIn(t.meth, "getOwner", nil); err == nil {
		if ret, _, err := t.interpreter.LiteRun(t.tokenContract, input, false); err == nil {
			if out, err := sveCommon.DecodeMethOut(t.meth, "getOwner", ret); err == nil {
				return out[0].(common.Address)
			}
		}
	}
	return common.Address{}
}

func (t *erc20Token) balanceOf(addr common.Address) *big.Int {
	if input, err := sveCommon.EncodeMethIn(t.meth, "balanceOf", []interface{}{addr}); err == nil {
		if ret, _, err := t.interpreter.LiteRun(t.tokenContract, input, false); err == nil {
			if out, err := sveCommon.DecodeMethOut(t.meth, "balanceOf", ret); err == nil {
				return out[0].(*big.Int)
			}
		}
	}
	return nil
}

func (t *erc20Token) balanceOfUint256(addr common.Address) (balance *uint256.Int) {
	balance = uint256.NewInt(0)
	if input, err := sveCommon.EncodeMethIn(t.meth, "balanceOf", []interface{}{addr}); err == nil {
		if ret, _, err := t.interpreter.LiteRun(t.tokenContract, input, false); err == nil {
			balance.SetBytes(ret)
			return
		}
	}
	return nil
}

func (t *erc20Token) approve(target common.Address, amount *big.Int) (succes bool, err error) {
	if input, err := sveCommon.EncodeMethIn(t.meth, "approve", []interface{}{target, amount}); err == nil {
		if ret, _, err := t.interpreter.LiteRun(t.tokenContract, input, false); err == nil {
			if out, err := sveCommon.DecodeMethOut(t.meth, "approve", ret); err == nil {
				return out[0].(bool), nil
			}
		} else {
			reason, _ := abi.UnpackRevert(ret)
			return false, errors.Wrap(err, reason)
		}
	}
	return false, err
}

func (t *erc20Token) ScanForBalance(address common.Address, method string, inputs []interface{}) (pc uint64, balanceVal *uint256.Int, reason string, err error) {
	balanceVal = t.balanceOfUint256(address)
	if input, err := sveCommon.EncodeMethIn(t.meth, method, inputs); err == nil {
		pc, res, err := t.interpreter.ScanFor(t.tokenContract, input, false, balanceVal)
		if err != nil {
			reason, errRevert := abi.UnpackRevert(res)
			if errRevert != nil {
				return pc, balanceVal, fmt.Sprintf("unable to unpack reason: %+v", common.Bytes2Hex(res)), err
			}
			return pc, balanceVal, reason, err
		}
		return pc, balanceVal, "", nil
	}
	return 0, nil, "", fmt.Errorf("failed to encode balanceOf")
}

func (t *erc20Token) allowanceUint256(owner common.Address, spender common.Address) (balance *uint256.Int, err error) {
	balance = uint256.NewInt(0)
	if input, err := sveCommon.EncodeMethIn(t.meth, "allowance", []interface{}{owner, spender}); err == nil {
		if ret, _, err := t.interpreter.LiteRun(t.tokenContract, input, false); err == nil {
			balance.SetBytes(ret)
			return balance, nil
		} else {
			reason, _ := abi.UnpackRevert(ret)
			return nil, errors.Wrap(err, "allowanceUint256 "+reason)
		}
	}
	return nil, nil
}
func (t *erc20Token) ScanForAllowance(owner common.Address, spender common.Address, method string, inputs []interface{}) (pc uint64, balanceVal *uint256.Int, reason string, err error) {
	balanceVal, err = t.allowanceUint256(owner, spender)
	if err != nil {
		return 0, nil, "", err
	}
	if input, err := sveCommon.EncodeMethIn(t.meth, method, inputs); err == nil {
		pc, res, err := t.interpreter.ScanFor(t.tokenContract, input, false, balanceVal)
		if err != nil {
			reason, errRevert := abi.UnpackRevert(res)
			if errRevert != nil {
				return pc, balanceVal, fmt.Sprintf("unable to unpack reason: %+v", common.Bytes2Hex(res)), err
			}
			return pc, balanceVal, reason, err
		}
		return pc, balanceVal, "", nil
	}
	return 0, nil, "", fmt.Errorf("failed to encode balanceOf")
}

func (t *erc20Token) TransferFrom(source common.Address, target common.Address, amount *big.Int) (succes bool, err error) {
	if input, err := sveCommon.EncodeMethIn(t.meth, "transferFrom", []interface{}{source, target, amount}); err == nil {
		if ret, _, err := t.interpreter.LiteRun(t.tokenContract, input, false); err == nil {
			if out, err := sveCommon.DecodeMethOut(t.meth, "transferFrom", ret); err == nil {
				return out[0].(bool), nil
			}
		} else {
			reason, _ := abi.UnpackRevert(ret)
			return false, errors.Wrap(err, reason)
		}
	}
	return false, err
}

func (t *erc20Token) TransferFromAndScan(source common.Address, target common.Address, amount *big.Int, values map[string]bool) (locations map[string][]uint64, reason string, err error) {
	if input, err := sveCommon.EncodeMethIn(t.meth, "transferFrom", []interface{}{source, target, amount}); err == nil {
		locations, res, err := t.interpreter.ScanForMultiple(t.tokenContract, input, false, values)
		if err != nil {
			reason, errRevert := abi.UnpackRevert(res)
			if errRevert != nil {
				return locations, fmt.Sprintf("unable to unpack reason: %+v", common.Bytes2Hex(res)), err
			}
			return locations, reason, err
		}
		return locations, "", nil
	}
	return nil, "", fmt.Errorf("failed to encode balanceOf")
}

func (t *erc20Token) detectBlockDelay(source common.Address, target common.Address) (err error) {

	input, err := sveCommon.EncodeMethIn(t.meth, "transferFrom", []interface{}{source, target, big.NewInt(1)})
	if err != nil {
		return errors.Wrap(err, "detectBlockDelay")
	}

	reason := ""
	ret, dRes, err := t.interpreter.RunWithOverride(t.tokenContract, input, false, nil, &vm.DebugCfg{
		TargetOpCodes: map[string]bool{"GT": true, "LT": true},
		StackLen:      2,
		MemOffset:     0,
		MemSize:       0,
	})
	if err != nil {
		reason, _ = abi.UnpackRevert(ret)
		return errors.Wrap(err, "detectBlockDelay "+reason)
	}

	// We only care about the last checkpoint.
	// if the revert was triggered, we accept the maxTxAmount to be the other value in the stack.
	t.meta.BlockDelay = big.NewInt(0)
	cps := dRes.(debugRes).CheckPoints().(checkPoints)
	len := cps.Len()

	for i := 0; i < len; i++ {
		cp := cps.ValueAt(i).(checkPoint)
		val_a := cp.StackValueAt(0).ToBig()
		val_b := cp.StackValueAt(1).ToBig()
		// blocknumber was used in a require condition somewhere..
		if val_a.Cmp(t.evm.BlockNumber()) == 0 || val_b.Cmp(t.evm.BlockNumber()) == 0 {
			t.meta.BlockDelayEnabled = true
			t.meta.BlockDelay = big.NewInt(2)
			return nil
		}

	}

	return nil
}

func (t *erc20Token) getMaxTxAmount(source common.Address, target common.Address) (err error) {

	testTransferAmount_bigInt, _ := big.NewInt(0).SetString("1000000000000000000000000000000000000000000000000000000000000000000000", 10)
	testTransferAmount_uint256 := uint256.NewInt(0)
	testTransferAmount_uint256.SetFromBig(testTransferAmount_bigInt)

	overrides := vm.StackOverride{
		OverWith: map[uint64]*uint256.Int{
			t.reference.allowancePC:     uint256.NewInt(0).Add(testTransferAmount_uint256, testTransferAmount_uint256),
			t.reference.walletBalancePC: uint256.NewInt(1),
		},
	}

	if t.reference.poolBalancePc != 0 {
		overrides.OverWith[t.reference.poolBalancePc] = testTransferAmount_uint256
	}

	// Trigger a revert by executing a TransferFrom() call
	// If a revert occurs, we want to trace what's in the stack when the GT OpCode was called.
	// The Transfer Amount would be a very large number.
	// This increases the probability of the transfer reverting due to MaxTx Amount.

	input, err := sveCommon.EncodeMethIn(t.meth, "transferFrom", []interface{}{source, target, testTransferAmount_bigInt})
	if err != nil {
		return errors.Wrap(err, "getMaxTxAmount")
	}

	t.evm.IncrementBlockNoBy(t.meta.BlockDelay)
	reason := ""
	ret, dRes, err := t.interpreter.RunWithOverride(t.tokenContract, input, false, &overrides, &vm.DebugCfg{
		TargetOpCodes: map[string]bool{"GT": true, "LT": true},
		StackLen:      2,
		MemOffset:     0,
		MemSize:       0,
	})
	if err != nil {
		reason, _ = abi.UnpackRevert(ret)
	}

	// NO Error = potentially no maxTxAmount is present.
	if err == nil || reason == "" || strings.Contains(strings.ToLower(reason), "insufficient allowance") || strings.Contains(strings.ToLower(reason), "subtraction overflow") || strings.Contains(strings.ToLower(reason), "multiplication overflow") {
		return nil
	}

	// We only care about the last checkpoint.
	checkpoints := dRes.(debugRes).CheckPoints().(checkPoints)

	last := checkpoints.LastVal().(checkPoint)
	// if the revert was triggered, we accept the maxTxAmount to be the other value in the stack.
	potentialMax := last.StackValueAt(1).ToBig()

	//We only want to set this as the maxTxAmount when its not larger than the TotalSupply
	if potentialMax.Cmp(t.meta.TotalSupply) < 0 {
		t.meta.MaxTxAmount = potentialMax
		t.meta.HasMaxTx = true
	}

	return nil
}

func (t *erc20Token) getMaxWallet(source common.Address, target common.Address) (err error) {

	// We then want to trigger a revert by executing a TransferFrom() call
	// If a revert occurs, we want to trace what's in the stack when the  GT OpCode was called.
	// The Wallet Size should be large to increase the probability of the transfer reverting due to MaxWalletSize Amount.
	testBalance_bigInt, _ := big.NewInt(0).SetString("1000000000000000000000000000000000000000000000000000000000000000000000", 10)
	testBalance_uint256 := uint256.NewInt(0)
	testBalance_uint256.SetFromBig(testBalance_bigInt)

	// prep input
	input, err := sveCommon.EncodeMethIn(t.meth, "transferFrom", []interface{}{source, target, big.NewInt(1)}) // Only Sending 1 unit.
	if err != nil {
		return errors.Wrap(err, "getMaxWallet")
	}

	overrides := vm.StackOverride{
		OverWith: map[uint64]*uint256.Int{
			t.reference.walletBalancePC: testBalance_uint256,
			t.reference.allowancePC:     testBalance_uint256,
		},
	}

	if t.reference.poolBalancePc != 0 {
		overrides.OverWith[t.reference.poolBalancePc] = testBalance_uint256
	}

	t.evm.IncrementBlockNoBy(t.meta.BlockDelay)
	reason := ""
	ret, dRes, err := t.interpreter.RunWithOverride(t.tokenContract, input, false, &overrides, &vm.DebugCfg{ // sppoofing the balance of the receiving wallet with testBalance_bigInt
		TargetOpCodes: map[string]bool{"GT": true, "LT": true},
		StackLen:      2,
		MemOffset:     0,
		MemSize:       0,
	})
	if err != nil {
		reason, _ = abi.UnpackRevert(ret)
	}

	// NO Error indicates that potentially no MaxWalletSize is present.
	if err == nil || reason == "" || strings.Contains(strings.ToLower(reason), "insufficient allowance") || strings.Contains(strings.ToLower(reason), "multiplication overflow") {
		return nil
	}

	// We only care about the last checkpoint.
	checkpoints := dRes.(debugRes).CheckPoints().(checkPoints)

	last := checkpoints.LastVal().(checkPoint)
	// if the revert was triggered, we accept the MaxWalletSize to be the other value in the stack.
	maxWallet := last.StackValueAt(1).ToBig()

	//only set the maxWalletSize when its less than totalSupply
	if maxWallet.Cmp(t.meta.TotalSupply) < 0 {
		t.meta.MaxWalletSize = maxWallet
		t.meta.HasMaxWallet = true
	}

	return nil
}
