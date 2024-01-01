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

package vm

import (
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/log"
	"github.com/holiman/uint256"
)

var (
	RestoreStackToLastSnap = "restoreStackToLastSnap"
)

// Ignores Gas consumption.
// Also returns stakc for inspection.

type ErrResp struct {
	ret []byte
	err error
}

type Debug struct {
	pc    uint64
	stack *Stack
}

type StackInterrupt struct {
	CatchVal     *uint256.Int // the Stack value to catch
	RplcWith     *uint256.Int // the Stack value to replace with
	DoSnapBefore bool         // whether to take a snapshot before the operation
	DoSnapAfter  bool         // whether to take a snapshot after the operation
}

func (in *EVMInterpreter) LiteRun(contract interface{}, input []byte, readOnly bool) (ret []byte, d interface{}, err error) {
	// Increment the call depth which is restricted to 1024
	in.evm.depth++
	defer func() { in.evm.depth-- }()

	// Make sure the readOnly is only set if we aren't in readOnly yet.
	// This also makes sure that the readOnly flag isn't removed for child calls.
	if readOnly && !in.readOnly {
		in.readOnly = true
		defer func() { in.readOnly = false }()
	}

	// Reset the previous call's return data. It's unimportant to preserve the old buffer
	// as every returning call will return new data anyway.
	in.returnData = nil

	// Don't bother with the execution if there's no code.
	if len(contract.(*Contract).Code) == 0 {
		return nil, nil, nil
	}

	stack := newstack() // local stack
	if in.enableStackHist {
		stack.EnableHistory(in.stackHistLen)
	}
	var (
		op  OpCode        // current opcode
		mem = NewMemory() // bound memory

		callContext = &ScopeContext{
			Memory:   mem,
			Stack:    stack,
			Contract: contract.(*Contract),
		}
		// For optimisation reason we're using uint64 as the program counter.
		// It's theoretically possible to go above 2^64. The YP defines the PC
		// to be uint256. Practically much less so feasible.
		pc  = uint64(0) // program counter
		res []byte      // result of the opcode execution function
	)
	// Don't move this deferred function, it's placed before the capturestate-deferred method,
	// so that it get's executed _after_: the capturestate needs the stacks before
	// they are returned to the pools
	defer func() {
		returnStack(stack)
	}()
	contract.(*Contract).Input = input

	// The Interpreter main run loop (contextual). This loop runs until either an
	// explicit STOP, RETURN or SELFDESTRUCT is executed, an error occurred during
	// the execution of one of the operations or until the done flag is set by the
	// parent context.
	for {
		// Get the operation from the jump table and validate the stack to ensure there are
		// enough stack items available to perform the operation.
		op = contract.(*Contract).GetOp(pc)
		operation := in.table[op]
		// Validate stack
		if sLen := stack.len(); sLen < operation.minStack {
			return nil, &Debug{pc: pc, stack: stack}, &ErrStackUnderflow{stackLen: sLen, required: operation.minStack}
		} else if sLen > operation.maxStack {
			return nil, &Debug{pc: pc, stack: stack}, &ErrStackOverflow{stackLen: sLen, limit: operation.maxStack}
		}
		if operation.dynamicGas != nil {
			// All ops with a dynamic memory usage also has a dynamic gas cost.
			var memorySize uint64
			// calculate the new memory size and expand the memory to fit
			// the operation
			// Memory check needs to be done prior to evaluating the dynamic gas portion,
			// to detect calculation overflows
			if operation.memorySize != nil {
				memSize, overflow := operation.memorySize(stack)
				if overflow {
					return nil, &Debug{pc: pc, stack: stack}, ErrGasUintOverflow
				}
				// memory is expanded in words of 32 bytes. Gas
				// is also calculated in words.
				if memorySize, overflow = math.SafeMul(toWordSize(memSize), 32); overflow {
					return nil, &Debug{pc: pc, stack: stack}, ErrGasUintOverflow
				}
			}
			if memorySize > 0 {
				mem.Resize(memorySize)
			}
		}

		// Trigger Here
		if in.stack_interrupt != nil {
			if stack.len() > 0 {
				if stack.peek().Cmp(in.stack_interrupt.CatchVal) == 0 {
					val := stack.peek()
					log.Info("Stack Interrupt", "pc", pc, "val", val.ToBig(), "catch", in.stack_interrupt.CatchVal.ToBig(), "rplc", in.stack_interrupt.RplcWith.ToBig())

					if in.stack_interrupt.DoSnapBefore {
						stack.Snapshot(pc) // create snapshot of stack at current program counter.
					}
					if in.stack_interrupt.RplcWith != nil {
						stack.pop()
						stack.push(in.stack_interrupt.RplcWith)
					}
					if in.stack_interrupt.DoSnapAfter {
						stack.Snapshot(pc) // create snapshot of stack at current program counter.
					}
				}
			}
		}

		// execute the operation
		res, err = operation.execute(&pc, in, callContext)
		if err != nil {
			break
		}
		pc++
	}

	if err == errStopToken {
		err = nil // clear stop token error
	}

	return res, &Debug{pc: pc, stack: stack}, err
}

