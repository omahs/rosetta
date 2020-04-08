package main

import (
	"context"
	"os"
	"time"

	"github.com/celo-org/rosetta/celo/client"
	"github.com/celo-org/rosetta/db"
	"github.com/celo-org/rosetta/internal/signals"
	"github.com/celo-org/rosetta/service"
	"github.com/celo-org/rosetta/service/geth"
	mservice "github.com/celo-org/rosetta/service/monitor"
	"github.com/ethereum/go-ethereum/log"
)

// Configuration Parameters
var (
	gethBinary  = "../bin/rc0/geth"
	genesis     = "./envs/rc0/genesis.json"
	datadir     = "./envs/rc0/celo"
	sqlitepath  = "./envs/rc0/rosetta.db"
	staticNodes = []string{
		"enode://33ac194052ccd10ce54101c8340dbbe7831de02a3e7dcbca7fd35832ff8c53a72fd75e57ce8c8e73a0ace650dc2c2ec1e36f0440e904bc20a3cf5927f2323e85@34.83.199.225:30303",
	}
)

func runMonitorWithGeth(ctx context.Context) error {
	gethSrv := geth.NewGethService(gethBinary, datadir, genesis, staticNodes)

	if err := gethSrv.Setup(); err != nil {
		log.Error("Error on geth setup", "err", err)
		return err
	}

	chainParams := gethSrv.ChainParameters()
	log.Info("Detected Chain Parameters", "chainId", chainParams.ChainId, "epochSize", chainParams.EpochSize)

	nodeUri := gethSrv.IpcFilePath()
	log.Debug("celo nodes ipc file", "filepath", nodeUri)

	celoStore, err := db.NewSqliteDb(sqlitepath)
	if err != nil {
		log.Error("Error opening CeloStore", "err", err)
		return err
	}

	sm := service.NewServiceManager(ctx)

	sm.Add(gethSrv)

	time.Sleep(5 * time.Second)

	cc, err := client.Dial(nodeUri)
	if err != nil {
		log.Error("Error on client connection to geth", "err", err)
		return err
	}

	monitorSrv := mservice.NewMonitorService(cc, celoStore)

	sm.Add(monitorSrv)

	if err = sm.Wait(); err != nil {
		log.Error("Error running Services", "err", err)
		return err
	}
	return nil
}

func main() {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlDebug, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))

	ctx, stopFn := context.WithCancel(context.Background())
	defer stopFn()

	// run the monitor on the background
	go runMonitorWithGeth(ctx)

	// wait a few seconds for everything to start
	// time.Sleep(10 * time.Second)

	gotExitSignal := signals.WatchForExitSignals()
	<-gotExitSignal

	// ADD your code here

}
