/*
Copyright © 2020 Celo Org

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"os"
	"time"

	"github.com/celo-org/rosetta/celo/client"
	"github.com/celo-org/rosetta/internal/signals"
	"github.com/celo-org/rosetta/service"
	"github.com/celo-org/rosetta/service/geth"
	"github.com/celo-org/rosetta/service/rpc"
	"github.com/ethereum/go-ethereum/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// localCmd represents the local command
var localCmd = &cobra.Command{
	Use:   "local",
	Short: "Launch Rosetta using a local celo-blockchain node",
	Long:  `On this mode, rosetta will spawn a celo-blockchain node`,
	Run:   runLocalCmd,
}

var gethBinary string
var staticNodes []string

func init() {
	serveCmd.AddCommand(localCmd)

	localCmd.Flags().String("geth", "", "Path to the celo-blockchain binary")
	exitOnError(viper.BindPFlag("geth", localCmd.Flags().Lookup("geth")))
	exitOnError(localCmd.MarkFlagFilename("geth"))

	localCmd.Flags().String("genesis", "", "path to the genesis.json")
	exitOnError(viper.BindPFlag("genesis", localCmd.Flags().Lookup("genesis")))
	exitOnError(localCmd.MarkFlagFilename("genesis", "json"))

	localCmd.Flags().StringArrayVar(&staticNodes, "staticNode", []string{}, "StaticNode to use (can be repeated many times")
	exitOnError(localCmd.MarkFlagRequired("staticNode"))
}

func runLocalCmd(cmd *cobra.Command, args []string) {
	exitOnMissingConfig(cmd, "geth")
	exitOnMissingConfig(cmd, "genesis")

	gethBinary = viper.GetString("geth")
	genesisPath := viper.GetString("genesis")

	gethSrv := geth.NewGethService(
		gethBinary,
		datadir.GethDatadir(),
		genesisPath,
		staticNodes,
	)

	if err := gethSrv.Setup(); err != nil {
		log.Error("Error on geth setup", "err", err)
		os.Exit(1)
	}

	chainParams := gethSrv.ChainParameters()

	rpcService := service.WithDelay(service.LazyService("rosetta-rpc", func() service.Service {
		log.Info("Initializing Rosetta In Local Mode..", "chainId", chainParams.ChainId, "epochSize", chainParams.EpochSize)
		cc, err := client.Dial(gethSrv.IpcFilePath())
		if err != nil {
			log.Crit("Error on client connection to geth", "err", err)
		}

		return rpc.NewRosettaServer(cc, &rosettaRpcConfig, chainParams)
	}), 5*time.Second)

	srvCtx, stopServices := context.WithCancel(context.Background())
	defer stopServices()

	gotExitSignal := signals.WatchForExitSignals()
	go func() {
		<-gotExitSignal
		stopServices()
	}()

	if err := service.RunServices(srvCtx, gethSrv, rpcService); err != nil {
		log.Error("Error running services", "err", err)
	}
}
