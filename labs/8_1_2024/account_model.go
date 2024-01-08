package main

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/sirupsen/logrus"
)

type accTbl struct {
	tbl     ethdb.KeyValueStore
	syncReq chan struct{}
}

type AccTbl interface {
	/*
		GetKey(index *big.Int) ([]byte, error)
		GetEvent(key []byte) (*TransferEvent, error)
		InsertEvent(event *TransferEvent, overWrite bool) (*big.Int, error)
		GetCount() (*big.Int, error)
		GetLatestEvent() (*TransferEvent, error)
	*/
}

func NewaccTbl(db ethdb.KeyValueStore, token string, address string) AccTbl {
	return &accTbl{
		tbl:     NewTable(db, token+"_"+address+"_"+accTbl_prefix),
		syncReq: make(chan struct{}, 3),
	}
}

type AccModel struct {
	Address      string   `json:"address"`
	FistEventKey []byte   `json:"first_event_key"`
	LastEventKey []byte   `json:"last_event_key"`
	Balance      *big.Int `json:"balance"`
	MaxBalance   *big.Int `json:"max_balance"`
	MaxBalanceAt uint64   `json:"max_balance_at"`
	MinBalance   *big.Int `json:"min_balance"`
	MinBalanceAt uint64   `json:"min_balance_at"`
	Credit       *big.Int `json:"credit"`
	CreditCount  uint64   `json:"credit_count"`
	Debit        *big.Int `json:"debit"`
	DebitCount   uint64   `json:"debit_count"`
}

func (a *AccModel) updateBalance(amount *big.Int, block uint64) {
	a.Balance = big.NewInt(0).Add(a.Balance, amount)

	if a.MaxBalance == nil || a.MaxBalance.Cmp(a.Balance) < 0 {
		a.MaxBalance = big.NewInt(0).Set(a.Balance)
		a.MaxBalanceAt = block
	} else if a.MinBalance == nil || a.MinBalance.Cmp(a.Balance) > 0 {
		a.MinBalance = big.NewInt(0).Set(a.Balance)
		a.MinBalanceAt = block
	}
}

func (a *AccModel) ProcessTransferEvent(event *TransferEvent) {
	amount := big.NewInt(0).Set(event.Amount)
	// if account is dst then debit
	if event.To != a.Address {
		a.Debit = big.NewInt(0).Add(a.Debit, amount)
		a.DebitCount++
		amount = big.NewInt(0).Neg(amount)
	} else {
		a.CreditCount++
		a.Credit = big.NewInt(0).Add(a.Credit, amount)
	}

	if len(a.FistEventKey) == 0 {
		copy(a.FistEventKey, event.Key())
	} else {
		copy(a.LastEventKey, event.Key())
	}

	a.updateBalance(amount, event.Block)
}

func (a *AccModel) Bytes() ([]byte, error) {
	return json.Marshal(a)
}

func (a *AccModel) Key() []byte {
	return a.LastEventKey // account key is tied to event key
}

func (a *AccModel) LogFields() logrus.Fields {
	return logrus.Fields{
		"address":      a.Address,
		"FistEventKey": string(a.FistEventKey),
		"LastEventKey": string(a.LastEventKey),
		"Balance":      a.Balance,
	}
}

