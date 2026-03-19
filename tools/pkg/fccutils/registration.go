package fccutils

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"strings"
	"extension-scaffold/tools/pkg/support"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/flare-foundation/go-flare-common/pkg/contracts/teemachineregistry"
	"github.com/flare-foundation/go-flare-common/pkg/encoding"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/tee-node/pkg/fdc"
	"github.com/flare-foundation/tee-node/pkg/types"
	"github.com/pkg/errors"
	"time"
)

func RegisterNode(s *support.Support, teeInfo *types.SignedTeeInfoResponse, hostURL, ftdcTeeURL string, ftdcTee common.Address, command, instructionIDstring string) error {
	teeID, proxyID, err := TeeProxyId(teeInfo)
	if err != nil {
		return err
	}

	var teeAttestInstructionID common.Hash
	if strings.Contains(command, "r") {
		_, teeAttestInstructionID, err = PreRegistration(s, hostURL, teeID, proxyID, teeInfo)
		if err != nil {
			return err
		}

		time.Sleep(1 * time.Second)
	}
	if strings.Contains(command, "R") {
		teeAttestInstructionID, err = RequestTeeAttestation(s, teeID)
		if err != nil {
			return err
		}

		time.Sleep(1 * time.Second)
	}

	var instructionID common.Hash
	if strings.Contains(command, "a") {
		instructionID, err = RequestFTDCAvailabilityCheck(s, teeID, ftdcTee, teeAttestInstructionID)
		if err != nil {
			return err
		}

		time.Sleep(1 * time.Second)
	} else {
		instructionID = common.HexToHash(instructionIDstring)
	}

	if strings.Contains(command, "p") {
		toProductionProof, err := GetFTDCAvailabilityCheckResult(ftdcTeeURL, instructionID)
		if err != nil {
			return err
		}

		err = ToProduction(s, toProductionProof)
		if err != nil {
			return err
		}
	}

	return nil
}

func PreRegistration(
	s *support.Support,
	hostURL string,
	teeID common.Address,
	proxyID common.Address,
	teeInfo *types.SignedTeeInfoResponse,
) ([32]byte, common.Hash, error) {
	opts, err := bind.NewKeyedTransactorWithChainID(s.Prv, s.ChainID)
	if err != nil {
		return [32]byte{}, common.Hash{}, errors.Errorf("%s", err)
	}
	opts.Value = big.NewInt(int64(1000000000))

	teeMachineDataRegistry := teemachineregistry.ITeeMachineRegistryTeeMachineData{
		ExtensionId:  new(big.Int).SetBytes(teeInfo.MachineData.ExtensionID.Bytes()),
		InitialOwner: teeInfo.MachineData.InitialOwner,
		CodeHash:     teeInfo.MachineData.CodeHash,
		Platform:     teeInfo.MachineData.Platform,
		PublicKey:    teemachineregistry.PublicKey{X: teeInfo.MachineData.PublicKey.X, Y: teeInfo.MachineData.PublicKey.Y},
	}

	if len(teeInfo.DataSignature) != 65 {
		return [32]byte{}, common.Hash{}, errors.New("signature error")
	}
	sigVRS := encoding.TransformSignatureRSVtoVRS(teeInfo.DataSignature)

	signature := teemachineregistry.Signature{
		V: sigVRS[0],
		R: [32]byte(sigVRS[1:33]),
		S: [32]byte(sigVRS[33:65]),
	}

	claimBackAddress := crypto.PubkeyToAddress(s.Prv.PublicKey)
	tx, err := s.TeeMachineRegistry.Register(opts, teeMachineDataRegistry, signature, proxyID, hostURL, claimBackAddress)
	if err != nil {
		return [32]byte{}, common.Hash{}, errors.Errorf("error: %s", err)
	}

	receipt, err := support.CheckTx(tx, s.ChainClient)
	if err != nil {
		return [32]byte{}, common.Hash{}, err
	}
	logger.Infof("(pre)registration of TEE with ID %s succeeded", hex.EncodeToString(teeID[:]))

	if len(receipt.Logs) < 2 {
		return common.Hash{}, common.Hash{}, errors.New("unexpected logs, this should not happen")
	}
	logWithChallenge := receipt.Logs[1]
	var challenge [32]byte
	copy(challenge[:], logWithChallenge.Data)

	instructionID := receipt.Logs[0].Topics[2]
	logger.Infof("tee-attestation requested, instructionId: %s", hex.EncodeToString(instructionID[:]))

	return challenge, instructionID, nil
}

func RequestTeeAttestation(s *support.Support, teeID common.Address) (common.Hash, error) {
	opts, err := bind.NewKeyedTransactorWithChainID(s.Prv, s.ChainID)
	if err != nil {
		return [32]byte{}, errors.Errorf("%s", err)
	}
	opts.Value = big.NewInt(int64(1000000000))

	claimBackAddress := crypto.PubkeyToAddress(s.Prv.PublicKey)
	tx, err := s.TeeVerification.RequestTeeAttestation(opts, teeID, claimBackAddress)
	if err != nil {
		return [32]byte{}, errors.Errorf("error: %s", err)
	}

	receipt, err := support.CheckTx(tx, s.ChainClient)
	if err != nil {
		return [32]byte{}, err
	}

	if len(receipt.Logs) < 2 {
		return common.Hash{}, errors.New("unexpected logs, this should not happen")
	}
	logWithChallenge := receipt.Logs[1]
	var challenge [32]byte
	copy(challenge[:], logWithChallenge.Data)

	instructionID := receipt.Logs[0].Topics[2]
	logger.Infof("tee attestation requested, instructionId: %s", hex.EncodeToString(instructionID[:]))

	return instructionID, nil
}

