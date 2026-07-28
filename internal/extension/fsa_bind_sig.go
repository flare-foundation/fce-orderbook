package extension

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"extension-scaffold/pkg/types"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/flare-foundation/go-flare-common/pkg/tee/instruction"
	teetypes "github.com/flare-foundation/tee-node/pkg/types"
)

// The binding statement the user signs in Xaman (as the SignIn's memo). It is
// deliberately human-readable — Xaman shows memo text, so the user sees exactly
// what they authorize — while staying trivially machine-parseable:
//
//	Flare Orderbook v1 | bind session key <0x04…65-byte-hex> | contract <0x…> | nonce <uint64>
//
// The domain literal separates it from any other use of a SignIn signature; the
// contract (InstructionSender) pins it to one deployment; the nonce makes it
// single-use. Must match the frontend's bindSigStatement exactly.
const bindStatementDomain = "Flare Orderbook v1"

type bindStatement struct {
	SessionPub []byte
	Contract   common.Address
	Nonce      uint64
}

// PaResolver answers "which PersonalAccount does this XRPL address control?".
// The production implementation asks the MasterAccountController on-chain;
// tests use a mock. The mapping is deterministic (counterfactual CREATE2), so
// it works before the PersonalAccount is even deployed.
type PaResolver interface {
	PersonalAccountFor(ctx context.Context, xrplAddress string) (common.Address, error)
}

var macABI = mustParseMacABI(`[
	{"type":"function","name":"getPersonalAccount","stateMutability":"view",
	 "inputs":[{"name":"","type":"string"}],
	 "outputs":[{"name":"","type":"address"}]}
]`)

func mustParseMacABI(raw string) abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(raw))
	if err != nil {
		panic("invalid MAC ABI: " + err.Error())
	}
	return parsed
}

// MacClient resolves PersonalAccounts via the on-chain MasterAccountController.
// Dialed lazily per call; no cache — binds are rare and the extra RPC keeps the
// code one branch simpler.
type MacClient struct {
	controller common.Address
	chainURL   string
}

func NewMacClient(chainURL string, controller common.Address) *MacClient {
	return &MacClient{controller: controller, chainURL: chainURL}
}

func (m *MacClient) PersonalAccountFor(ctx context.Context, xrplAddress string) (common.Address, error) {
	if m == nil {
		return common.Address{}, fmt.Errorf("MAC resolver not configured")
	}
	cli, err := ethclient.DialContext(ctx, m.chainURL)
	if err != nil {
		return common.Address{}, fmt.Errorf("dialing chain: %w", err)
	}
	defer cli.Close()

	contract := bind.NewBoundContract(m.controller, macABI, cli, cli, cli)
	var out []any
	if err := contract.Call(&bind.CallOpts{Context: ctx}, &out, "getPersonalAccount", xrplAddress); err != nil {
		return common.Address{}, fmt.Errorf("getPersonalAccount call: %w", err)
	}
	if len(out) == 0 {
		return common.Address{}, fmt.Errorf("getPersonalAccount returned no value")
	}
	pa, ok := out[0].(common.Address)
	if !ok {
		return common.Address{}, fmt.Errorf("getPersonalAccount returned %T, expected address", out[0])
	}
	return pa, nil
}

// sessionPubFingerprint is a short human-comparable digest of a session pubkey.
func sessionPubFingerprint(pub []byte) string {
	sum := sha256.Sum256(pub)
	return hex.EncodeToString(sum[:8])
}

// requireBoundContract pins a signed request to this deployment's
// InstructionSender so a signature captured against one deployment can't be
// replayed against another.
func requireBoundContract(got, want common.Address) error {
	if want == (common.Address{}) {
		return fmt.Errorf("INSTRUCTION_SENDER not configured on this TEE")
	}
	if got != want {
		return fmt.Errorf("request is for contract %s, this TEE serves %s", got.Hex(), want.Hex())
	}
	return nil
}

