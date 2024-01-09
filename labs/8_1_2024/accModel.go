package main

import (
	"encoding/json"
	"math/big"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type accTbl struct {
	tbl     ethdb.KeyValueStore
	syncReq chan struct{}
}

type AccTbl interface {
	GetCount() (*big.Int, error)
	GetAcc([]byte) (*AccModel, error)
	GetKey(*big.Int) ([]byte, error)
	InsertAcc(*AccModel, bool) (*big.Int, error)
	GetLatestAcc() (*AccModel, error)
}

func NewAccTbl(db ethdb.KeyValueStore, token string, address string) AccTbl {
	return &accTbl{
		tbl:     NewTable(db, token+"_"+address+"_"+accTbl_prefix),
		syncReq: make(chan struct{}, 3),
	}
}

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

/*
Account Models are stored with key of:
(token+"_"+address+"_"+accTbl_prefix)(event key)
*/
func (t *accTbl) GetAcc(key []byte) (*AccModel, error) {
	eventBytes, err := t.tbl.Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	event := &AccModel{}
	err = json.Unmarshal(eventBytes, event)
	if err != nil {
		return nil, errors.Wrap(err, "GetAcc Unmarshal")
	}
	return event, nil
}

// sets the value of (token+"_"+address+"_"+accTbl_prefix)(index) = key for the Acc model
// key for Acc model is just the Event key tied to the model.
func (t *accTbl) attachKey(index *big.Int, key []byte) error {
	return t.tbl.Put(index.Bytes(), key)
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

// Insert Account mmodel into DB
// Events are stored with key (txEventTbl_prefix)(event Key)
func (t *accTbl) InsertAcc(acc *AccModel, overWrite bool) (*big.Int, error) {
	// Check for previous tx event record
	prevEvent, err := t.GetAcc(acc.Key())
	if prevEvent != nil {
		if !overWrite {
			return nil, errors.New("InsertAcc: event already exists")
		}
		// make sure we inheret the previous event's index
		acc.Index = big.NewInt(0).Set(prevEvent.Index)
	} else {
		// if no previous event, increment the count
		acc.Index, err = t.upCount()
		if err != nil {
			return nil, errors.Wrap(err, "InsertAcc upCount")
		}

		// attach the new key to the index
		err = t.attachKey(acc.Index, acc.Key())
		if err != nil {
			return nil, errors.Wrap(err, "InsertAcc attachKey")
		}
	}

	eventBytes, err := acc.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "InsertAcc Bytes")
	}

	err = t.tbl.Put(acc.Key(), eventBytes)
	if err != nil {
		return nil, errors.Wrap(err, "InsertAcc Put")
	}

	return acc.Index, nil
}

func (t *accTbl) GetLatestAcc() (*AccModel, error) {
	count, err := t.GetCount()
	if err != nil {
		return nil, errors.Wrap(err, "GetLatestAcc GetCount")
	}

	// work backwards from the latest count and find one where the accoutn model exists
	for i := count.Int64(); i > 0; i-- {
		// get event key from index
		key, err := t.GetKey(big.NewInt(i))
		if err != nil {
			return nil, errors.Wrap(err, "GetLatestAcc GetKey")
		}

		// no key exists for this index
		// continue
		if key == nil {
			continue
		}

		// get model from key
		acc, err := t.GetAcc(key)
		if err != nil {
			return nil, errors.Wrap(err, "GetLatestAcc GetEvent")
		}

		if acc == nil {
			// signal to fix this
			t.syncReq <- struct{}{}
			continue
		}

		return acc, nil
	}
	return nil, nil
}

type accRegTbl struct {
	tbl     ethdb.KeyValueStore
	syncReq chan struct{}
}

type AccRegTbl interface {
	GetCount() (*big.Int, error)
	GetKey(*big.Int) ([]byte, error)
	RegAddr(addr []byte) (*big.Int, error)
	GetAllAddresses() ([]string, error)
}

func NewAccRegTbl(db ethdb.KeyValueStore, token string) AccRegTbl {
	return &accRegTbl{
		tbl:     NewTable(db, token+"_"+accRegTbl_prefix),
		syncReq: make(chan struct{}, 3),
	}
}

