package extension

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"strings"

	"extension-scaffold/pkg/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/flare-foundation/go-flare-common/pkg/tee/instruction"
	teetypes "github.com/flare-foundation/tee-node/pkg/types"
)

// processWithdrawRequest handles WITHDRAW_REQUEST direct instructions — the
// off-chain twin of the on-chain WITHDRAW.
//
// Withdrawing is two separable things: an authorization ("this user wants X
// out, paid to Y") and a value movement (the vault releasing tokens). Only the
// movement must live on-chain — executeWithdrawal already accepts the TEE's
// slip from any caller. This op moves the authorization off-chain: the user
// (or the session key bound to them — the FSA model, since a gasless
// PersonalAccount can't call withdraw() itself) signs the canonical request,
// the TEE debits and signs the slip in seconds, and a relayer carries the slip
// on-chain. No XRPL payment, no FDC round-trip.
//
// Flow:
//  1. Unmarshal WithdrawRequestPayload; structural validation; pin to this
//     deployment's InstructionSender.
//  2. Recover the inner signer; authorize against `user` (self or bound key).
//  3. Advance the per-user nonce (replay protection — without it a replayed
//     request would debit the balance again and mint a second slip).
//  4. issueWithdrawal — same debit + slip as the on-chain path, with the
//     Direct action id as the single-use withdrawalId.
//
// The response is public JSON, like the on-chain path: the slip must be
// carriable by third parties, and it authorizes nothing except paying `to`
// exactly what the signer requested.
func (e *Extension) processWithdrawRequest(action teetypes.Action, df *instruction.DataFixed, msg hexutil.Bytes) teetypes.ActionResult {
	var req types.WithdrawRequestPayload
	if err := json.Unmarshal(msg, &req); err != nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("decoding request: %w", err))
	}

	if req.Amount == 0 {
		return buildResult(action, df, nil, 0, fmt.Errorf("amount must be positive"))
	}
	if req.User == (common.Address{}) {
		return buildResult(action, df, nil, 0, fmt.Errorf("user address is zero"))
	}
	if req.Token == (common.Address{}) {
		return buildResult(action, df, nil, 0, fmt.Errorf("token address is zero"))
	}
	if req.To == (common.Address{}) {
		return buildResult(action, df, nil, 0, fmt.Errorf("destination address is zero"))
	}
	if len(req.Signature) != 65 {
		return buildResult(action, df, nil, 0, fmt.Errorf("signature length: expected 65, got %d", len(req.Signature)))
	}
	if err := requireBoundContract(req.Contract, e.instructionSender); err != nil {
		return buildResult(action, df, nil, 0, err)
	}

	// Recover + authorize: the user's own key, or the session key bound to them.
	canonical, err := types.CanonicalWithdrawRequestBytes(req.Contract, req.User, req.Token, req.To, req.Amount, req.Nonce)
	if err != nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("canonical encoding: %w", err))
	}
	_, signerAddr, err := recoverSignerKey(canonical, req.Signature)
	if err != nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("recovering signer: %w", err))
	}
	userKey := strings.ToLower(req.User.Hex())
	var boundPub *ecdsa.PublicKey
	if boundBytes, bound := e.fsa.GetBinding(userKey); bound {
		if boundPub, err = crypto.UnmarshalPubkey(boundBytes); err != nil {
			return buildResult(action, df, nil, 0, fmt.Errorf("malformed bound session pubkey: %w", err))
		}
	}
	if err := authorizeDirectSigner(signerAddr, req.User, boundPub); err != nil {
		return buildResult(action, df, nil, 0, err)
	}

	if !e.fsa.CheckAndAdvanceNonce(userKey, req.Nonce) {
		return buildResult(action, df, nil, 0, fmt.Errorf("stale or replayed nonce: %d", req.Nonce))
	}

	// The Direct action id doubles as the single-use withdrawalId, exactly like
	// the on-chain path uses its instruction id.
	resp, err := e.issueWithdrawal(userKey, req.Token, req.Amount, req.To, df.InstructionID)
	if err != nil {
		return buildResult(action, df, nil, 0, err)
	}

	data, _ := json.Marshal(resp)
	return buildResult(action, df, data, 1, nil)
}
