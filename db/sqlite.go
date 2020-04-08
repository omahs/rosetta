package db

import (
	"context"
	"database/sql"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	_ "github.com/mattn/go-sqlite3"
)

type rosettaSqlDb struct {
	db *sql.DB
}

const (
	setLastPersistedBlockStmt  = "update stats set lastPersistedBlock = ?"
	setGasPriceMinimumOnStmt   = "insert into gasPriceMinimum (fromBlock, val) values (?, ?, ?)"
	setRegisteredAddressOnStmt = "insert into registryAddresses (contract, fromBlock, fromTx, address) values (?, ?, ?, ?)"
)

func NewSqliteDb(dbpath string) (*rosettaSqlDb, error) {
	db, err := sql.Open("sqlite3", dbpath)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec("create table if not exists registryAddresses (contract text, fromBlock integer, fromTx integer, address blob)"); err != nil {
		return nil, err
	}

	if _, err := db.Exec("create table if not exists gasPriceMinimum (fromBlock integer, val integer)"); err != nil {
		return nil, err
	}

	_, err = db.Exec(
		`CREATE table IF NOT EXISTS stats (
			Lock char(1) not null DEFAULT 'X',
			lastPersistedBlock integer not null DEFAULT 0,
			constraint pk_stats PRIMARY KEY (Lock),
			constraint ck_stats_locked CHECK (Lock='X'))`)
	if err != nil {
		return nil, err
	}

	return &rosettaSqlDb{
		db: db,
	}, nil
}

func (cs *rosettaSqlDb) LastPersistedBlock(ctx context.Context) (*big.Int, error) {
	rows, err := cs.db.QueryContext(ctx, "select lastPersistedBlock from stats")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var block int64 // TODO: Figure out better (safer) way of storing bigints
	if rows.Next() {
		if err := rows.Scan(&block); err != nil {
			return nil, err
		}
		log.Info("Last Persisted Block Found", "block", block)
		return big.NewInt(block), nil
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return big.NewInt(0), nil
}

func (cs *rosettaSqlDb) GasPriceMinimunOn(ctx context.Context, block *big.Int) (*big.Int, error) {
	rows, err := cs.db.QueryContext(ctx, "select val from gasPriceMinimum where fromBlock <= ? order by desc fromblock limit 1", block.Int64())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var value int64
	if rows.Next() {
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		log.Info("Gas Price Minimum Found", "block", block.Int64(), "val", value)
		return big.NewInt(value), nil
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return big.NewInt(0), nil
}

func (cs *rosettaSqlDb) RegistryAddressOn(ctx context.Context, block *big.Int, txIndex uint, contractName string) (common.Address, error) {
	rows, err := cs.db.QueryContext(ctx, "select address from registryAddresses where id == ? and fromBlock <= ? and fromTx <= ? order by desc fromblock, fromTx limit 1", contractName, block.Int64(), int64(txIndex))
	if err != nil {
		return common.ZeroAddress, err
	}
	defer rows.Close()

	var address common.Address
	if rows.Next() {
		if err := rows.Scan(&address); err != nil {
			return common.ZeroAddress, err
		}
		log.Info("Registry Address Found", "contract", contractName, "address", address)
		return address, nil
	}

	if rows.Err() != nil {
		return common.ZeroAddress, rows.Err()
	}

	return common.ZeroAddress, ErrContractNotFound
}

func (cs *rosettaSqlDb) RegistryAddressesOn(ctx context.Context, block *big.Int, txIndex uint, contractNames ...string) (map[string]common.Address, error) {
	addresses := make(map[string]common.Address)
	// TODO: Could this be done more efficiently, perhaps concurrently?
	for _, name := range contractNames {
		address, err := cs.RegistryAddressOn(ctx, block, txIndex, name)
		if err == ErrContractNotFound {
			continue
		} else if err != nil {
			return nil, err
		}
		addresses[name] = address
	}
	return addresses, nil
}

func (cs *rosettaSqlDb) ApplyChanges(ctx context.Context, changeSet *BlockChangeSet) error {

	//TODO: check if this is the right isolation level, or should keep default
	tx, err := cs.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, setLastPersistedBlockStmt, changeSet.BlockNumber.Int64(), changeSet.BlockNumber.Int64()); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return rollbackErr
		}
		return err
	}
	if changeSet.GasPriceMinimun != nil {
		if _, err := tx.ExecContext(ctx, setGasPriceMinimumOnStmt, changeSet.BlockNumber.Int64(), changeSet.GasPriceMinimun.Int64()); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				return rollbackErr
			}
			return err
		}
	}

	setRegisteredAddressOnStmtPrep, err := tx.PrepareContext(ctx, setRegisteredAddressOnStmt)
	if err != nil {
		return err
	}

	for _, rc := range changeSet.RegistryChanges {
		if _, err := setRegisteredAddressOnStmtPrep.ExecContext(ctx, rc.Contract, changeSet.BlockNumber.Int64(), int64(rc.TxIndex), rc.NewAddress); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				return rollbackErr
			}
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (cs *rosettaSqlDb) setLastPersistedBlock(ctx context.Context, block *big.Int) error {
	_, err := cs.db.ExecContext(ctx, setLastPersistedBlockStmt, block.Int64(), block.Int64())
	if err != nil {
		return err
	}
	return nil
}

func (cs *rosettaSqlDb) setGasPriceMinimumOn(ctx context.Context, block, gasPriceMinimum *big.Int) error {
	_, err := cs.db.ExecContext(ctx, setGasPriceMinimumOnStmt, block.Int64(), gasPriceMinimum.Int64())
	if err != nil {
		return err
	}
	return nil
}

func (cs *rosettaSqlDb) setRegisteredAddressOn(ctx context.Context, contractName string, blockNumber *big.Int, txIndex uint, address common.Address) error {
	_, err := cs.db.ExecContext(ctx, setRegisteredAddressOnStmt, contractName, blockNumber.Int64(), int64(txIndex), address)
	if err != nil {
		return err
	}
	return nil
}
