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

	"github.com/Prudence/pkg/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func cookERC20Meth(meth map[string]abi.Method) (err error) {

	defJam := map[string]*abi.FunctionSpec{
		"WETH": {
			Name:            "weth",
			StateMutability: "nonpayable",
			Constant:        false,
			Payable:         false,
			Outputs: []abi.ArgumentMarshaling{
				{
					Name:         "",
					Type:         "address",
					InternalType: "address",
					Components:   nil,
					Indexed:      false,
				},
			},
		},
		"balanceOf": {
			Name:            "balanceOf",
			StateMutability: "nonpayable",
			Constant:        false,
			Payable:         false,
			Inputs: []abi.ArgumentMarshaling{
				{
					Name:         "account",
					Type:         "address",
					InternalType: "address",
					Components:   nil,
					Indexed:      false,
				},
			},
			Outputs: []abi.ArgumentMarshaling{
				{
					Name:         "",
					Type:         "uint256",
					InternalType: "uint256",
					Components:   nil,
					Indexed:      false,
				},
			},
		},
		"allowance": {
			Name:            "allowance",
			StateMutability: "nonpayable",
			Constant:        false,
			Payable:         false,
			Inputs: []abi.ArgumentMarshaling{
				{
					Name:         "owner",
					Type:         "address",
					InternalType: "address",
					Components:   nil,
					Indexed:      false,
				},
				{
					Name:         "spender",
					Type:         "address",
					InternalType: "address",
					Components:   nil,
					Indexed:      false,
				},
			},
			Outputs: []abi.ArgumentMarshaling{
				{
					Name:         "",
					Type:         "uint256",
					InternalType: "uint256",
					Components:   nil,
					Indexed:      false,
				},
			},
		},
		"name": {
			Name:            "name",
			StateMutability: "nonpayable",
			Constant:        false,
			Payable:         false,
			Outputs: []abi.ArgumentMarshaling{
				{
					Name:         "",
					Type:         "string",
					InternalType: "string",
					Components:   nil,
					Indexed:      false,
				},
			},
		},
		"symbol": {
			Name:            "symbol",
			StateMutability: "nonpayable",
			Constant:        false,
			Payable:         false,
			Outputs: []abi.ArgumentMarshaling{
				{
					Name:         "",
					Type:         "string",
					InternalType: "string",
					Components:   nil,
					Indexed:      false,
				},
			},
		},
		"decimals": {
			Name:            "decimals",
			StateMutability: "nonpayable",
			Constant:        false,
			Payable:         false,
			Outputs: []abi.ArgumentMarshaling{
				{
					Name:         "",
					Type:         "uint8",
					InternalType: "uint8",
					Components:   nil,
					Indexed:      false,
				},
			},
		},
		"totalSupply": {
			Name:            "totalSupply",
			StateMutability: "nonpayable",
			Constant:        false,
			Payable:         false,
			Outputs: []abi.ArgumentMarshaling{
				{
					Name:         "",
					Type:         "uint256",
					InternalType: "uint256",
					Components:   nil,
					Indexed:      false,
				},
			},
		},
		"owner": {
			Name:            "owner",
			StateMutability: "nonpayable",
			Constant:        false,
			Payable:         false,
			Outputs: []abi.ArgumentMarshaling{
				{
					Name:         "",
					Type:         "address",
					InternalType: "address",
					Components:   nil,
					Indexed:      false,
				},
			},
		},
		"getOwner": {
			Name:            "getOwner",
			StateMutability: "nonpayable",
			Constant:        false,
			Payable:         false,
			Outputs: []abi.ArgumentMarshaling{
				{
					Name:         "",
					Type:         "address",
					InternalType: "address",
					Components:   nil,
					Indexed:      false,
				},
			},
		},
		"token0": {
			Name:            "token0",
			StateMutability: "nonpayable",
			Constant:        false,
			Payable:         false,
			Outputs: []abi.ArgumentMarshaling{
				{
					Name:         "",
					Type:         "address",
					InternalType: "address",
					Components:   nil,
					Indexed:      false,
				},
			},
		},
		// Uni Router Functions
		"getAmountOut": {
			Name:            "getAmountOut",
			StateMutability: "pure",
			Constant:        false,
			Payable:         false,
			Inputs: []abi.ArgumentMarshaling{
				{
					Name:         "amountIn",
					Type:         "uint256",
					InternalType: "uint256",
					Components:   nil,
					Indexed:      false,
				},
				{
					Name:         "reserveIn",
					Type:         "uint256",
					InternalType: "uint256",
					Components:   nil,
					Indexed:      false,
				},
				{
					Name:         "reserveOut",
					Type:         "uint256",
					InternalType: "uint256",
					Components:   nil,
					Indexed:      false,
				},
			},
			Outputs: []abi.ArgumentMarshaling{
				{
					Name:         "amountOut",
					Type:         "uint256",
					InternalType: "uint256",
					Components:   nil,
					Indexed:      false,
				},
			},
		},
		"swapExactTokensForTokensSupportingFeeOnTransferTokens": {
			Name:            "swapExactTokensForTokensSupportingFeeOnTransferTokens",
			StateMutability: "nonpayable",
			Constant:        false,
			Payable:         false,
			Inputs: []abi.ArgumentMarshaling{
				{
					Name:         "amountIn",
					Type:         "uint256",
					InternalType: "uint256",
					Components:   nil,
					Indexed:      false,
				},
				{
					Name:         "amountOutMin",
					Type:         "uint256",
					InternalType: "uint256",
					Components:   nil,
					Indexed:      false,
				},
				{
					Name:         "path",
					Type:         "address[]",
					InternalType: "address[]",
					Components:   nil,
					Indexed:      false,
				},
				{
					Name:         "to",
					Type:         "address",
					InternalType: "address",
					Components:   nil,
					Indexed:      false,
				},
				{
					Name:         "deadline",
					Type:         "uint256",
					InternalType: "uint256",
					Components:   nil,
					Indexed:      false,
				},
			},
			Outputs: []abi.ArgumentMarshaling{
				{
					Name:         "amountOut",
					Type:         "uint256",
					InternalType: "uint256",
					Components:   nil,
					Indexed:      false,
				},
			},
		},
		// UniV2 Functions
		"getPair": {
			Name:            "getPair",
			StateMutability: "nonpayable",
			Constant:        false,
			Payable:         false,
			Inputs: []abi.ArgumentMarshaling{
				{
					Name:         "",
					Type:         "address",
					InternalType: "address",
					Components:   nil,
					Indexed:      false,
				},
				{
					Name:         "",
					Type:         "address",
					InternalType: "address",
					Components:   nil,
					Indexed:      false,
				},
			},
			Outputs: []abi.ArgumentMarshaling{
				{
					Name:         "",
					Type:         "address",
					InternalType: "address",
					Components:   nil,
					Indexed:      false,
				},
			},
		},
		"sync": {
			Name:            "sync",
			StateMutability: "nonpayable",
			Constant:        false,
			Payable:         false,
			Outputs:         nil,
		},
		"token1": {
			Name:            "token1",
			StateMutability: "nonpayable",
			Constant:        false,
			Payable:         false,
			Outputs: []abi.ArgumentMarshaling{
				{
					Name:         "",
					Type:         "address",
					InternalType: "address",
					Components:   nil,
					Indexed:      false,
				},
			},
		},
		"getReserves": {
			Name:            "getReserves",
			StateMutability: "nonpayable",
			Constant:        false,
			Payable:         false,
			Outputs: []abi.ArgumentMarshaling{
				{
					Name:         "_reserve0",
					Type:         "uint112",
					InternalType: "uint112",
					Components:   nil,
					Indexed:      false,
				},
				{
					Name:         "_reserve1",
					Type:         "uint112",
					InternalType: "uint112",
					Components:   nil,
					Indexed:      false,
				},
				{
					Name:         "_blockTimestampLast",
					Type:         "uint32",
					InternalType: "uint32",
					Components:   nil,
					Indexed:      false,
				},
			},
		},
		"transfer": {
			Name:            "transfer",
			StateMutability: "nonpayable",
			Constant:        false,
			Payable:         false,
			Inputs: []abi.ArgumentMarshaling{
				{
					Name:         "to",
					Type:         "address",
					InternalType: "address",
					Components:   nil,
					Indexed:      false,
				},
				{
					Name:         "value",
					Type:         "uint256",
					InternalType: "uint256",
					Components:   nil,
					Indexed:      false,
				},
			},
			Outputs: []abi.ArgumentMarshaling{
				{
					Name:         "",
					Type:         "bool",
					InternalType: "bool",
					Components:   nil,
					Indexed:      false,
				},
			},
		},
		"approve": {
			Name:            "approve",
			StateMutability: "nonpayable",
			Constant:        false,
			Payable:         false,
			Inputs: []abi.ArgumentMarshaling{
				{
					Name:         "spender",
					Type:         "address",
					InternalType: "address",
					Components:   nil,
					Indexed:      false,
				},
				{
					Name:         "amount",
					Type:         "uint256",
					InternalType: "uint256",
					Components:   nil,
					Indexed:      false,
				},
			},
			Outputs: []abi.ArgumentMarshaling{
				{
					Name:         "",
					Type:         "bool",
					InternalType: "bool",
					Components:   nil,
					Indexed:      false,
				},
			},
		},
		"transferFrom": {
			Name:            "transferFrom",
			StateMutability: "nonpayable",
			Constant:        false,
			Payable:         false,
			Inputs: []abi.ArgumentMarshaling{
				{
					Name:         "sender",
					Type:         "address",
					InternalType: "address",
					Components:   nil,
					Indexed:      false,
				},
				{
					Name:         "recipient",
					Type:         "address",
					InternalType: "address",
					Components:   nil,
					Indexed:      false,
				},
				{
					Name:         "amount",
					Type:         "uint256",
					InternalType: "uint256",
					Components:   nil,
					Indexed:      false,
				},
			},
			Outputs: []abi.ArgumentMarshaling{
				{
					Name:         "",
					Type:         "bool",
					InternalType: "bool",
					Components:   nil,
					Indexed:      false,
				},
			},
		},
	}

	return abi.PopulateMethods(defJam, meth)
}

func EncodeMethIn(meth map[string]abi.Method, funcID string, inputs []interface{}) (in []byte, err error) {
	var arguments []byte
	if meth, ok := meth[funcID]; ok {
		if arguments, err = meth.Inputs.Pack(inputs...); err == nil {
			return append(meth.ID, arguments...), nil
		}
	} else {
		err = fmt.Errorf("funcID %s doesn't exist", funcID)
	}
	return nil, fmt.Errorf("failed to pack input for %s err: %+v", funcID, err)
}

func DecodeMethOut(meth map[string]abi.Method, funcID string, output []byte) (ret []interface{}, err error) {
	if meth, ok := meth[funcID]; ok {
		if len(output)%32 != 0 {
			return nil, fmt.Errorf("{decodeMethOut} abi: improperly formatted output: %s - Bytes: [%+v]", hexutil.Encode(output), output)
		}
		return meth.Outputs.Unpack(output)
	}
	return nil, fmt.Errorf("{decodeMethOut} abi: could not locate named method or event: %s", funcID)
}