func (t *accRegTbl) upCount() (*big.Int, error) {
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

func (t *accRegTbl) GetCount() (*big.Int, error) {
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

func (t *accRegTbl) attachKey(index *big.Int, addr []byte) error {
	return t.tbl.Put(index.Bytes(), addr)
}

// returns the account address tied to the index
func (t *accRegTbl) GetKey(index *big.Int) ([]byte, error) {
	val, err := t.tbl.Get(index.Bytes())
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return val, nil
}

func (t *accRegTbl) RegAddr(addr []byte) (*big.Int, error) {
	ok, err := t.tbl.Has(addr)
	if err != nil {
		return nil, errors.Wrap(err, "GetCount Has")
	}
	// if addres exists, don't register it
	if ok {
		return nil, nil
	}

	// if address doesn't exist, increment the count
	index, err := t.upCount()
	if err != nil {
		return nil, errors.Wrap(err, "RegAddr upCount")
	}

	// attach key to index
	err = t.attachKey(index, addr)
	if err != nil {
		return nil, errors.Wrap(err, "RegAddr attachKey")
	}
	return index, nil
}

func (t *accRegTbl) GetAllAddresses() ([]string, error) {
	count, err := t.GetCount()
	if err != nil {
		return nil, errors.Wrap(err, "GetAllAddresses GetCount")
	}

	var addresses []string
	for i := int64(0); i <= count.Int64(); i++ {
		// get event key from index
		addr, err := t.GetKey(big.NewInt(i))
		if err != nil {
			return nil, errors.Wrap(err, "GetAllAddresses GetKey")
		}

		// no key exists for this index
		// continue
		if addr == nil {
			continue
		}

		addresses = append(addresses, string(addr))
	}
	return addresses, nil
}

type AccRegistry struct {
	token  string
	db     ethdb.KeyValueStore
	regTbl AccRegTbl
}

func NewAccRegistry(db ethdb.KeyValueStore, token string) *AccRegistry {
	return &AccRegistry{
		token:  token,
		db:     db,
		regTbl: NewAccRegTbl(db, token),
	}
}

func (ar *AccRegistry) updateAcc(address string, event *TransferEvent, eventBlock uint64, eventLogIndex uint) (*AccModel, error) {
	accTbl := NewAccTbl(ar.db, ar.token, address)

	// check the latest entry
	acc, err := accTbl.GetLatestAcc()
	if err != nil {
		return nil, errors.Wrap(err, "updateAcc GetLatestAcc")
	}

	// no records exists, create a new entry
	if acc == nil {
		acc = NewAccModel(address)
		// register new account in registry
		_, err := ar.regTbl.RegAddr([]byte(address))
		if err != nil {
			return nil, errors.Wrap(err, "updateAcc RegAddr")
		}
	} else {
		// compare the key with the new event key
		block, _, logIndex, err := DecodeTransferEventKey(acc.LastEventKey)
		if err != nil {
			return nil, errors.Wrap(err, "Transfer Key invalid")
		}

		// If latest acc entry is tied to an event that is newer than the current event
		// ignore update request since if we update an older entry we will need to update every proceeding entry afterwards.
		// we can figure this out at a later entry
		if block > eventBlock || (block == eventBlock && logIndex > eventLogIndex) {
			return nil, nil
		}
	}

	// update the latest entry
	acc.ProcessTransferEvent(event)
	_, err = accTbl.InsertAcc(acc, true)
	if err != nil {
		return nil, errors.Wrap(err, "updateAcc InsertAcc")
	}
	return acc, nil
}

func (ar *AccRegistry) ProcessTransferEvent(event *TransferEvent) (accs []*AccModel, err error) {

	block, _, logIndex, err := DecodeTransferEventKey(event.Key())
	if err != nil {
		return nil, errors.Wrap(err, "Transfer Key invalid")
	}

	for _, account := range []string{event.From, event.To} {
		acc, err := ar.updateAcc(account, event, block, logIndex)
		if err != nil {
			return nil, errors.Wrap(err, "ProcessTransferEvent updateAcc")
		}
		if acc != nil {
			accs = append(accs, acc)
		}
	}

	return accs, nil
}

func (ar *AccRegistry) LatestAccounts() (accs []*AccModel, err error) {
	// addresses are allready sorted by index
	// means addresses[0] earliest detected address.
	addresses, err := ar.regTbl.GetAllAddresses()

	for _, address := range addresses {
		accTbl := NewAccTbl(ar.db, ar.token, address)
		acc, err := accTbl.GetLatestAcc()
		if err != nil {
			return nil, errors.Wrap(err, "LatestAccounts GetLatestAcc")
		}
		if acc != nil {
			accs = append(accs, acc)
		}
	}
	return accs, nil
}

func (ar *AccRegistry) TotalAddress() (uint64, error) {
	count, err := ar.regTbl.GetCount()
	if err != nil {
		return 0, errors.Wrap(err, "TotalAddress GetCount")
	}
	return count.Uint64(), nil
}

type AccModel struct {
	Index        *big.Int `json:"index"` // updated when inserted to DB
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

func NewAccModel(address string) *AccModel {
	return &AccModel{
		Address:    address,
		Index:      big.NewInt(0),
		Balance:    big.NewInt(0),
		MaxBalance: big.NewInt(0),
		MinBalance: big.NewInt(0),
		Credit:     big.NewInt(0),
		Debit:      big.NewInt(0),
	}
}

func (a *AccModel) updateBalance(amount *big.Int, block uint64) {
	a.Balance = big.NewInt(0).Add(a.Balance, amount)

	if a.MaxBalance == nil || a.MaxBalance.Cmp(a.Balance) <= 0 {
		a.MaxBalance = big.NewInt(0).Set(a.Balance)
		a.MaxBalanceAt = block
	} else if a.MinBalance == nil || a.MinBalance.Cmp(a.Balance) >= 0 {
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
	} else if event.To == a.Address {
		a.CreditCount++
		a.Credit = big.NewInt(0).Add(a.Credit, amount)
	} else {
		panic("invalid transfer event")
	}

	if len(a.FistEventKey) == 0 {
		a.FistEventKey = append([]byte{}, event.Key()...)
	}
	a.LastEventKey = append([]byte{}, event.Key()...)

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
		"index":        a.Index,
		"FistEventKey": string(a.FistEventKey),
		"LastEventKey": string(a.LastEventKey),
		"Balance":      a.Balance,
	}
}

func (a *AccModel) GetFields() []string {
	return []string{
		"address",
		"index",
		"FistEventKey",
		"LastEventKey",
		"Balance",
		"MaxBalance",
		"MaxBalanceAt",
		"MinBalance",
		"MinBalanceAt",
		"Credit",
		"CreditCount",
		"Debit",
		"DebitCount",
	}
}

func (a *AccModel) Serialize() []string {
	return []string{
		a.Address,
		a.Index.String(),
		string(a.FistEventKey),
		string(a.LastEventKey),
		a.Balance.String(),
		a.MaxBalance.String(),
		big.NewInt(int64(a.MaxBalanceAt)).String(),
		a.MinBalance.String(),
		big.NewInt(int64(a.MinBalanceAt)).String(),
		a.Credit.String(),
		big.NewInt(int64(a.CreditCount)).String(),
		a.Debit.String(),
		big.NewInt(int64(a.DebitCount)).String(),
	}
}
