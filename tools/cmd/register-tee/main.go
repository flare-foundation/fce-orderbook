package main

import (
	"encoding/hex"
	"flag"
	"extension-e2e/configs"
	"extension-e2e/pkg/support"
	"extension-e2e/pkg/utils"

	"github.com/flare-foundation/go-flare-common/pkg/logger"
)

func main() {
	af := flag.String("a", configs.AddressesFile, "file with deployed addresses")
	cf := flag.String("c", configs.ChainNodeURL, "chain node url")
	pf := flag.String("p", configs.ExtensionProxyURL, "extension proxy url")
	epf := flag.String("ep", "http://localhost:6662", "external proxy url (for FTDC)")
	lf := flag.Bool("l", false, "local")
	instructionF := flag.String("i", "", "instructionID")
	command := flag.String("command", "rap", "command (rap)")

	flag.Parse()

	testSupport, err := support.DefaultSupport(*af, *cf)
	if err != nil {
		utils.FatalWithCause(err)
	}

	// get teeID from proxy
	teeInfo, err := utils.TeeInfo(*pf)
	if err != nil {
		utils.FatalWithCause(err)
	}

	teeID, _, err := utils.TeeProxyId(teeInfo)
	if err != nil {
		utils.FatalWithCause(err)
	}

	ftdcTeeID, _, err := utils.GetTeeProxyID(*epf)
	if err != nil {
		utils.FatalWithCause(err)
	}

	// to check if things are ok
	_, _, err = utils.GetCodeHashAndPlatform(teeInfo, *lf)
	if err != nil {
		utils.FatalWithCause(err)
	}

	logger.Infof("Registration of TEE with ID %s", hex.EncodeToString(teeID[:]))
	err = utils.RegisterNode(testSupport, teeInfo, *pf, *epf, ftdcTeeID, *command, *instructionF)
	if err != nil {
		utils.FatalWithCause(err)
	}

	logger.Infof("Registered TEE node with id %s", teeID)
}
