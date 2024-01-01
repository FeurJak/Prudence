package eth

import "fmt"

/*
Dummy handlers
*/
func handleNewBlockhashes(backend Backend, msg Decoder, peer *Peer) error {
	return nil
}

func handleNewBlock(backend Backend, msg Decoder, peer *Peer) error {
	return nil
}

func handleGetBlockHeaders(backend Backend, msg Decoder, peer *Peer) error {
	return nil
}

func handleBlockHeaders(backend Backend, msg Decoder, peer *Peer) error {
	return nil
}

func handleGetBlockBodies(backend Backend, msg Decoder, peer *Peer) error {
	return nil
}

func handleBlockBodies(backend Backend, msg Decoder, peer *Peer) error {
	return nil
}

func handleGetReceipts(backend Backend, msg Decoder, peer *Peer) error {
	return nil
}

func handleReceipts(backend Backend, msg Decoder, peer *Peer) error {
	return nil
}

func handleTrahandleGetPooledTransactionsnsactions(backend Backend, msg Decoder, peer *Peer) error {
	return nil
}

func handleGetPooledTransactions(backend Backend, msg Decoder, peer *Peer) error {
	return nil
}

func handleNewPooledTransactionHashes67(backend Backend, msg Decoder, peer *Peer) error {
	ann := new(NewPooledTransactionHashesPacket67)
	if err := msg.Decode(ann); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	// Schedule all the unknown hashes for retrieval
	for _, hash := range *ann {
		peer.markTransaction(hash)
	}
	return backend.Handle(peer, ann)
}

func handleNewPooledTransactionHashes68(backend Backend, msg Decoder, peer *Peer) error {
	ann := new(NewPooledTransactionHashesPacket68)
	if err := msg.Decode(ann); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	if len(ann.Hashes) != len(ann.Types) || len(ann.Hashes) != len(ann.Sizes) {
		return fmt.Errorf("%w: message %v: invalid len of fields: %v %v %v", errDecode, msg, len(ann.Hashes), len(ann.Types), len(ann.Sizes))
	}
	// Schedule all the unknown hashes for retrieval
	for _, hash := range ann.Hashes {
		peer.markTransaction(hash)
	}
	return backend.Handle(peer, ann)
}

func handleTransactions(backend Backend, msg Decoder, peer *Peer) error {
	// Transactions can be processed, parse all of them and deliver to the pool
	var txs TransactionsPacket
	if err := msg.Decode(&txs); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	for i, tx := range txs {
		// Validate and mark the remote transaction
		if tx == nil {
			return fmt.Errorf("%w: transaction %d is nil", errDecode, i)
		}
		peer.markTransaction(tx.Hash())
	}
	return backend.Handle(peer, &txs)
}

func handlePooledTransactions(backend Backend, msg Decoder, peer *Peer) error {

	// Transactions can be processed, parse all of them and deliver to the pool
	var txs PooledTransactionsPacket
	if err := msg.Decode(&txs); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	for i, tx := range txs.PooledTransactionsResponse {
		// Validate and mark the remote transaction
		if tx == nil {
			return fmt.Errorf("%w: transaction %d is nil", errDecode, i)
		}
		peer.markTransaction(tx.Hash())
	}
	requestTracker.Fulfil(peer.id, peer.version, PooledTransactionsMsg, txs.RequestId)
	return backend.Handle(peer, &txs.PooledTransactionsResponse)
}