/*
func (a *AccModel) Set(acc *AccModel) *AccModel {
	a.Address = acc.Address
	a.FistEventKey = acc.FistEventKey
	a.LastEventKey = acc.LastEventKey
	a.Balance = big.NewInt(0).Set(acc.Balance)
	a.MaxBalance = big.NewInt(0).Set(acc.MaxBalance)
	a.MaxBalanceAt = acc.MaxBalanceAt
	a.MinBalance = big.NewInt(0).Set(acc.MinBalance)
	a.MinBalanceAt = acc.MinBalanceAt
	a.Credit = big.NewInt(0).Set(acc.Credit)
	a.CreditCount = acc.CreditCount
	a.Debit = big.NewInt(0).Set(acc.Debit)
	a.DebitCount = acc.DebitCount
}

type AccRegistry struct {
	tbl      AccTbl
	Accounts map[string]*AccModel
}

func NewAccRegistry(tbl AccTbl) *AccRegistry {
	return &AccRegistry{
		tbl:      tbl,
		Accounts: make(map[string]*AccModel),
	}
}

type accTbl struct {
	tbl     ethdb.KeyValueStore
	syncReq chan struct{}
}

type AccTbl interface {
	GetKey(index *big.Int) ([]byte, error)
	GetEvent(key []byte) (*TransferEvent, error)
	InsertEvent(event *TransferEvent, overWrite bool) (*big.Int, error)
	GetCount() (*big.Int, error)
	GetLatestEvent() (*TransferEvent, error)
}

func NewaccTbl(db ethdb.KeyValueStore, token string) AccTbl {
	return &accTbl{
		tbl:     NewTable(db, token+"_"+accTbl_prefix),
		syncReq: make(chan struct{}, 3),
	}
}

// value of accTbl_prefix = number of tx events
func (t *accTbl) upCount() (*big.Int, error) {
	ok, err := t.tbl.Has([]byte{})
	if err != nil {
		return nil, errors.Wrap(err, "upCount Has")
	}
	count := big.NewInt(0)
	if ok {
		val, err := t.tbl.Get([]byte{})
		if err != nil {
			return nil, errors.Wrap(err, "upCount Get")
		}
		count = big.NewInt(0).SetBytes(val)
	}
	err = t.tbl.Put([]byte{}, count.Add(count, big.NewInt(1)).Bytes())
	if err != nil {
		return nil, errors.Wrap(err, "upCount Put")
	}
	return count, nil
}

func (t *accTbl) GetCount() (*big.Int, error) {
	ok, err := t.tbl.Has([]byte{})
	if err != nil {
		return nil, errors.Wrap(err, "GetCount Has")
	}
	count := big.NewInt(0)
	if ok {
		val, err := t.tbl.Get([]byte{})
		if err != nil {
			return nil, errors.Wrap(err, "GetCount Get")
		}
		count = big.NewInt(0).SetBytes(val)
	}
	return count, nil
}

// sets the value of (accTbl_prefix)(index) = key for the Transaction Event
func (t *accTbl) attachKey(index *big.Int, txKey []byte) error {
	return t.tbl.Put(index.Bytes(), txKey)
}

func (t *accTbl) GetKey(index *big.Int) ([]byte, error) {
	val, err := t.tbl.Get(index.Bytes())
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return val, nil
}

func (t *accTbl) GetEvent(key []byte) (*TransferEvent, error) {
	eventBytes, err := t.tbl.Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	event := &TransferEvent{}
	err = json.Unmarshal(eventBytes, event)
	if err != nil {
		return nil, errors.Wrap(err, "GetEvent Unmarshal")
	}
	return event, nil
}

// Insert Tx Event into DB
// Events are stored with key (accTbl_prefix)(event Key)
func (t *accTbl) InsertEvent(event *TransferEvent, overWrite bool) (*big.Int, error) {
	// Check for previous tx event record
	prevEvent, err := t.GetEvent(event.Key())
	if prevEvent != nil {
		if !overWrite {
			return nil, errors.New("InsertEvent: event already exists")
		}
		// make sure we inheret the previous event's index
		event.Index = big.NewInt(0).Set(prevEvent.Index)
	} else {
		// if no previous event, increment the count
		event.Index, err = t.upCount()
		if err != nil {
			return nil, errors.Wrap(err, "InsertEvent upCount")
		}

		// attach the new key to the index
		err = t.attachKey(event.Index, event.Key())
		if err != nil {
			return nil, errors.Wrap(err, "InsertEvent attachKey")
		}
	}

	eventBytes, err := event.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "InsertEvent Bytes")
	}

	err = t.tbl.Put(event.Key(), eventBytes)
	if err != nil {
		return nil, errors.Wrap(err, "InsertEvent Put")
	}

	return event.Index, nil
}

func (t *accTbl) GetLatestEvent() (*TransferEvent, error) {
	count, err := t.GetCount()
	if err != nil {
		return nil, errors.Wrap(err, "GetLatestEvent GetCount")
	}

	// work backwards from the latest count and find one where the event exists
	for i := count.Int64(); i > 0; i-- {
		// get event key from index
		key, err := t.GetKey(big.NewInt(i))
		if err != nil {
			return nil, errors.Wrap(err, "GetLatestEvent GetKey")
		}

		// no key exists for this index
		// continue
		if key == nil {
			continue
		}

		// get event from key
		event, err := t.GetEvent(key)
		if err != nil {
			return nil, errors.Wrap(err, "GetLatestEvent GetEvent")
		}

		if event == nil {
			// signal to fix this
			t.syncReq <- struct{}{}
			continue
		}

		return event, nil
	}
	return nil, nil
}
*/
