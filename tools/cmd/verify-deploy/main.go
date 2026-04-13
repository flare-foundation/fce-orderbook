package main

import (
	"flag"
	"fmt"
	"os"

	"extension-scaffold/tools/pkg/configs"
	"extension-scaffold/tools/pkg/support"
	"extension-scaffold/tools/pkg/validate"

	"github.com/ethereum/go-ethereum/common"
)

func main() {
	af := flag.String("a", configs.AddressesFile, "file with deployed addresses")
	cf := flag.String("c", configs.ChainNodeURL, "chain node url")
	configPath := flag.String("config", "", "path to extension.env for post-deploy validation")
	step := flag.String("step", "all", "which step to check: deploy, register, services, tee-version, tee-machine, test, or all")
	jsonOut := flag.Bool("json", false, "output as JSON instead of colored terminal")
	flag.Parse()

	s, err := support.DefaultSupport(*af, *cf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing support: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nHints:\n")
		fmt.Fprintf(os.Stderr, "  - Check that the addresses file exists: %s\n", *af)
		fmt.Fprintf(os.Stderr, "  - Check that the chain node is running: %s\n", *cf)
		fmt.Fprintf(os.Stderr, "  - If using a custom key, ensure DEPLOYMENT_PRIVATE_KEY is set in .env\n")
		os.Exit(1)
	}

	addresses := map[string]common.Address{
		"TeeExtensionRegistry": s.Addresses.TeeExtensionRegistry,
		"TeeMachineRegistry":   s.Addresses.TeeMachineRegistry,
	}

	report := &validate.Report{}

	switch *step {
	case "deploy":
		validate.RegisterDeployChecks(report, s.ChainClient, s.Prv, addresses, *configPath)
	case "register":
		validate.RegisterRegistrationChecks(report, s.ChainClient, s.Prv, s.TeeExtensionRegistry, s.TeeOwnerAllowlist, *configPath)
	case "services":
		validate.RegisterServicesChecks(report, *configPath)
	case "tee-version":
		validate.RegisterTeeVersionChecks(report, *configPath)
	case "tee-machine":
		validate.RegisterTeeMachineChecks(report, *configPath)
	case "test":
		validate.RegisterTestChecks(report, *configPath)
	case "all":
		validate.RegisterDeployChecks(report, s.ChainClient, s.Prv, addresses, *configPath)
		validate.RegisterRegistrationChecks(report, s.ChainClient, s.Prv, s.TeeExtensionRegistry, s.TeeOwnerAllowlist, *configPath)
		validate.RegisterServicesChecks(report, *configPath)
		validate.RegisterTeeVersionChecks(report, *configPath)
		validate.RegisterTeeMachineChecks(report, *configPath)
		validate.RegisterTestChecks(report, *configPath)
	default:
		fmt.Fprintf(os.Stderr, "Unknown step: %q\n", *step)
		fmt.Fprintf(os.Stderr, "Valid steps: deploy, register, services, tee-version, tee-machine, test, all\n")
		os.Exit(1)
	}

	if *jsonOut {
		report.PrintJSON()
	} else {
		report.Print()
	}

	if report.HasFailures() {
		os.Exit(1)
	}
}
