package main

import (
	"encoding/json"
	"flag"
	"time"

	"extension-e2e/configs"
	"extension-e2e/pkg/support"
	"extension-e2e/pkg/utils"
	instrutils "extension-scaffold/tools/pkg/utils"

	"github.com/ethereum/go-ethereum/common"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/pkg/errors"
)

// =============================================================================
// ★ EXTENSION-SPECIFIC: Define your response types here.
//
// The scaffold's example uses a simple message-in/message-out pattern.
// Replace this with your own response struct(s) matching what your
// extension's processAction handler returns via buildResult.
// =============================================================================

type MyActionResponse struct {
	Message string `json:"message"`
}

func main() {
	af := flag.String("a", configs.AddressesFile, "file with deployed addresses")
	cf := flag.String("c", configs.ChainNodeURL, "chain node url")
	pf := flag.String("p", configs.ExtensionProxyURL, "extension proxy url")
	instructionSenderF := flag.String("instructionSender", "", "instructionSender address")
	flag.Parse()

	instructionSenderAddress := common.HexToAddress(*instructionSenderF)

	testSupport, err := support.DefaultSupport(*af, *cf)
	if err != nil {
		utils.FatalWithCause(err)
	}

	// --- Generic: configure contract (do not modify) --------------------------
	// SetExtensionId is idempotent — safe to call even if already set.
	logger.Infof("Setting extension ID on instruction sender...")
	err = instrutils.SetExtensionId(testSupport, instructionSenderAddress)
	if err != nil {
		logger.Warnf("setExtensionId failed (may already be set): %s", err)
	}

	// =========================================================================
	// ★ EXTENSION-SPECIFIC: Your test cases below.
	//
	// The scaffold shows ONE example: send a JSON message via SendInstruction(),
	// wait for TEE processing, then verify the result.
	//
	// For YOUR extension, replace with:
	//   1. YOUR message payload structs (instead of {"message": "hello"})
	//   2. YOUR contract function calls (instead of SendMyInstruction)
	//   3. Multiple test cases covering each of YOUR op types
	// =========================================================================

	// --- Test case 1: Send a MY_ACTION instruction ---
	logger.Infof("Sending MY_ACTION instruction (message='hello')...")
	payload, err := json.Marshal(map[string]string{"message": "hello"})
	if err != nil {
		utils.FatalWithCause(err)
	}

	instructionId, _, err := instrutils.SendInstruction(testSupport, instructionSenderAddress, payload)
	if err != nil {
		utils.FatalWithCause(err)
	}
	logger.Infof("Instruction sent. ID: %s", instructionId.Hex())

	time.Sleep(5 * time.Second)

	err = verifyResult(*pf, instructionId)
	if err != nil {
		utils.FatalWithCause(err)
	}
	logger.Infof("Test passed: MY_ACTION instruction processed successfully")

	logger.Infof("All tests passed.")
}

// =============================================================================
// ★ EXTENSION-SPECIFIC: Replace this with your own verification logic.
//
// The generic part is:
//   - utils.ActionResult() polls the proxy and returns *types.ActionResponse
//   - ActionResponse.Result.Status: 0=failed, 1=success, 2=pending
//   - ActionResponse.Result.Data contains YOUR extension's JSON response
//
// You must:
//   1. Define your response struct (see MyActionResponse above)
//   2. Unmarshal actionResult.Data into it
//   3. Validate the fields match your expectations
// =============================================================================

func verifyResult(proxyURL string, instructionId common.Hash) error {
	// --- Generic: poll proxy for result ---
	actionResponse, err := utils.ActionResult(proxyURL, instructionId)
	if err != nil {
		return err
	}
	actionResult := actionResponse.Result

	// --- Generic: check processing status ---
	if actionResult.Status == 0 {
		return errors.Errorf("instruction processing failed: %s", actionResult.Log)
	}
	if actionResult.Status == 2 {
		return errors.New("instruction still pending after polling, expected completed")
	}

	if len(actionResult.Data) == 0 {
		return errors.New("expected response data but got none")
	}

	// ★ CUSTOM: unmarshal into YOUR response type
	var resp MyActionResponse
	err = json.Unmarshal(actionResult.Data, &resp)
	if err != nil {
		return errors.Errorf("failed to unmarshal response: %s", err)
	}

	// ★ CUSTOM: validate YOUR specific fields
	logger.Infof("Response data: %+v", resp)

	return nil
}
