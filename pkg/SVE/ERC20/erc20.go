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

	"github.com/Prudence/pkg/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type execRef struct {
	poolBalancePc   uint64
	walletBalancePC uint64
	allowancePC     uint64
	tokenBalancePC  uint64
}

type evmInterpreter interface {
	LiteRun(contract interface{}, input []byte, readOnly bool) (ret []byte, d interface{}, err error)
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
	if err := cookERC20Meth(token.meth); err != nil {
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
	if input, err := EncodeMethIn(t.meth, "name", nil); err == nil {
		if ret, _, err := t.interpreter.LiteRun(t.tokenContract, input, false); err == nil {
			if out, err := DecodeMethOut(t.meth, "name", ret); err == nil {
				return out[0].(string)
			}
		}
	}
	return ""
}

func (t *erc20Token) symbol() string {
	if input, err := EncodeMethIn(t.meth, "symbol", nil); err == nil {
		if ret, _, err := t.interpreter.LiteRun(t.tokenContract, input, false); err == nil {
			if out, err := DecodeMethOut(t.meth, "symbol", ret); err == nil {
				return out[0].(string)
			}
		}
	}
	return ""
}

func (t *erc20Token) decimals() uint8 {
	if input, err := EncodeMethIn(t.meth, "decimals", nil); err == nil {
		if ret, _, err := t.interpreter.LiteRun(t.tokenContract, input, false); err == nil {
			if out, err := DecodeMethOut(t.meth, "decimals", ret); err == nil {
				return out[0].(uint8)
			}
		}
	}
	return 0
}

func (t *erc20Token) totalSupply() *big.Int {
	if input, err := EncodeMethIn(t.meth, "totalSupply", nil); err == nil {
		if ret, _, err := t.interpreter.LiteRun(t.tokenContract, input, false); err == nil {
			if out, err := DecodeMethOut(t.meth, "totalSupply", ret); err == nil {
				return out[0].(*big.Int)
			}
		}
	}
	return nil
}

func (t *erc20Token) owner() common.Address {
	if input, err := EncodeMethIn(t.meth, "owner", nil); err == nil {
		if ret, _, err := t.interpreter.LiteRun(t.tokenContract, input, false); err == nil {
			if out, err := DecodeMethOut(t.meth, "owner", ret); err == nil {
				return out[0].(common.Address)
			}
		}
	}

	if input, err := EncodeMethIn(t.meth, "getOwner", nil); err == nil {
		if ret, _, err := t.interpreter.LiteRun(t.tokenContract, input, false); err == nil {
			if out, err := DecodeMethOut(t.meth, "getOwner", ret); err == nil {
				return out[0].(common.Address)
			}
		}
	}
	return common.Address{}
}

func (t *erc20Token) balanceOf(addr common.Address) *big.Int {
	if input, err := EncodeMethIn(t.meth, "balanceOf", []interface{}{addr}); err == nil {
		if ret, _, err := t.interpreter.LiteRun(t.tokenContract, input, false); err == nil {
			if out, err := DecodeMethOut(t.meth, "balanceOf", ret); err == nil {
				return out[0].(*big.Int)
			}
		}
	}
	return nil
}

func (t *erc20Token) balanceOfUint256(addr common.Address) (balance *uint256.Int) {
	balance = uint256.NewInt(0)
	if input, err := EncodeMethIn(t.meth, "balanceOf", []interface{}{addr}); err == nil {
		if ret, _, err := t.interpreter.LiteRun(t.tokenContract, input, false); err == nil {
			balance.SetBytes(ret)
			return
		}
	}
	return nil
}

func (t *erc20Token) approve(target common.Address, amount *big.Int) (succes bool, err error) {
	if input, err := EncodeMethIn(t.meth, "approve", []interface{}{target, amount}); err == nil {
		if ret, _, err := t.interpreter.LiteRun(t.tokenContract, input, false); err == nil {
			if out, err := DecodeMethOut(t.meth, "approve", ret); err == nil {
				return out[0].(bool), nil
			}
		} else {
			//reason, _ := abi.UnpackRevert(ret)
			return false, err
		}
	}
	return false, err
}
