package extension

import (
	"encoding/json"
	"fmt"

	"extension-scaffold/internal/config"
	"extension-scaffold/pkg/types"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/flare-foundation/go-flare-common/pkg/tee/instruction"
	teetypes "github.com/flare-foundation/tee-node/pkg/types"
	teeutils "github.com/flare-foundation/tee-node/pkg/utils"
)

// processGetBookState handles GET_BOOK_STATE direct instructions.
// Returns the public orderbook depth and recent matches — same payload as the
// internal GET /state HTTP endpoint, but reachable through the TEE proxy.
func (e *Extension) processGetBookState(action teetypes.Action, df *instruction.DataFixed, msg hexutil.Bytes) teetypes.ActionResult {
	if len(msg) > 0 {
		var req types.GetBookStateRequest
		if err := json.Unmarshal(msg, &req); err != nil {
			return buildResult(action, df, nil, 0, fmt.Errorf("decoding request: %w", err))
		}
	}

	e.mu.RLock()
	pairStates := make(map[string]types.PairState, len(e.orderbooks))
	for name, ob := range e.orderbooks {
		bids, asks := ob.Depth()
		pairStates[name] = types.PairState{Bids: bids, Asks: asks}
	}
	resp := types.StateResponse{
		StateVersion: teeutils.ToHash(config.Version),
		State: types.State{
			Pairs:      pairStates,
			MatchCount: len(e.matches),
			Matches:    e.matches,
		},
	}
	e.mu.RUnlock()

	data, _ := json.Marshal(resp)
	return buildResult(action, df, data, 1, nil)
}
