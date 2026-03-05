package main

import (
	"encoding/hex"
	"flag"
	"extension-scaffold/tools/pkg/configs"
	"extension-scaffold/tools/pkg/fccutils"
	"extension-scaffold/tools/pkg/support"

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
		fccutils.FatalWithCause(err)
	}

	// get teeID from proxy
	teeInfo, err := fccutils.TeeInfo(*pf)
	if err != nil {
		fccutils.FatalWithCause(err)
	}

	teeID, _, err := fccutils.TeeProxyId(teeInfo)
	if err != nil {
		fccutils.FatalWithCause(err)
	}

	ftdcTeeID, _, err := fccutils.GetTeeProxyID(*epf)
	if err != nil {
		fccutils.FatalWithCause(err)
	}

	// to check if things are ok
	_, _, err = fccutils.GetCodeHashAndPlatform(teeInfo, *lf)
	if err != nil {
		fccutils.FatalWithCause(err)
	}

	logger.Infof("Registration of TEE with ID %s", hex.EncodeToString(teeID[:]))
	err = fccutils.RegisterNode(testSupport, teeInfo, *pf, *epf, ftdcTeeID, *command, *instructionF)
	if err != nil {
		fccutils.FatalWithCause(err)
	}

	logger.Infof("Registered TEE node with id %s", teeID)
}