func (in *EVMInterpreter) ScanFor(contract interface{}, input []byte, readOnly bool, value *uint256.Int) (loc uint64, res []byte, err error) {
	// Increment the call depth which is restricted to 1024
	in.evm.depth++
	defer func() { in.evm.depth-- }()

	// Make sure the readOnly is only set if we aren't in readOnly yet.
	// This also makes sure that the readOnly flag isn't removed for child calls.
	if readOnly && !in.readOnly {
		in.readOnly = true
		defer func() { in.readOnly = false }()
	}

	// Reset the previous call's return data. It's unimportant to preserve the old buffer
	// as every returning call will return new data anyway.
	in.returnData = nil

	// Don't bother with the execution if there's no code.
	if len(contract.(*Contract).Code) == 0 {
		return 0, nil, nil
	}

	stack := newstack() // local stack
	if in.enableStackHist {
		stack.EnableHistory(in.stackHistLen)
	}
	var (
		op  OpCode        // current opcode
		mem = NewMemory() // bound memory

		callContext = &ScopeContext{
			Memory:   mem,
			Stack:    stack,
			Contract: contract.(*Contract),
		}
		// For optimisation reason we're using uint64 as the program counter.
		// It's theoretically possible to go above 2^64. The YP defines the PC
		// to be uint256. Practically much less so feasible.
		pc = uint64(0) // program counter
	)
	// Don't move this deferred function, it's placed before the capturestate-deferred method,
	// so that it get's executed _after_: the capturestate needs the stacks before
	// they are returned to the pools
	defer func() {
		returnStack(stack)
	}()
	contract.(*Contract).Input = input

	// The Interpreter main run loop (contextual). This loop runs until either an
	// explicit STOP, RETURN or SELFDESTRUCT is executed, an error occurred during
	// the execution of one of the operations or until the done flag is set by the
	// parent context.
	for {
		// Get the operation from the jump table and validate the stack to ensure there are
		// enough stack items available to perform the operation.
		op = contract.(*Contract).GetOp(pc)
		operation := in.table[op]
		// Validate stack
		if sLen := stack.len(); sLen < operation.minStack {
			return 0, nil, &ErrStackUnderflow{stackLen: sLen, required: operation.minStack}
		} else if sLen > operation.maxStack {
			return 0, nil, &ErrStackOverflow{stackLen: sLen, limit: operation.maxStack}
		}
		if operation.dynamicGas != nil {
			// All ops with a dynamic memory usage also has a dynamic gas cost.
			var memorySize uint64
			// calculate the new memory size and expand the memory to fit
			// the operation
			// Memory check needs to be done prior to evaluating the dynamic gas portion,
			// to detect calculation overflows
			if operation.memorySize != nil {
				memSize, overflow := operation.memorySize(stack)
				if overflow {
					return 0, nil, ErrGasUintOverflow
				}
				// memory is expanded in words of 32 bytes. Gas
				// is also calculated in words.
				if memorySize, overflow = math.SafeMul(toWordSize(memSize), 32); overflow {
					return 0, nil, ErrGasUintOverflow
				}
			}
			if memorySize > 0 {
				mem.Resize(memorySize)
			}
		}

		// execute the operation
		res, err = operation.execute(&pc, in, callContext)
		// If the latest value on the stack is equal to the value we are looking for, then we have found it.
		// return PC and the result.
		if value != nil {
			if stack.len() > 0 {
				val := stack.peek()
				if val.Cmp(value) == 0 {
					loc = uint64(pc)
					return
				}
			}
		}

		if err != nil {
			break
		}

		pc++
	}

	if err == errStopToken {
		err = nil // clear stop token error
	}

	return
}

