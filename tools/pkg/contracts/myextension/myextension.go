//go:generate go run github.com/ethereum/go-ethereum/cmd/abigen --abi=MyExtensionInstructionSender.abi --bin=MyExtensionInstructionSender.bin --pkg=myextension --type=MyExtensionInstructionSender --out=autogen.go

package myextension
