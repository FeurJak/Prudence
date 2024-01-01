package abi

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
)

// FunctionSpec is the parsed representation of a function's ABI.
type FunctionSpec struct {
	Name            string
	StateMutability string
	Constant        bool
	Payable         bool
	Inputs          []ArgumentMarshaling
	Outputs         []ArgumentMarshaling
}

func getMethod(spec *FunctionSpec) (Method, error) {
	var inputs Arguments
	var err error
	for _, arg := range spec.Inputs {
		input := Argument{Name: arg.Name, Indexed: arg.Indexed}
		input.Type, err = NewType(arg.Type, arg.InternalType, arg.Components)
		if err != nil {
			return Method{}, err
		}

		inputs = append(inputs, input)
	}
	var outputs Arguments
	for _, arg := range spec.Outputs {
		output := Argument{Name: arg.Name, Indexed: arg.Indexed}
		output.Type, err = NewType(arg.Type, arg.InternalType, arg.Components)
		if err != nil {
			return Method{}, err
		}

		outputs = append(outputs, output)
	}
	return NewMethod(spec.Name, spec.Name, Function, spec.StateMutability, spec.Constant, spec.Payable, inputs, outputs), nil
}

// PopulateMethods populates the map of methods with the given specs.
func PopulateMethods(specs map[string]*FunctionSpec, meths map[string]Method) (err error) {
	for meth, spec := range specs {
		method, err := getMethod(spec)
		if err != nil {
			return err
		}
		meths[meth] = method
	}
	return nil
}

// PackInputs packs the given inputs into a byte array.
func PackInputs(meths map[string]Method, funcName string, funcInputs ...interface{}) ([]byte, error) {
	var (
		meth Method
		ok   bool
	)
	if meth, ok = meths[funcName]; !ok {
		return nil, fmt.Errorf("method doesn't exist")
	}

	arguments, err := meth.Inputs.Pack(funcInputs...)
	if err != nil {
		return nil, fmt.Errorf("unable to pack inputs for moethod: %s", funcName)
	}
	return append(meth.ID, arguments...), nil

}

// revertSelector is a special function selector for revert reason unpacking.
var revertSelector = crypto.Keccak256([]byte("Error(string)"))[:4]

// UnpackRevert resolves the abi-encoded revert reason. According to the solidity
// spec https://solidity.readthedocs.io/en/latest/control-structures.html#revert,
// the provided revert reason is abi-encoded as if it were a call to a function
// `Error(string)`. So it's a special tool for it.
func UnpackRevert(data []byte) (string, error) {
	if len(data) < 4 || !bytes.Equal(data[:4], revertSelector) {
		return "", errors.New("invalid data for unpacking")
	}
	typ, _ := NewType("string", "", nil)
	unpacked, err := (Arguments{{Type: typ}}).Unpack(data[4:])
	if err != nil {
		return "", err
	}
	return unpacked[0].(string), nil
}