func (in *EVMInterpreter) ScanForMultiple(contract interface{}, input []byte, readOnly bool, values map[string]bool) (loc map[string][]uint64, res []byte, err error) {
	// Increment the call depth which is restricted to 1024
	in.evm.depth++
	defer func() { in.evm.depth-- }()

	loc = make(map[string][]uint64)

	// Make sure the readOnly is only set if we aren't in readOnly yet.
	// This also makes sure that the readOnly flag isn't removed for child calls.
	if readOnly && !in.readOnly {
		in.readOnly = true
		defer func() { in.readOnly = false }()
	}

	// Reset the previous call's return data. It's unimportant to preserve the old buffer
	// as every returning call will return new data anyway.
	in.returnData = nil

	// Don't bother with the execution if there's no code.
	if len(contract.(*Contract).Code) == 0 {
		return nil, nil, nil
	}

	stack := newstack() // local stack
	if in.enableStackHist {
		stack.EnableHistory(in.stackHistLen)
	}
	var (
		op  OpCode        // current opcode
		mem = NewMemory() // bound memory

		callContext = &ScopeContext{
			Memory:   mem,
			Stack:    stack,
			Contract: contract.(*Contract),
		}
		// For optimisation reason we're using uint64 as the program counter.
		// It's theoretically possible to go above 2^64. The YP defines the PC
		// to be uint256. Practically much less so feasible.
		pc = uint64(0) // program counter
	)
	// Don't move this deferred function, it's placed before the capturestate-deferred method,
	// so that it get's executed _after_: the capturestate needs the stacks before
	// they are returned to the pools
	defer func() {
		returnStack(stack)
	}()
	contract.(*Contract).Input = input

	// The Interpreter main run loop (contextual). This loop runs until either an
	// explicit STOP, RETURN or SELFDESTRUCT is executed, an error occurred during
	// the execution of one of the operations or until the done flag is set by the
	// parent context.
	for {
		// Get the operation from the jump table and validate the stack to ensure there are
		// enough stack items available to perform the operation.
		op = contract.(*Contract).GetOp(pc)
		operation := in.table[op]
		// Validate stack
		if sLen := stack.len(); sLen < operation.minStack {
			return nil, nil, &ErrStackUnderflow{stackLen: sLen, required: operation.minStack}
		} else if sLen > operation.maxStack {
			return nil, nil, &ErrStackOverflow{stackLen: sLen, limit: operation.maxStack}
		}
		if operation.dynamicGas != nil {
			// All ops with a dynamic memory usage also has a dynamic gas cost.
			var memorySize uint64
			// calculate the new memory size and expand the memory to fit
			// the operation
			// Memory check needs to be done prior to evaluating the dynamic gas portion,
			// to detect calculation overflows
			if operation.memorySize != nil {
				memSize, overflow := operation.memorySize(stack)
				if overflow {
					return nil, nil, ErrGasUintOverflow
				}
				// memory is expanded in words of 32 bytes. Gas
				// is also calculated in words.
				if memorySize, overflow = math.SafeMul(toWordSize(memSize), 32); overflow {
					return nil, nil, ErrGasUintOverflow
				}
			}
			if memorySize > 0 {
				mem.Resize(memorySize)
			}
		}

		// execute the operation
		res, err = operation.execute(&pc, in, callContext)
		// If the latest value on the stack is equal to the value we are looking for, then we have found it.
		// return PC and the result.
		if values != nil {
			if stack.len() > 0 {
				val := stack.peek()
				if _, ok := values[val.String()]; ok {
					loc[val.String()] = append(loc[val.String()], uint64(pc))
				}
			}
		}

		if err != nil {
			break
		}

		pc++
	}

	if err == errStopToken {
		err = nil // clear stop token error
	}

	return
}

type DebugCfg struct {
	TargetOpCodes map[string]bool // allows you to target multiple opcodes
	StackLen      int64
	MemOffset     int64
	MemSize       int64
	LastPcsLen    int
}

type CheckPoint struct {
	OpCode string
	pc     uint64
	Stack  []uint256.Int
	mem    []byte
}

type DebugRes struct {
	CheckPoints []CheckPoint
	LastPcs     []uint64
	LastOps     []string
	lastPcsLen  int
}

func newDebugRes(cfg *DebugCfg) *DebugRes {
	return &DebugRes{
		CheckPoints: make([]CheckPoint, 0),
		LastPcs:     make([]uint64, cfg.LastPcsLen),
		LastOps:     make([]string, cfg.LastPcsLen),
		lastPcsLen:  cfg.LastPcsLen,
	}
}

func (d *DebugRes) Add(cp *CheckPoint) {
	d.CheckPoints = append(d.CheckPoints, *cp)
}

func (d *DebugRes) UpdateLastPcs(pc uint64, op string) {
	if len(d.LastPcs) > d.lastPcsLen {
		d.LastPcs = d.LastPcs[1:]
		d.LastOps = d.LastOps[1:]
	}
	d.LastPcs = append(d.LastPcs, pc)
	d.LastOps = append(d.LastOps, op)
}