func RequestFTDCAvailabilityCheck(s *support.Support, teeID, externalTeeID common.Address, teeAttestInstructionID [32]byte) (common.Hash, error) {
	opts, err := bind.NewKeyedTransactorWithChainID(s.Prv, s.ChainID)
	if err != nil {
		return common.Hash{}, errors.Errorf("%s", err)
	}
	opts.Value = big.NewInt(int64(1000000000))

	claimBackAddress := crypto.PubkeyToAddress(s.Prv.PublicKey)
	proofOwner := claimBackAddress
	tx, err := s.TeeVerification.RequestAvailabilityCheckAttestation(opts, teeID, teeAttestInstructionID, externalTeeID, proofOwner, claimBackAddress)
	if err != nil {
		return common.Hash{}, errors.Errorf("%s", err)
	}
	receipt, err := support.CheckTx(tx, s.ChainClient)
	if err != nil {
		return common.Hash{}, errors.Errorf("%s", err)
	}
	if len(receipt.Logs) < 2 || len(receipt.Logs[1].Topics) < 3 {
		return common.Hash{}, errors.Errorf("unexpected logs, this should not happen, number of logs %d, number of topics %d", len(receipt.Logs), len(receipt.Logs[1].Topics))
	}
	instructionID := receipt.Logs[1].Topics[1]

	logger.Infof("availability check sent, instructionId: %s", hex.EncodeToString(instructionID[:]))

	return instructionID, nil
}

func GetFTDCAvailabilityCheckResult(hostURL string, instructionId common.Hash) (*teemachineregistry.ITeeAvailabilityCheckProof, error) {
	actionResult, err := ActionResult(hostURL, instructionId)
	if err != nil {
		return nil, err
	}
	var ftdcProof fdc.ProveResponse
	err = json.Unmarshal(actionResult.Result.Data, &ftdcProof)
	if err != nil {
		return nil, errors.Errorf("%s", err)
	}

	header, err := fdc.DecodeResponse(ftdcProof.ResponseHeader)
	if err != nil {
		return nil, errors.Errorf("%s", err)
	}

	request, err := DecodeFTDCTeeAvailabilityCheckRequest(ftdcProof.RequestBody)
	if err != nil {
		return nil, errors.Errorf("%s", err)
	}
	response, err := DecodeFTDCTeeAvailabilityCheckResponse(ftdcProof.ResponseBody)
	if err != nil {
		return nil, errors.Errorf("%s", err)
	}

	toProductionProof := teemachineregistry.ITeeAvailabilityCheckProof{
		Signatures:  teemachineregistry.IFdc2VerificationFdc2Signatures{SigningPolicySignatures: ftdcProof.DataProviderSignatures},
		Header:      teemachineregistry.IFdc2HubFdc2ResponseHeader(header),
		RequestBody: teemachineregistry.ITeeAvailabilityCheckRequestBody(request),
		ResponseBody: teemachineregistry.ITeeAvailabilityCheckResponseBody{
			Status:                 response.Status,
			TeeTimestamp:           response.TeeTimestamp,
			CodeHash:               response.CodeHash,
			Platform:               response.Platform,
			InitialSigningPolicyId: response.InitialSigningPolicyId,
			LastSigningPolicyId:    response.LastSigningPolicyId,
			State:                  teemachineregistry.ITeeAvailabilityCheckTeeState(response.State),
		},
	}

	logger.Infof("availability check proof obtained")

	return &toProductionProof, nil
}

func ToProduction(s *support.Support, toProductionProof *teemachineregistry.ITeeAvailabilityCheckProof) error {
	opts, err := bind.NewKeyedTransactorWithChainID(s.Prv, s.ChainID)
	if err != nil {
		return errors.Errorf("%s", err)
	}

	tx, err := s.TeeMachineRegistry.ToProduction(opts, *toProductionProof)
	if err != nil {
		return errors.Errorf("%s", err)
	}
	_, err = support.CheckTx(tx, s.ChainClient)
	if err != nil {
		return errors.Errorf("%s", err)
	}

	teeMachineInfo, err := s.TeeMachineRegistry.GetTeeMachine(nil, toProductionProof.RequestBody.TeeId)
	if err != nil {
		return errors.Errorf("%s", err)
	}
	if teeMachineInfo.TeeId != toProductionProof.RequestBody.TeeId {
		return errors.New("tee machine not set up correctly")
	}

	return nil
}

func AddTeeVersion(s *support.Support, privKey *ecdsa.PrivateKey, extensionId *big.Int, codeHash common.Hash, platform common.Hash, governanceHash common.Hash, version string) error {
	opts, err := bind.NewKeyedTransactorWithChainID(privKey, s.ChainID)
	if err != nil {
		return errors.Errorf("%s", err)
	}

	tx, err := s.TeeExtensionRegistry.AddTeeVersion(opts, extensionId, version, codeHash, [][32]byte{platform}, governanceHash)
	if err != nil {
		return errors.Errorf("TeeExtensionRegistry.AddTeeVersion failed: %s", err)
	}

	_, err = support.CheckTx(tx, s.ChainClient)
	if err != nil {
		return errors.Errorf("%s", err)
	}

	return nil
}
