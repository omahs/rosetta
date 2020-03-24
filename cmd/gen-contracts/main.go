package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/celo-org/rosetta/internal/build"
)

const contractsPath = "contract"

var contractsToGenerate = []string{
	"Registry",
	"LockedGold",
	"Election",
	"StableToken",
}

func main() {
	monorepoPath := flag.String("monorepo", "", "Path to celo-monorepo")
	celoBlockchainPath := flag.String("gcelo", "", "Path to celo-blockchain")

	flag.Parse()
	fmt.Println(*monorepoPath, *celoBlockchainPath)

	if *monorepoPath == "" || *celoBlockchainPath == "" {
		exitWithHelpMessage()
	}

	validatePathExists(*monorepoPath)
	validatePathExists(*celoBlockchainPath)

	abigen := path.Join(*celoBlockchainPath, "build/bin", "abigen")

	for _, contract := range contractsToGenerate {
		sourceName := strings.ToLower(contract) + ".go"
		outPath := path.Join(contractsPath, sourceName)
		if pathExists(outPath) {
			if err := os.Remove(outPath); err != nil {
				exitMessage("Error removing"+outPath+": %s\n", err)
			}
		}
		contractTrufflePath := path.Join(*monorepoPath, "packages/protocol/build/contracts/", contract+".json")
		validatePathExists(contractTrufflePath)
		build.MustRunCommand(abigen, "--truffle", contractTrufflePath,
			"--pkg", contractsPath, "--type", contract,
			"--out", outPath)
	}
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func validatePathExists(dirpath string) {
	if _, err := os.Stat(dirpath); err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Path %s does not exists", dirpath)
		} else {
			fmt.Printf("Can't access %s: %s", dirpath, err)
		}
		exitWithHelpMessage()
	}
}

func exitWithHelpMessage() {
	flag.PrintDefaults()
	os.Exit(1)
}

func exitMessage(msg string, a ...interface{}) {
	fmt.Printf(msg, a...)
	os.Exit(1)
}
