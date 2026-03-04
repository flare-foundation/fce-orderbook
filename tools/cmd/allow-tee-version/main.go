package main

import (
	"crypto/ecdsa"
	"flag"
	"os"
	"extension-e2e/configs"
	"extension-e2e/pkg/support"
	"extension-e2e/pkg/utils"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/flare-foundation/go-flare-common/pkg/logger"
)

func main() {
	af := flag.String("a", configs.AddressesFile, "file with deployed addresses")
	cf := flag.String("c", configs.ChainNodeURL, "chain node url")
	pf := flag.String("p", configs.ExtensionProxyURL, "proxy url")
	versionF := flag.String("version", "v0.1.0", "version")
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

	var privKey *ecdsa.PrivateKey
	privKeyString := os.Getenv("EXTENSION_OWNER_KEY")
	if privKeyString != "" {
		privKey, err = crypto.HexToECDSA(privKeyString)
		if err != nil {
			utils.FatalWithCause(err)
		}
	} else {
		privKey = testSupport.Prv
	}

	teeID, _, err := utils.TeeProxyId(teeInfo)
	if err != nil {
		utils.FatalWithCause(err)
	}

	logger.Infof("registering version: %s, %s, extension: %v tee id: %s", teeInfo.MachineData.CodeHash, teeInfo.MachineData.Platform, teeInfo.MachineData.ExtensionID.Big(), teeID)
	err = utils.AddTeeVersion(testSupport, privKey, teeInfo.MachineData.ExtensionID.Big(), teeInfo.MachineData.CodeHash, teeInfo.MachineData.Platform, common.Hash{}, *versionF)
	if err != nil {
		utils.FatalWithCause(err)
	}
}
