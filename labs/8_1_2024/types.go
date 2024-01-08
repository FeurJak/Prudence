package main

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/sirupsen/logrus"
)

type TransferEvent struct {
	Index    *big.Int `json:"index"` // updated when inserted to DB
	Block    uint64   `json:"block"`
	TxHash   string   `json:"tx_hash"`
	LogIndex uint     `json:"log_index"`
	From     string   `json:"from"`
	To       string   `json:"to"`
	Amount   *big.Int `json:"amount"`
}

func NewTransferEvent(block uint64, txHash string, logIndex uint, from string, to string, amount *big.Int) *TransferEvent {
	return &TransferEvent{
		Block:    block,
		TxHash:   txHash,
		LogIndex: logIndex,
		From:     from,
		To:       to,
		Amount:   amount,
	}
}

func (t *TransferEvent) Bytes() ([]byte, error) {
	return json.Marshal(t)
}

func (t *TransferEvent) Key() []byte {
	key := fmt.Sprintf("%d:%s:%d", t.Block, t.TxHash, t.LogIndex)
	return []byte(key)
}

func (t *TransferEvent) LogFields() logrus.Fields {
	return logrus.Fields{
		"block":     t.Block,
		"tx_hash":   t.TxHash,
		"log_index": t.LogIndex,
		"from":      t.From,
		"to":        t.To,
		"amount":    t.Amount,
	}
}
