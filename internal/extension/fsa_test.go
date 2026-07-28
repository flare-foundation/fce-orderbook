package extension

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"extension-scaffold/pkg/balance"
	"extension-scaffold/pkg/types"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/flare-foundation/go-flare-common/pkg/tee/instruction"
	xrplenc "github.com/flare-foundation/go-flare-common/pkg/xrpl/encoding"
	"github.com/flare-foundation/go-flare-common/pkg/xrpl/signing/secp256k1"
	"github.com/flare-foundation/go-flare-common/pkg/xrpl/signing/utils"
	teetypes "github.com/flare-foundation/tee-node/pkg/types"
)

// ---- helpers ----------------------------------------------------------------

// Real signed SignIn blob captured from a live Xaman approval (account r9TzTm…),
// including a domain memo. If verifyXrplSingleSig validates it and derives the
// right r-address, the mint-free bind's trust root works against real wallet
// output. (Fixture shared with shielded-transfer.)
const realXamanSignInBlob = "73210292F900AB3BFCF842D49F61AB30C13B66662E8A921C24076418692722305BE73C74473045022100F1729A1BF975390916CAEF2642828C7105D8917E591FF013651C485ECCC1BCF00220139F1C85B2970F7AAF7A43A781E2243E385961D562148B1075AB172C1592B5F781145CDBC9A961A9188F7FFE8EB39574B65B5C1C766EF9EA7D6E466C61726520536869656C646564205472616E736665727320E28094206465726976652073657373696F6E206B657920E28094207661756C742030783538626241373235634643413438354246434239424338413631353361423135453133376261624520636861696E20313134E1F1"

const realXamanAddr = "r9TzTmZjQvbYTnLb89jcfFEFRJKovDgwQb"

// startMockSignServer serves /sign with the given TEE key, mirroring the real
// sign server (keccak256 + EIP-191 prefix over the raw message).
func startMockSignServer(t *testing.T, key *ecdsa.PrivateKey) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/sign", func(w http.ResponseWriter, r *http.Request) {
		var req teetypes.SignRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		msgHash := crypto.Keccak256(req.Message)
		sig, err := crypto.Sign(accounts.TextHash(msgHash), key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(teetypes.SignResponse{Message: req.Message, Signature: sig})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func portOf(t *testing.T, rawURL string) int {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	p, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatalf("port: %v", err)
	}
	return p
}

// newFsaTestExtension builds a minimal Extension for FSA direct-op tests.
func newFsaTestExtension(t *testing.T, contract common.Address, teeKey *ecdsa.PrivateKey) *Extension {
	t.Helper()
	e := &Extension{
		balances:          balance.NewManager(),
		history:           newHistory(),
		fsa:               newFsaStore(),
		instructionSender: contract,
	}
	if teeKey != nil {
		e.signPort = portOf(t, startMockSignServer(t, teeKey).URL)
	}
	return e
}

