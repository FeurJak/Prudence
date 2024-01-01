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
	"github.com/pkg/errors"
)

type uniV2Router struct {
	meth        map[string]abi.Method
	routerAddr  common.Address
	callerAddr  common.Address
	contract    contractCode
	evm         evm
	interpreter evmInterpreter
}

func (a *univ2API) NewUniv2Router(routerAddr common.Address, callerAddr common.Address) (interface{}, error) {
	router := &uniV2Router{
		routerAddr:  routerAddr,
		callerAddr:  callerAddr,
		evm:         a.evm,
		interpreter: a.evm.Interpreter().(evmInterpreter),
	}

	addrCopy := routerAddr
	pairCode := a.evm.GetCode(routerAddr)
	if pairCode == nil {
		return nil, fmt.Errorf("token doesn't exist, no code at addr %s", routerAddr.Hex())
	}

	// prep context for contract calls.
	router.contract = a.evm.NewContract(callerAddr, addrCopy, big.NewInt(0), 100000, pairCode).(contractCode)

	// Prepare Method interface for contract
	// Potential to cache this.
	// Prepping ABI is expensive.
	router.meth = make(map[string]abi.Method)
	if err := sveCommon.CookMeth(router.meth); err != nil {
		return nil, fmt.Errorf("failed to pack erc20 methods %s", err.Error())
	}

	return router, nil
}

func (t *uniV2Router) RouterGetAmountOut(amountIn *big.Int, reserveIn *big.Int, reserveOut *big.Int) (amountOut *big.Int, err error) {
	var (
		input  []byte
		output []interface{}
		ret    []byte
	)
	if input, err = sveCommon.EncodeMethIn(t.meth, "getAmountOut", []interface{}{amountIn, reserveIn, reserveOut}); err == nil {
		if ret, _, err = t.interpreter.LiteRun(t.contract, input, false); err == nil {
			if output, err = sveCommon.DecodeMethOut(t.meth, "getAmountOut", ret); err == nil {
				return output[0].(*big.Int), err
			}
		} else {
			reason, _ := abi.UnpackRevert(ret)
			return nil, errors.Wrap(err, reason)
		}
	}
	return nil, err
}

// To-DO Implement SwapExactTokensForTokensSupportingFeeOnTransferTokens
func (t *uniV2Router) RouterSwapExactTokensForTokensSupportingFeeOnTransferTokens(amountIn *big.Int, amountOutMin *big.Int, path []common.Address, to common.Address, deadline *big.Int) error {
	var (
		input []byte
		err   error
		ret   []byte
	)

	if input, err = sveCommon.EncodeMethIn(t.meth, "swapExactTokensForTokensSupportingFeeOnTransferTokens", []interface{}{amountIn, amountOutMin, path, to, deadline}); err == nil {
		if ret, _, err = t.interpreter.LiteRun(t.contract, input, false); err == nil {
			return nil
		} else {
			reason, _ := abi.UnpackRevert(ret)
			return errors.Wrap(err, reason)
		}
	}
	return err
}
