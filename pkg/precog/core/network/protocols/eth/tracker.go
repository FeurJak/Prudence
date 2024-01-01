package eth

import (
	"time"

	"github.com/ethereum/go-ethereum/p2p/tracker"
)

// requestTracker is a singleton tracker for eth/66 and newer request times.
var requestTracker = tracker.New(ProtocolName, 5*time.Minute)