// processBindSessionSig handles BIND_SESSION_SIG direct instructions — the
// mint-free session-key bind for Flare Smart Account (Xaman) users.
//
// A PersonalAccount is derived deterministically from an XRPL address, so a
// valid XRPL signature over our domain-separated binding statement, verified
// in-enclave, proves account ownership — instant and free (no FXRP mint, no
// gas, no FDC round-trip). Unlike shielded-transfer, the payload is plain JSON:
// this extension's direct path is unencrypted, and the statement contains no
// secrets — the XRPL signature covers it, so tampering is detectable.
//
// Flow:
//  1. Unmarshal BindSessionSigRequest; pin to this deployment's contract.
//  2. Verify the XRPL signature on the SignIn blob and recover the r-address.
//  3. Reject anything that could double as a ledger transaction.
//  4. Parse the memo's binding statement; require our domain and contract.
//  5. Resolve PersonalAccount(r-address) via the MasterAccountController.
//  6. Advance the per-user nonce (replay protection).
//  7. Bind.
func (e *Extension) processBindSessionSig(action teetypes.Action, df *instruction.DataFixed, msg hexutil.Bytes) teetypes.ActionResult {
	var req types.BindSessionSigRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("decoding request: %w", err))
	}

	if err := requireBoundContract(req.Contract, e.instructionSender); err != nil {
		return buildResult(action, df, nil, 0, err)
	}
	if len(req.XrplBlob) == 0 {
		return buildResult(action, df, nil, 0, fmt.Errorf("xrplBlob is empty"))
	}

	rAddress, tx, err := verifyXrplSingleSig(req.XrplBlob)
	if err != nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("verifying XRPL signature: %w", err))
	}
	// A Xaman SignIn omits TransactionType entirely; a blob that declares one is
	// (or aspires to be) a ledger transaction and must not double as a bind.
	if tt, ok := tx["TransactionType"].(string); ok && tt != "SignIn" {
		return buildResult(action, df, nil, 0, fmt.Errorf("blob is a %s transaction, expected a SignIn", tt))
	}

	stmt, err := parseBindStatement(tx)
	if err != nil {
		return buildResult(action, df, nil, 0, err)
	}
	if stmt.Contract != e.instructionSender {
		return buildResult(action, df, nil, 0, fmt.Errorf("statement binds contract %s, this TEE serves %s", stmt.Contract.Hex(), e.instructionSender.Hex()))
	}
	if len(stmt.SessionPub) != EncPubLen || stmt.SessionPub[0] != 0x04 {
		return buildResult(action, df, nil, 0, fmt.Errorf("session key must be 65 bytes uncompressed (0x04 prefix); got %d bytes", len(stmt.SessionPub)))
	}
	if _, err := crypto.UnmarshalPubkey(stmt.SessionPub); err != nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("invalid session key: %w", err))
	}

	if e.paResolver == nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("BIND_SESSION_SIG unavailable: MAC resolver not configured (set CHAIN_URL / MASTER_ACCOUNT_CONTROLLER)"))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pa, err := e.paResolver.PersonalAccountFor(ctx, rAddress)
	if err != nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("resolving PersonalAccount for %s: %w", rAddress, err))
	}
	if pa == (common.Address{}) {
		return buildResult(action, df, nil, 0, fmt.Errorf("no PersonalAccount for XRPL address %s", rAddress))
	}

	userKey := strings.ToLower(pa.Hex())
	if !e.fsa.CheckAndAdvanceNonce(userKey, stmt.Nonce) {
		return buildResult(action, df, nil, 0, fmt.Errorf("stale or replayed nonce: %d", stmt.Nonce))
	}

	e.fsa.SetBinding(userKey, stmt.SessionPub)

	resp := types.BindSessionSigResponse{
		User:        pa,
		XrplAddress: rAddress,
		SessionPub:  hexutil.Bytes(stmt.SessionPub),
		Fingerprint: sessionPubFingerprint(stmt.SessionPub),
	}
	data, _ := json.Marshal(resp)
	return buildResult(action, df, data, 1, nil)
}