// signCanonical signs EIP-191 personal_sign over keccak256(canonical) — the
// session-key signing scheme recoverSignerKey expects.
func signCanonical(t *testing.T, key *ecdsa.PrivateKey, canonical []byte) []byte {
	t.Helper()
	sig, err := crypto.Sign(accounts.TextHash(crypto.Keccak256(canonical)), key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return sig
}

type mockPaResolver struct {
	pa    map[string]common.Address
	calls int
}

func (m *mockPaResolver) PersonalAccountFor(_ context.Context, r string) (common.Address, error) {
	m.calls++
	return m.pa[r], nil
}

func bindStatementFor(contract common.Address, sessionPub []byte, nonce uint64) string {
	return fmt.Sprintf("%s | bind session key 0x%x | contract %s | nonce %d",
		bindStatementDomain, sessionPub, contract.Hex(), nonce)
}

// signedSignInBlob builds a Xaman-style SignIn blob (no TransactionType, memo
// carries `statement`) signed with the given XRPL secp256k1 key — the same
// canonical encode → prepare → sign path a real wallet uses.
func signedSignInBlob(t *testing.T, xrplKey *ecdsa.PrivateKey, statement string) []byte {
	t.Helper()
	tx := map[string]any{
		"Account":       secp256k1.PrvToAddress(xrplKey),
		"SigningPubKey": secp256k1.PrvToPub(xrplKey),
		"Memos": []any{
			map[string]any{"Memo": map[string]any{
				"MemoData": strings.ToUpper(hex.EncodeToString([]byte(statement))),
			}},
		},
	}
	forSigning, err := xrplenc.Encode(tx, true)
	if err != nil {
		t.Fatalf("encode for signing: %v", err)
	}
	msg, err := utils.Prepare(forSigning, false, nil)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	sig, err := secp256k1.SignXRPL(msg, xrplKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	tx["TxnSignature"] = strings.ToUpper(hex.EncodeToString(sig))
	blob, err := xrplenc.Encode(tx, false)
	if err != nil {
		t.Fatalf("encode full: %v", err)
	}
	return blob
}

// ---- verifyXrplSingleSig ----------------------------------------------------

func TestVerifyXrplSingleSig_RealXamanBlob(t *testing.T) {
	blob, err := hex.DecodeString(realXamanSignInBlob)
	if err != nil {
		t.Fatalf("decode blob hex: %v", err)
	}

	addr, tx, err := verifyXrplSingleSig(blob)
	if err != nil {
		t.Fatalf("verifyXrplSingleSig: %v", err)
	}
	if addr != realXamanAddr {
		t.Errorf("r-address: got %s, want %s", addr, realXamanAddr)
	}
	if _, ok := tx["Memos"]; !ok {
		t.Errorf("expected Memos (domain separation) in decoded tx")
	}
}

func TestVerifyXrplSingleSig_TamperedRejected(t *testing.T) {
	blob, _ := hex.DecodeString(realXamanSignInBlob)
	blob[len(blob)-1] ^= 0xff // flip a byte → signature must no longer verify
	if _, _, err := verifyXrplSingleSig(blob); err == nil {
		t.Fatal("expected verification to fail on a tampered blob")
	}
}

// ---- BIND_SESSION_SIG ---------------------------------------------------------

func TestProcessBindSessionSig_HappyPathAndReplay(t *testing.T) {
	sessionKey, _ := crypto.GenerateKey()
	xrplKey, _ := crypto.GenerateKey()
	contract := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	pa := common.HexToAddress("0x1212121212121212121212121212121212121212")
	rAddr := secp256k1.PrvToAddress(xrplKey)

	resolver := &mockPaResolver{pa: map[string]common.Address{rAddr: pa}}
	e := newFsaTestExtension(t, contract, nil)
	e.paResolver = resolver

	sessionPub := crypto.FromECDSAPub(&sessionKey.PublicKey)
	blob := signedSignInBlob(t, xrplKey, bindStatementFor(contract, sessionPub, 7))

	msg, _ := json.Marshal(types.BindSessionSigRequest{Contract: contract, XrplBlob: blob})
	ar := e.processBindSessionSig(teetypes.Action{}, &instruction.DataFixed{}, msg)
	if ar.Status != 1 {
		t.Fatalf("expected Status=1, got %d (log=%s)", ar.Status, ar.Log)
	}
	if resolver.calls != 1 {
		t.Errorf("expected 1 resolver call, got %d", resolver.calls)
	}

	var resp types.BindSessionSigResponse
	if err := json.Unmarshal(ar.Data, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.User != pa || resp.XrplAddress != rAddr {
		t.Errorf("response wrong: %+v", resp)
	}

	bound, ok := e.fsa.GetBinding(strings.ToLower(pa.Hex()))
	if !ok || !bytes.Equal(bound, sessionPub) {
		t.Fatalf("binding not written correctly (ok=%v)", ok)
	}

	// Replaying the exact same signed statement must fail the nonce check.
	ar2 := e.processBindSessionSig(teetypes.Action{}, &instruction.DataFixed{}, msg)
	if ar2.Status != 0 || !strings.Contains(ar2.Log, "nonce") {
		t.Fatalf("expected replay rejection, got status=%d log=%s", ar2.Status, ar2.Log)
	}
}

func TestProcessBindSessionSig_TamperedBlobRejected(t *testing.T) {
	sessionKey, _ := crypto.GenerateKey()
	xrplKey, _ := crypto.GenerateKey()
	contract := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	e := newFsaTestExtension(t, contract, nil)
	e.paResolver = &mockPaResolver{pa: map[string]common.Address{}}

	sessionPub := crypto.FromECDSAPub(&sessionKey.PublicKey)
	blob := signedSignInBlob(t, xrplKey, bindStatementFor(contract, sessionPub, 1))
	blob[len(blob)-1] ^= 0xff

	msg, _ := json.Marshal(types.BindSessionSigRequest{Contract: contract, XrplBlob: blob})
	if ar := e.processBindSessionSig(teetypes.Action{}, &instruction.DataFixed{}, msg); ar.Status != 0 {
		t.Fatalf("expected rejection of tampered blob, got status=%d", ar.Status)
	}
}

func TestProcessBindSessionSig_WrongContractInStatementRejected(t *testing.T) {
	sessionKey, _ := crypto.GenerateKey()
	xrplKey, _ := crypto.GenerateKey()
	contract := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	other := common.HexToAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	e := newFsaTestExtension(t, contract, nil)
	e.paResolver = &mockPaResolver{pa: map[string]common.Address{}}

	sessionPub := crypto.FromECDSAPub(&sessionKey.PublicKey)
	blob := signedSignInBlob(t, xrplKey, bindStatementFor(other, sessionPub, 1))

	msg, _ := json.Marshal(types.BindSessionSigRequest{Contract: contract, XrplBlob: blob})
	ar := e.processBindSessionSig(teetypes.Action{}, &instruction.DataFixed{}, msg)
	if ar.Status != 0 || !strings.Contains(ar.Log, "contract") {
		t.Fatalf("expected wrong-contract rejection, got status=%d log=%s", ar.Status, ar.Log)
	}
}

func TestProcessBindSessionSig_WrongDomainRejected(t *testing.T) {
	sessionKey, _ := crypto.GenerateKey()
	xrplKey, _ := crypto.GenerateKey()
	contract := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	e := newFsaTestExtension(t, contract, nil)
	e.paResolver = &mockPaResolver{pa: map[string]common.Address{}}

	sessionPub := crypto.FromECDSAPub(&sessionKey.PublicKey)
	stmt := fmt.Sprintf("Some Other Dapp v9 | bind session key 0x%x | contract %s | nonce 1", sessionPub, contract.Hex())
	blob := signedSignInBlob(t, xrplKey, stmt)

	msg, _ := json.Marshal(types.BindSessionSigRequest{Contract: contract, XrplBlob: blob})
	ar := e.processBindSessionSig(teetypes.Action{}, &instruction.DataFixed{}, msg)
	if ar.Status != 0 || !strings.Contains(ar.Log, "domain") {
		t.Fatalf("expected wrong-domain rejection, got status=%d log=%s", ar.Status, ar.Log)
	}
}

// ---- GET_BINDING --------------------------------------------------------------

func TestProcessGetBinding(t *testing.T) {
	contract := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	pa := common.HexToAddress("0x1212121212121212121212121212121212121212")
	e := newFsaTestExtension(t, contract, nil)

	msg, _ := json.Marshal(types.GetBindingRequest{Target: pa})
	ar := e.processGetBinding(teetypes.Action{}, &instruction.DataFixed{}, msg)
	var resp types.GetBindingResponse
	_ = json.Unmarshal(ar.Data, &resp)
	if ar.Status != 1 || resp.Bound {
		t.Fatalf("expected unbound, got status=%d bound=%v", ar.Status, resp.Bound)
	}

	sessionKey, _ := crypto.GenerateKey()
	sessionPub := crypto.FromECDSAPub(&sessionKey.PublicKey)
	e.fsa.SetBinding(strings.ToLower(pa.Hex()), sessionPub)

	ar = e.processGetBinding(teetypes.Action{}, &instruction.DataFixed{}, msg)
	_ = json.Unmarshal(ar.Data, &resp)
	if !resp.Bound || !bytes.Equal(resp.SessionPub, sessionPub) {
		t.Fatalf("expected bound with session pub, got %+v", resp)
	}
}

// ---- WITHDRAW_REQUEST ---------------------------------------------------------

func withdrawReqMsg(t *testing.T, signer *ecdsa.PrivateKey, contract, user, token, to common.Address, amount, nonce uint64) []byte {
	t.Helper()
	canon, err := types.CanonicalWithdrawRequestBytes(contract, user, token, to, amount, nonce)
	if err != nil {
		t.Fatalf("canonical: %v", err)
	}
	req := types.WithdrawRequestPayload{
		Contract: contract, User: user, Token: token, To: to,
		Amount: amount, Nonce: nonce,
		Signature: hexutil.Bytes(signCanonical(t, signer, canon)),
	}
	msg, _ := json.Marshal(req)
	return msg
}

// FSA model: `user` is a PersonalAccount; the bound session key signs.
func TestProcessWithdrawRequest_SessionKeyHappyPathAndReplay(t *testing.T) {
	teeKey, _ := crypto.GenerateKey()
	sessionKey, _ := crypto.GenerateKey()
	user := common.HexToAddress("0x1212121212121212121212121212121212121212")
	contract := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	token := common.HexToAddress("0xdddddddddddddddddddddddddddddddddddddddd")
	dest := common.HexToAddress("0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")

	e := newFsaTestExtension(t, contract, teeKey)
	e.fsa.SetBinding(strings.ToLower(user.Hex()), crypto.FromECDSAPub(&sessionKey.PublicKey))
	if err := e.balances.Deposit(strings.ToLower(user.Hex()), token, 100); err != nil {
		t.Fatal(err)
	}

	withdrawalID := common.HexToHash("0x0101010101010101010101010101010101010101010101010101010101010101")
	msg := withdrawReqMsg(t, sessionKey, contract, user, token, dest, 75, 1)
	ar := e.processWithdrawRequest(teetypes.Action{}, &instruction.DataFixed{InstructionID: withdrawalID}, msg)
	if ar.Status != 1 {
		t.Fatalf("expected Status=1, got %d (log=%s)", ar.Status, ar.Log)
	}

	var resp types.WithdrawResponse
	if err := json.Unmarshal(ar.Data, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Token != token || resp.Amount != 75 || resp.To != dest || resp.Available != 25 {
		t.Errorf("response wrong: %+v", resp)
	}
	if resp.WithdrawalID != withdrawalID || len(resp.Signature) != 65 {
		t.Errorf("slip malformed: id=%s sigLen=%d", resp.WithdrawalID.Hex(), len(resp.Signature))
	}
	// The slip must recover to the TEE key — that's what executeWithdrawal checks.
	payload := packWithdrawalMessage(token, 75, dest, resp.WithdrawalID)
	_, signer, err := recoverSignerKey(payload, resp.Signature)
	if err != nil || signer != crypto.PubkeyToAddress(teeKey.PublicKey) {
		t.Errorf("slip signature does not recover to TEE authority (err=%v, signer=%s)", err, signer.Hex())
	}

	// Replay: same nonce again must be rejected, balance untouched.
	ar2 := e.processWithdrawRequest(teetypes.Action{}, &instruction.DataFixed{InstructionID: withdrawalID}, withdrawReqMsg(t, sessionKey, contract, user, token, dest, 10, 1))
	if ar2.Status != 0 || !strings.Contains(ar2.Log, "nonce") {
		t.Fatalf("expected replay rejection, got status=%d log=%s", ar2.Status, ar2.Log)
	}
	if bal := e.balances.Get(strings.ToLower(user.Hex()), token); bal.Available != 25 {
		t.Errorf("balance changed on rejected replay: %d", bal.Available)
	}
}

// Legacy model: an EOA signs as itself; no binding required.
func TestProcessWithdrawRequest_SelfSignedUnbound(t *testing.T) {
	teeKey, _ := crypto.GenerateKey()
	userKey, _ := crypto.GenerateKey()
	user := crypto.PubkeyToAddress(userKey.PublicKey)
	contract := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	token := common.HexToAddress("0xdddddddddddddddddddddddddddddddddddddddd")

	e := newFsaTestExtension(t, contract, teeKey)
	if err := e.balances.Deposit(strings.ToLower(user.Hex()), token, 50); err != nil {
		t.Fatal(err)
	}

	msg := withdrawReqMsg(t, userKey, contract, user, token, user, 50, 1)
	ar := e.processWithdrawRequest(teetypes.Action{}, &instruction.DataFixed{InstructionID: common.HexToHash("0x02")}, msg)
	if ar.Status != 1 {
		t.Fatalf("expected Status=1, got %d (log=%s)", ar.Status, ar.Log)
	}
}

func TestProcessWithdrawRequest_WrongSignerRejected(t *testing.T) {
	teeKey, _ := crypto.GenerateKey()
	sessionKey, _ := crypto.GenerateKey()
	mallory, _ := crypto.GenerateKey()
	user := common.HexToAddress("0x1212121212121212121212121212121212121212")
	contract := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	token := common.HexToAddress("0xdddddddddddddddddddddddddddddddddddddddd")

	e := newFsaTestExtension(t, contract, teeKey)
	e.fsa.SetBinding(strings.ToLower(user.Hex()), crypto.FromECDSAPub(&sessionKey.PublicKey))
	if err := e.balances.Deposit(strings.ToLower(user.Hex()), token, 100); err != nil {
		t.Fatal(err)
	}

	msg := withdrawReqMsg(t, mallory, contract, user, token, user, 10, 1)
	ar := e.processWithdrawRequest(teetypes.Action{}, &instruction.DataFixed{InstructionID: common.HexToHash("0x03")}, msg)
	if ar.Status != 0 || !strings.Contains(ar.Log, "signer mismatch") {
		t.Fatalf("expected signer rejection, got status=%d log=%s", ar.Status, ar.Log)
	}
	if bal := e.balances.Get(strings.ToLower(user.Hex()), token); bal.Available != 100 {
		t.Errorf("balance changed on rejected request: %d", bal.Available)
	}
}

func TestProcessWithdrawRequest_InsufficientBalance(t *testing.T) {
	teeKey, _ := crypto.GenerateKey()
	sessionKey, _ := crypto.GenerateKey()
	user := common.HexToAddress("0x1212121212121212121212121212121212121212")
	contract := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	token := common.HexToAddress("0xdddddddddddddddddddddddddddddddddddddddd")

	e := newFsaTestExtension(t, contract, teeKey)
	e.fsa.SetBinding(strings.ToLower(user.Hex()), crypto.FromECDSAPub(&sessionKey.PublicKey))
	if err := e.balances.Deposit(strings.ToLower(user.Hex()), token, 5); err != nil {
		t.Fatal(err)
	}

	msg := withdrawReqMsg(t, sessionKey, contract, user, token, user, 10, 1)
	ar := e.processWithdrawRequest(teetypes.Action{}, &instruction.DataFixed{InstructionID: common.HexToHash("0x04")}, msg)
	if ar.Status != 0 || !strings.Contains(ar.Log, "debiting") {
		t.Fatalf("expected insufficient-balance rejection, got status=%d log=%s", ar.Status, ar.Log)
	}
}
