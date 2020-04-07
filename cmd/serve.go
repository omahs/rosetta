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
	"os"
	"path/filepath"
	"time"

	"github.com/celo-org/rosetta/service/rpc"
	"github.com/ethereum/go-ethereum/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:              "serve",
	Short:            "Start rosetta server",
	Args:             cobra.NoArgs,
	PersistentPreRun: validateDatadir,
}

var rosettaRpcConfig rpc.RosettaServerConfig

type ConfigPaths string

// var _datadir string
var datadir ConfigPaths

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.PersistentFlags().UintVar(&rosettaRpcConfig.Port, "port", 8080, "Listening port for http server")
	serveCmd.PersistentFlags().StringVar(&rosettaRpcConfig.Interface, "address", "", "Listening address for http server")
	serveCmd.PersistentFlags().DurationVar(&rosettaRpcConfig.RequestTimeout, "reqTimeout", 25*time.Second, "Timeout when serving a request")

	serveCmd.PersistentFlags().String("datadir", "", "datadir to use")
	exitOnError(viper.BindPFlag("datadir", serveCmd.PersistentFlags().Lookup("datadir")))
	exitOnError(serveCmd.MarkPersistentFlagDirname("datadir"))
}

func validateDatadir(cmd *cobra.Command, args []string) {
	exitOnMissingConfig(cmd, "datadir")

	absDatadir, err := filepath.Abs(viper.GetString("datadir"))
	if err != nil {
		log.Crit("Can't resolve datadir path", "datadir", absDatadir, "err", err)
	}

	stat, err := os.Stat(absDatadir)
	switch {
	case err != nil:
		log.Crit("Can't access datadir", "datadir", absDatadir, "err", err)
	case !stat.IsDir():
		log.Crit("Datadir is not a directory", "datadir", absDatadir)
	}
	datadir = ConfigPaths(absDatadir)
	log.Info("DataDir Configured", "datadir", datadir)
}

func (g ConfigPaths) Datadir() string {
	return string(g)
}

func (g ConfigPaths) GethDatadir() string {
	return filepath.Join(string(g), "celo")
}
