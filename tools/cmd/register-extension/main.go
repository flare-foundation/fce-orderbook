package main

import (
	"flag"
	"fmt"

	"extension-e2e/configs"
	"extension-e2e/pkg/support"
	extutils "extension-e2e/pkg/utils"

	"github.com/ethereum/go-ethereum/common"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
)

func main() {
	af := flag.String("a", configs.AddressesFile, "file with deployed addresses")
	cf := flag.String("c", configs.ChainNodeURL, "chain node url")
	instructionSenderF := flag.String("instructionSender", "", "InstructionSender contract address (required)")
	governanceHashF := flag.String("governanceHash", "", "governance hash (optional)")
	flag.Parse()

	if *instructionSenderF == "" {
		logger.Fatal("--instructionSender flag is required")
	}

	testSupport, err := support.DefaultSupport(*af, *cf)
	if err != nil {
		extutils.FatalWithCause(err)
	}

	governanceHash := common.HexToHash(*governanceHashF)
	instructionSenderAddress := common.HexToAddress(*instructionSenderF)

	logger.Infof("Registering extension with InstructionSender %s...", instructionSenderAddress.Hex())
	extensionID, err := extutils.SetupExtension(testSupport, governanceHash, instructionSenderAddress, common.Address{})
	if err != nil {
		extutils.FatalWithCause(err)
	}

	extensionIDHex := fmt.Sprintf("0x%064x", extensionID)
	logger.Infof("Extension registered with ID: %s", extensionIDHex)

	// Machine-readable output on stdout
	fmt.Println(extensionIDHex)
}
