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

// --- CUSTOMIZE: Define your response types here. ---
// These must match the types your extension returns in ActionResult.Data.
// See pkg/types/types.go for the corresponding request/response definitions.

type MyActionResponse struct {
	// TODO: Add fields matching your types.MyActionResponse.
	// Example:
	//   TxHash string `json:"txHash"`
	//   Status string `json:"status"`
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

	// --- CUSTOMIZE: Your test cases below. ------------------------------------
	//
	// Each test case follows the same pattern:
	//   1. Build your JSON payload (matching your request type)
	//   2. Send it via instrutils.SendInstruction()
	//   3. Wait for TEE processing
	//   4. Verify the result
	//
	// The scaffold shows one placeholder test. Replace the payload and
	// verification with your own logic, and add more test cases as needed.

	// --- Test case 1: Send a MY_ACTION instruction ---
	logger.Infof("Sending MY_ACTION instruction...")

	// TODO: Build your actual test payload here.
	payload, err := json.Marshal(map[string]interface{}{
		// "from":   "0x...",
		// "to":     "0x...",
		// "amount": 100,
	})
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

	// TODO: Add more test cases for each operation type your extension supports.
	// --- Test case 2: ... ---

	logger.Infof("All tests passed.")
}

// --- CUSTOMIZE: Replace the unmarshalling and validation below. ---
//
// The generic part (polling the proxy, checking status) stays the same.
// You customize the part that unmarshals ActionResult.Data into your
// response type and validates the fields.

func verifyResult(proxyURL string, instructionId common.Hash) error {
	// --- Generic: poll proxy for result (do not modify) ---
	actionResponse, err := utils.ActionResult(proxyURL, instructionId)
	if err != nil {
		return err
	}
	actionResult := actionResponse.Result

	if actionResult.Status == 0 {
		return errors.Errorf("instruction processing failed: %s", actionResult.Log)
	}
	if actionResult.Status == 2 {
		return errors.New("instruction still pending after polling, expected completed")
	}

	if len(actionResult.Data) == 0 {
		return errors.New("expected response data but got none")
	}

	// --- CUSTOMIZE: unmarshal and validate your response ---
	var resp MyActionResponse
	err = json.Unmarshal(actionResult.Data, &resp)
	if err != nil {
		return errors.Errorf("failed to unmarshal response: %s", err)
	}

	// TODO: Add assertions for your response fields. For example:
	// if resp.TxHash == "" {
	//     return errors.New("expected non-empty TxHash")
	// }

	logger.Infof("Response data: %+v", resp)

	return nil
}