// processGetBinding handles GET_BINDING direct instructions — an
// unauthenticated public lookup: "is target bound, and to which session
// pubkey?". The frontend uses it to skip the bind flow when the TEE already
// holds the same key.
func (e *Extension) processGetBinding(action teetypes.Action, df *instruction.DataFixed, msg hexutil.Bytes) teetypes.ActionResult {
	var req types.GetBindingRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		return buildResult(action, df, nil, 0, fmt.Errorf("decoding request: %w", err))
	}
	if req.Target == (common.Address{}) {
		return buildResult(action, df, nil, 0, fmt.Errorf("target address is zero"))
	}

	resp := types.GetBindingResponse{Target: req.Target}
	if pub, ok := e.fsa.GetBinding(strings.ToLower(req.Target.Hex())); ok {
		resp.Bound = true
		resp.SessionPub = hexutil.Bytes(pub)
		resp.Fingerprint = sessionPubFingerprint(pub)
	}
	data, _ := json.Marshal(resp)
	return buildResult(action, df, data, 1, nil)
}

// parseBindStatement extracts and parses the binding statement from the decoded
// SignIn's first memo.
func parseBindStatement(tx map[string]any) (bindStatement, error) {
	memoText, err := firstMemoText(tx)
	if err != nil {
		return bindStatement{}, err
	}

	parts := strings.Split(memoText, "|")
	if len(parts) != 4 {
		return bindStatement{}, fmt.Errorf("binding statement must have 4 '|'-separated parts, got %d", len(parts))
	}
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	if parts[0] != bindStatementDomain {
		return bindStatement{}, fmt.Errorf("wrong statement domain %q", parts[0])
	}

	keyHex, ok := strings.CutPrefix(parts[1], "bind session key ")
	if !ok {
		return bindStatement{}, fmt.Errorf("part 2 must be 'bind session key <hex>'")
	}
	sessionPub, err := hexutil.Decode(keyHex)
	if err != nil {
		return bindStatement{}, fmt.Errorf("decoding session key: %w", err)
	}

	contractHex, ok := strings.CutPrefix(parts[2], "contract ")
	if !ok || !common.IsHexAddress(contractHex) {
		return bindStatement{}, fmt.Errorf("part 3 must be 'contract <address>'")
	}

	nonceStr, ok := strings.CutPrefix(parts[3], "nonce ")
	if !ok {
		return bindStatement{}, fmt.Errorf("part 4 must be 'nonce <uint>'")
	}
	nonce, err := strconv.ParseUint(nonceStr, 10, 64)
	if err != nil {
		return bindStatement{}, fmt.Errorf("parsing nonce: %w", err)
	}

	return bindStatement{SessionPub: sessionPub, Contract: common.HexToAddress(contractHex), Nonce: nonce}, nil
}

// firstMemoText hex-decodes the MemoData of the transaction's first memo.
func firstMemoText(tx map[string]any) (string, error) {
	memos, ok := tx["Memos"].([]any)
	if !ok || len(memos) == 0 {
		return "", fmt.Errorf("SignIn carries no memo (binding statement required)")
	}
	wrapper, ok := memos[0].(map[string]any)
	if !ok {
		return "", fmt.Errorf("malformed memo entry")
	}
	memo, ok := wrapper["Memo"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("malformed memo entry (no Memo object)")
	}
	dataHex, ok := memo["MemoData"].(string)
	if !ok || dataHex == "" {
		return "", fmt.Errorf("memo has no MemoData")
	}
	data, err := hex.DecodeString(dataHex)
	if err != nil {
		return "", fmt.Errorf("decoding MemoData hex: %w", err)
	}
	return string(data), nil
}
