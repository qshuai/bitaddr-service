package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"time"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/go-xorm/xorm"
	"github.com/qshuai/bitaddr/models"
	"github.com/qshuai/tcolor"
	"github.com/syndtr/goleveldb/leveldb"
)

type Engine struct {
	coindb *models.CoinDB
	addrdb *xorm.Engine
	client *rpcclient.Client
	net    *chaincfg.Params
}

func (engine *Engine) start() {
	exitFunc := func(err error) {
		engine.addrdb.Close()
		engine.coindb.Close()
		engine.client.Shutdown()
		fmt.Println(tcolor.WithColor(tcolor.Red, "Release resource after encountering error: "+err.Error()))
		os.Exit(1)
	}

	var height int64
	for {
		start := time.Now()
		rpcT1 := time.Now()
		blockHash, err := engine.client.GetBlockHash(height)
		if err != nil {
			exitFunc(err)
		}

		block, err := engine.client.GetBlock(blockHash)
		if err != nil {
			exitFunc(err)
		}
		rpcT2 := time.Now()

		var coinDBT, dbT float64
		for _, tx := range block.Transactions {
			batch := make([]models.Tuple, 0, len(tx.TxIn)+len(tx.TxOut))
			txhash := tx.TxHash()

			if !blockchain.IsCoinBaseTx(tx) {
				for _, input := range tx.TxIn {
					key := getKey(&input.PreviousOutPoint)
					coinDBT1 := time.Now()
					v, err := engine.coindb.DB.Get(key, nil)
					coinDBT2 := time.Now()
					if err == leveldb.ErrNotFound {
						continue
					} else if err != nil {
						exitFunc(err)
					}

					coinDBT3 := time.Now()
					err = engine.coindb.DB.Delete(key, nil)
					coinDBT4 := time.Now()
					if err != nil {
						exitFunc(err)
					}
					coinDBT += coinDBT2.Sub(coinDBT1).Seconds() + coinDBT4.Sub(coinDBT3).Seconds()

					dbT1 := time.Now()
					_, err = engine.addrdb.Exec("INSERT INTO address (`address`, `withdraw`, `last_block`) VALUES"+
						"(?, 1, ?) ON DUPLICATE KEY UPDATE `withdraw` = `withdraw` + 1", string(v), height)
					dbT2 := time.Now()
					if err != nil {
						exitFunc(err)
					}
					dbT += dbT2.Sub(dbT1).Seconds()
				}
			}

			for idx, output := range tx.TxOut {
				_, addrs, _, err := txscript.ExtractPkScriptAddrs(output.PkScript, engine.net)
				if err != nil {
					exitFunc(err)
				}
				if len(addrs) >= 1 {
					key := make([]byte, 36)
					copy(key, txhash[:])
					binary.LittleEndian.PutUint32(key[chainhash.HashSize:], uint32(idx))
					batch = append(batch, models.Tuple{
						Key:   key,
						Value: []byte(addrs[0].EncodeAddress()),
					})

					dbT1 := time.Now()
					_, err = engine.addrdb.Exec("INSERT INTO address (`address`, `deposit`, `last_block`) VALUES"+
						"(?, 1, ?) ON DUPLICATE KEY UPDATE `deposit` = `deposit` + 1", addrs[0].EncodeAddress(), height)
					dbT2 := time.Now()
					if err != nil {
						exitFunc(err)
					}
					dbT += dbT2.Sub(dbT1).Seconds()
				}
			}

			coinDBT1 := time.Now()
			err = engine.coindb.Batch(batch)
			coinDBT2 := time.Now()
			if err != nil {
				exitFunc(err)
			}
			coinDBT += coinDBT2.Sub(coinDBT1).Seconds()
		}

		end := time.Now()
		str := fmt.Sprintf("Handled block hash: %s, height: %d, txs: %d, total_time(s): %f, rpc_time(s): %f, coindb_time(s): %f, db_time(s): %f",
			blockHash.String(), height, len(block.Transactions), end.Sub(start).Seconds(), rpcT2.Sub(rpcT1).Seconds(), coinDBT, dbT)
		fmt.Println(tcolor.WithColor(tcolor.Green, str))

		// sync the next block
		height++
	}
}

func getKey(prev *wire.OutPoint) []byte {
	hash := prev.Hash
	ret := make([]byte, 36)
	copy(ret, hash[:])
	binary.LittleEndian.PutUint32(ret[chainhash.HashSize:], uint32(prev.Index))
	return ret
}

func NewEngine(config *AppConfig) (*Engine, error) {
	coindb, err := models.NewCoinDB(config.LevelDB.DBPath)
	if err != nil {
		return nil, err
	}

	db, err := models.NewAddressDB(&models.DBConfig{
		User:   config.DB.User,
		PassWD: config.DB.Pass,
		Host:   config.DB.Host,
		DBName: config.DB.DBName,
	})
	if err != nil {
		return nil, err
	}

	rpc, err := rpcclient.New(&rpcclient.ConnConfig{
		User:         config.RPC.RPCUser,
		Pass:         config.RPC.RPCPass,
		Host:         config.RPC.RPCHost,
		DisableTLS:   true,
		HTTPPostMode: true,
	}, nil)
	if err != nil {
		return nil, err
	}

	return &Engine{
		coindb: coindb,
		addrdb: db,
		client: rpc,
		net:    &chaincfg.TestNet3Params,
	}, nil
}