func newCp(opcCode string, pc uint64, stack *Stack, mem *Memory, cfg *DebugCfg) (cp *CheckPoint) {
	cp = &CheckPoint{
		pc:    pc,
		mem:   mem.GetCopy(cfg.MemOffset, cfg.MemSize),
		Stack: make([]uint256.Int, cfg.StackLen),
	}
	for i := 0; i < len(cp.Stack); i++ {
		cp.Stack[i] = *stack.Back(i)
	}
	return cp
}

type StackOverride struct {
	OverWith map[uint64]*uint256.Int
}

func (in *EVMInterpreter) RunWithOverride(contract interface{}, input []byte, readOnly bool, overrides interface{}, dCfg interface{}) (ret []byte, dRes interface{}, err error) {
	// Increment the call depth which is restricted to 1024
	in.evm.depth++
	defer func() { in.evm.depth-- }()

	// Make sure the readOnly is only set if we aren't in readOnly yet.
	// This also makes sure that the readOnly flag isn't removed for child calls.
	if readOnly && !in.readOnly {
		in.readOnly = true
		defer func() { in.readOnly = false }()
	}
	// Reset the previous call's return data. It's unimportant to preserve the old buffer
	// as every returning call will return new data anyway.
	in.returnData = nil

	// Don't bother with the execution if there's no code.
	if len(contract.(*Contract).Code) == 0 {
		return nil, nil, nil
	}

	stack := newstack() // local stack
	var (
		op  OpCode        // current opcode
		mem = NewMemory() // bound memory

		callContext = &ScopeContext{
			Memory:   mem,
			Stack:    stack,
			Contract: contract.(*Contract),
		}
		// For optimisation reason we're using uint64 as the program counter.
		// It's theoretically possible to go above 2^64. The YP defines the PC
		// to be uint256. Practically much less so feasible.
		pc  = uint64(0) // program counter
		res []byte      // result of the opcode execution function
	)

	if dCfg != nil {
		dRes = newDebugRes(dCfg.(*DebugCfg))
	}

	// Don't move this deferred function, it's placed before the capturestate-deferred method,
	// so that it get's executed _after_: the capturestate needs the stacks before
	// they are returned to the pools
	defer func() {
		returnStack(stack)
	}()
	contract.(*Contract).Input = input

	// The Interpreter main run loop (contextual). This loop runs until either an
	// explicit STOP, RETURN or SELFDESTRUCT is executed, an error occurred during
	// the execution of one of the operations or until the done flag is set by the
	// parent context.
	for {
		// Get the operation from the jump table and validate the stack to ensure there are
		// enough stack items available to perform the operation.
		op = contract.(*Contract).GetOp(pc)
		operation := in.table[op]

		// If we hit the Target Opcode
		// Store Stack & Mem Values for that OpCode
		if dCfg != nil {
			if _, ok := dCfg.(*DebugCfg).TargetOpCodes[op.String()]; ok {
				dRes.(*DebugRes).Add(newCp(op.String(), pc, stack, mem, dCfg.(*DebugCfg)))
			}
		}

		// Validate stack
		if sLen := stack.len(); sLen < operation.minStack {
			return nil, dRes, &ErrStackUnderflow{stackLen: sLen, required: operation.minStack}
		} else if sLen > operation.maxStack {
			return nil, dRes, &ErrStackOverflow{stackLen: sLen, limit: operation.maxStack}
		}
		if operation.dynamicGas != nil {
			// All ops with a dynamic memory usage also has a dynamic gas cost.
			var memorySize uint64
			// calculate the new memory size and expand the memory to fit
			// the operation
			// Memory check needs to be done prior to evaluating the dynamic gas portion,
			// to detect calculation overflows
			if operation.memorySize != nil {
				memSize, overflow := operation.memorySize(stack)
				if overflow {
					return nil, dRes, ErrGasUintOverflow
				}
				// memory is expanded in words of 32 bytes. Gas
				// is also calculated in words.
				if memorySize, overflow = math.SafeMul(toWordSize(memSize), 32); overflow {
					return nil, dRes, ErrGasUintOverflow
				}
			}
			if memorySize > 0 {
				mem.Resize(memorySize)
			}
		}

		// execute the operation
		res, err = operation.execute(&pc, in, callContext)
		if err != nil {
			break
		}

		// program counter matches, replace value in stack with specified value.
		if overrides != nil {
			if rplcValue, ok := overrides.(*StackOverride).OverWith[pc]; ok {
				stack.pop()
				stack.push(rplcValue)
			}
		}
		if dCfg != nil {
			dRes.(*DebugRes).UpdateLastPcs(pc, op.String())
		}
		pc++
	}

	if err == errStopToken {
		err = nil // clear stop token error
	}
	return res, dRes, err
}
