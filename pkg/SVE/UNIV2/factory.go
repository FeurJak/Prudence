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

type uniV2Factory struct {
	meth            map[string]abi.Method
	factoryAddr     common.Address
	callerAddr      common.Address
	factoryContract contractCode
	evm             evm
	interpreter     evmInterpreter
}

func (a *univ2API) NewUniV2Factory(factoryAddr common.Address, callerAddr common.Address) (interface{}, error) {
	factory := &uniV2Factory{
		factoryAddr: factoryAddr,
		callerAddr:  callerAddr,
		evm:         a.evm,
		interpreter: a.evm.Interpreter().(evmInterpreter),
	}

	addrCopy := factoryAddr
	factoryCode := a.evm.GetCode(factoryAddr)
	if factoryCode == nil {
		return nil, fmt.Errorf("token doesn't exist, no code at addr %s", factoryAddr.Hex())
	}

	// prep context for contract calls.
	factory.factoryContract = a.evm.NewContract(callerAddr, addrCopy, big.NewInt(0), 100000, factoryCode).(contractCode)

	// Prepare Method interface for contract
	// Potential to cache this.
	// Prepping ABI is expensive.
	factory.meth = make(map[string]abi.Method)
	if err := sveCommon.CookMeth(factory.meth); err != nil {
		return nil, fmt.Errorf("failed to pack erc20 methods %s", err.Error())
	}

	return factory, nil
}

func (t *uniV2Factory) GetPair(tA common.Address, tB common.Address) common.Address {
	if input, err := sveCommon.EncodeMethIn(t.meth, "getPair", []interface{}{tA, tB}); err == nil {
		if ret, _, err := t.interpreter.LiteRun(t.factoryContract, input, false); err == nil {
			if out, err := sveCommon.DecodeMethOut(t.meth, "getPair", ret); err == nil {
				return out[0].(common.Address)
			}
		}
	}
	return common.Address{}
}
