package extension

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// recoverSignerKey recovers the signer's ECDSA public key (and derived address) from an
// EIP-191 personal_sign signature over `message`: the signer signs the 32-byte
// keccak256(message) digest via personal_sign, i.e. sign(TextHash(keccak256(message))).
// Mirrors shielded-transfer's crypto.go so session keys work identically across extensions.
func recoverSignerKey(message, signature []byte) (*ecdsa.PublicKey, common.Address, error) {
	if len(signature) != 65 {
		return nil, common.Address{}, fmt.Errorf("signature length: expected 65, got %d", len(signature))
	}
	sig := bytes.Clone(signature)
	if sig[64] >= 27 {
		sig[64] -= 27
	}
	msgHash := crypto.Keccak256(message)
	pub, err := crypto.SigToPub(accounts.TextHash(msgHash), sig)
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("SigToPub: %w", err)
	}
	return pub, crypto.PubkeyToAddress(*pub), nil
}

// authorizeDirectSigner authorizes a signed Direct-path op that claims to act as `identity`.
// It accepts the recovered signer if EITHER:
//
//	(a) the signer IS the identity — an EOA signing as itself (the MetaMask model), or
//	(b) the signer is the session key currently bound to the identity — the FSA model,
//	    where the identity is a PersonalAccount with no private key of its own, so its
//	    XRPL owner authorizes via a session key bound through BIND_SESSION_SIG.
//
// `boundPub` is the identity's bound key (nil if none). A PersonalAccount can never
// satisfy (a), so for FSA identities authorization reduces to "signed by the bound
// session key" — and that key can only be set by a valid XRPL signature from the
// account's owner.
func authorizeDirectSigner(signerAddr, identity common.Address, boundPub *ecdsa.PublicKey) error {
	if signerAddr == identity {
		return nil
	}
	if boundPub != nil && crypto.PubkeyToAddress(*boundPub) == signerAddr {
		return nil
	}
	return fmt.Errorf("signer mismatch: signature recovered to %s, which is neither %s nor its bound session key", signerAddr.Hex(), identity.Hex())
}
