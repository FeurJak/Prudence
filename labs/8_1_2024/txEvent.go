package main

import (
	"encoding/json"
	"math/big"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/pkg/errors"
)

type txEventTbl struct {
	tbl     ethdb.KeyValueStore
	syncReq chan struct{}
}

type TxEventTbl interface {
	GetKey(index *big.Int) ([]byte, error)
	GetEvent(key []byte) (*TransferEvent, error)
	InsertEvent(event *TransferEvent, overWrite bool) (*big.Int, error)
	GetCount() (*big.Int, error)
	GetLatestEvent() (*TransferEvent, error)
}

func NewTxEventTbl(db ethdb.KeyValueStore, token string) TxEventTbl {
	return &txEventTbl{
		tbl:     NewTable(db, token+"_"+txEventTbl_prefix),
		syncReq: make(chan struct{}, 3),
	}
}

// value of txEventTbl_prefix = number of tx events
func (t *txEventTbl) upCount() (*big.Int, error) {
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

func (t *txEventTbl) GetCount() (*big.Int, error) {
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

// sets the value of (txEventTbl_prefix)(index) = key for the Transaction Event
func (t *txEventTbl) attachKey(index *big.Int, txKey []byte) error {
	return t.tbl.Put(index.Bytes(), txKey)
}

func (t *txEventTbl) GetKey(index *big.Int) ([]byte, error) {
	val, err := t.tbl.Get(index.Bytes())
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return val, nil
}

func (t *txEventTbl) GetEvent(key []byte) (*TransferEvent, error) {
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
// Events are stored with key (txEventTbl_prefix)(event Key)
func (t *txEventTbl) InsertEvent(event *TransferEvent, overWrite bool) (*big.Int, error) {
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

func (t *txEventTbl) GetLatestEvent() (*TransferEvent, error) {
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
