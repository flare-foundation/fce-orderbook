package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"extension-scaffold/tools/pkg/configs"
	"extension-scaffold/tools/pkg/fccutils"
	"extension-scaffold/tools/pkg/support"
	instrutils "extension-scaffold/tools/pkg/utils"

	"github.com/flare-foundation/go-flare-common/pkg/logger"
)

func main() {
	af := flag.String("a", configs.AddressesFile, "file with deployed addresses")
	cf := flag.String("c", configs.ChainNodeURL, "chain node url")
	outFile := flag.String("o", "", "write deployed address to this file (optional)")
	flag.Parse()

	testSupport, err := support.DefaultSupport(*af, *cf)
	if err != nil {
		fccutils.FatalWithCause(err)
	}

	logger.Infof("Deploying InstructionSender contract...")
	address, _, err := instrutils.DeployInstructionSender(testSupport)
	if err != nil {
		fccutils.FatalWithCause(err)
	}

	logger.Infof("InstructionSender deployed at: %s", address.Hex())

	// Optionally write address to file for script consumption
	if *outFile != "" {
		os.MkdirAll(filepath.Dir(*outFile), 0755)
		os.WriteFile(*outFile, []byte(address.Hex()), 0644)
	}

	// Machine-readable output on stdout (for scripts)
	fmt.Println(address.Hex())
}
