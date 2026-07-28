package extension

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/flare-foundation/go-flare-common/pkg/xrpl/address"
	"github.com/flare-foundation/go-flare-common/pkg/xrpl/encoding"
	"github.com/flare-foundation/go-flare-common/pkg/xrpl/encoding/types"
	"github.com/flare-foundation/go-flare-common/pkg/xrpl/signing/ed25519"
	"github.com/flare-foundation/go-flare-common/pkg/xrpl/signing/secp256k1"
	"github.com/flare-foundation/go-flare-common/pkg/xrpl/signing/utils"
)

// verifyXrplSingleSig verifies a single-signed XRPL transaction blob (e.g. a Xaman SignIn) and
// returns the signer's classic r-address plus the decoded transaction.
//
// It mirrors signing.ValidateMultiSig but for the single-sign path: decode the blob, re-encode it
// "for signing" (which drops the TxnSignature), prepare the single-sign message
// (utils.Prepare(..., multiSig=false, nil) — the STX prefix, no signer suffix), then validate the
// signature against the embedded SigningPubKey and derive the r-address from that pubkey.
//
// This is the trust root of the mint-free FSA bind: a valid signature proves the holder of that
// XRPL account authorized this exact (domain-memo'd) message, in-enclave, with no FXRP mint, no
// gas, and no FDC round-trip. Copied verbatim from shielded-transfer.
func verifyXrplSingleSig(blob []byte) (rAddress string, tx map[string]any, err error) {
	tx, err = types.Decode(blob)
	if err != nil {
		return "", nil, fmt.Errorf("decoding xrpl blob: %w", err)
	}

	pub, _ := tx["SigningPubKey"].(string)
	sigHex, _ := tx["TxnSignature"].(string)
	if pub == "" || sigHex == "" {
		return "", nil, fmt.Errorf("blob missing SigningPubKey or TxnSignature")
	}

	forSigning, err := encoding.Encode(tx, true) // signing=true drops TxnSignature
	if err != nil {
		return "", nil, fmt.Errorf("re-encoding for signing: %w", err)
	}
	msg, err := utils.Prepare(forSigning, false, nil) // single-sign (not multi-sig)
	if err != nil {
		return "", nil, fmt.Errorf("preparing signing message: %w", err)
	}
	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return "", nil, fmt.Errorf("decoding signature: %w", err)
	}

	var ok bool
	switch strings.ToUpper(pub[:2]) {
	case "ED":
		ok, err = ed25519.Validate(msg, sig, pub)
	default:
		ok, err = secp256k1.Validate(msg, sig, pub)
	}
	if err != nil {
		return "", nil, fmt.Errorf("validating signature: %w", err)
	}
	if !ok {
		return "", nil, fmt.Errorf("invalid XRPL signature")
	}

	rAddress, err = address.PubToAddress(pub)
	if err != nil {
		return "", nil, fmt.Errorf("deriving r-address from pubkey: %w", err)
	}
	return rAddress, tx, nil
}
