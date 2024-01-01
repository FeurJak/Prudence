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

package UNIV2

import (
	"fmt"
	"math/big"

	sveCommon "github.com/Prudence/pkg/SVE/common"
	"github.com/Prudence/pkg/abi"
	"github.com/ethereum/go-ethereum/common"
)

type uniV2Pair struct {
	meth        map[string]abi.Method
	pairAddr    common.Address
	callerAddr  common.Address
	contract    contractCode
	evm         evm
	interpreter evmInterpreter
}

func (a *univ2API) NewUniV2Pair(pairAddr common.Address, callerAddr common.Address) (interface{}, error) {
	pair := &uniV2Pair{
		pairAddr:    pairAddr,
		callerAddr:  callerAddr,
		evm:         a.evm,
		interpreter: a.evm.Interpreter().(evmInterpreter),
	}

	addrCopy := pairAddr
	pairCode := a.evm.GetCode(pairAddr)
	if pairCode == nil {
		return nil, fmt.Errorf("token doesn't exist, no code at addr %s", pairAddr.Hex())
	}

	// prep context for contract calls.
	pair.contract = a.evm.NewContract(callerAddr, addrCopy, big.NewInt(0), 100000, pairCode).(contractCode)

	// Prepare Method interface for contract
	// Potential to cache this.
	// Prepping ABI is expensive.
	pair.meth = make(map[string]abi.Method)
	if err := sveCommon.CookMeth(pair.meth); err != nil {
		return nil, fmt.Errorf("failed to pack erc20 methods %s", err.Error())
	}

	return pair, nil
}

func (t *uniV2Pair) Token0() common.Address {
	if input, err := sveCommon.EncodeMethIn(t.meth, "token0", nil); err == nil {
		if ret, _, err := t.interpreter.LiteRun(t.contract, input, false); err == nil {
			if out, err := sveCommon.DecodeMethOut(t.meth, "token0", ret); err == nil {
				return out[0].(common.Address)
			}
		}
	}
	return common.Address{}
}

func (t *uniV2Pair) GetReserves() (*big.Int, *big.Int) {
	if input, err := sveCommon.EncodeMethIn(t.meth, "getReserves", nil); err == nil {
		if ret, _, err := t.interpreter.LiteRun(t.contract, input, false); err == nil {
			if out, err := sveCommon.DecodeMethOut(t.meth, "getReserves", ret); err == nil {
				return out[0].(*big.Int), out[1].(*big.Int)
			}
		}
	}
	return nil, nil
}

func (t *uniV2Pair) Sync() (err error) {
	var (
		input []byte
		ret   []byte
	)
	if input, err = sveCommon.EncodeMethIn(t.meth, "sync", nil); err == nil {
		if ret, _, err = t.interpreter.LiteRun(t.contract, input, false); err == nil {
			if _, err = sveCommon.DecodeMethOut(t.meth, "sync", ret); err == nil {
				return nil
			}
		}
	}
	return err
}
