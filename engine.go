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
	"github.com/jmoiron/sqlx"
	"github.com/qshuai/bitaddr/models"
	"github.com/qshuai/tcolor"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/thinkeridea/go-extend/exbytes"
	"github.com/thinkeridea/go-extend/pool"
)

type Engine struct {
	coindb *models.CoinDB
	// addrdb *xorm.Engine
	addrdb *sqlx.DB
	client *rpcclient.Client
	net    *chaincfg.Params
}

var bufPool = pool.GetBuff8192()

func (engine *Engine) start() {
	exitFunc := func(err error) {
		engine.addrdb.Close()
		engine.coindb.Close()
		engine.client.Shutdown()
		fmt.Println(tcolor.WithColor(tcolor.Red, "Release resource after encountering error: "+err.Error()))
		os.Exit(1)
	}

	buf := bufPool.Get()
	defer bufPool.Put(buf)

	args := make([]interface{}, 0, 8000)
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
		var ins, outs int

		for _, tx := range block.Transactions {
			batch := make([]models.Tuple, 0, len(tx.TxIn)+len(tx.TxOut))
			txhash := tx.TxHash()

			buf.Reset()
			buf.WriteString("INSERT INTO address (`address`, `withdraw`, `last_block`) VALUES")
			args = args[0:0]
			if !blockchain.IsCoinBaseTx(tx) {
				for _, input := range tx.TxIn {
					ins++

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

					// dbT1 := time.Now()
					// exist, err := engine.addrdb.Where("address = ?", string(v)).Exist(&models.Address{})
					// if err != nil {
					// 	exitFunc(err)
					// }
					// if exist {
					// 	_, err = engine.addrdb.Exec("UPDATE `address` SET `withdraw` = `withdraw` + 1 where `address` = ?", string(v))
					// 	if err != nil {
					// 		exitFunc(err)
					// 	}
					// } else {
					// 	_, err = engine.addrdb.Exec("INSERT INTO `address` (`address`, `withdraw`, `last_block`) VALUES (?, 1, ?)", string(v), height)
					// 	if err != nil {
					// 		exitFunc(err)
					// 	}
					// }

					if len(args) > 0 {
						buf.WriteByte(',')
					}
					buf.WriteString("(?, 1, ?)")
					args = append(args, exbytes.ToString(v), height)
					// engine.addrdb.
					// 	_, err = engine.addrdb.Exec("INSERT INTO address (`address`, `withdraw`, `last_block`) VALUES"+
					// 	"(?, 1, ?) ON DUPLICATE KEY UPDATE `withdraw` = `withdraw` + 1", string(v), height)
					// dbT2 := time.Now()
					// if err != nil {
					// 	exitFunc(err)
					// }
					// dbT += dbT2.Sub(dbT1).Seconds()
				}
			}

			if len(args) > 0 {
				buf.WriteString("ON DUPLICATE KEY UPDATE `withdraw` = `withdraw` + 1")

				dbT1 := time.Now()
				engine.addrdb.MustExec(exbytes.ToString(buf.Bytes()), args...)
				dbT += time.Now().Sub(dbT1).Seconds()
			}

			buf.Reset()
			buf.WriteString("INSERT INTO address (`address`, `deposit`, `last_block`) VALUES")
			args = args[:0]
			for idx, output := range tx.TxOut {
				outs++

				_, addrs, _, err := txscript.ExtractPkScriptAddrs(output.PkScript, engine.net)
				if err != nil {
					exitFunc(err)
				}
				if len(addrs) >= 1 {
					addr := addrs[0].EncodeAddress()
					key := make([]byte, 36)
					copy(key, txhash[:])
					binary.LittleEndian.PutUint32(key[chainhash.HashSize:], uint32(idx))
					batch = append(batch, models.Tuple{
						Key:   key,
						Value: []byte(addr),
					})

					// dbT1 := time.Now()
					// exist, err := engine.addrdb.Where("address = ?", string(addrs[0].EncodeAddress())).Exist(&models.Address{})
					// if err != nil {
					// 	exitFunc(err)
					// }
					// if exist {
					// 	_, err = engine.addrdb.Exec("UPDATE `address` SET `deposit` = `deposit` + 1 where `address` = ?", addr)
					// 	if err != nil {
					// 		exitFunc(err)
					// 	}
					// } else {
					// 	_, err = engine.addrdb.Exec("INSERT INTO `address` (`address`, `deposit`, `last_block`) VALUES (?, 1, ?)", addr, height)
					// 	if err != nil {
					// 		exitFunc(err)
					// 	}
					// }

					if len(args) > 0 {
						buf.WriteByte(',')
					}
					buf.WriteString("(?, 1, ?)")
					args = append(args, addr, height)

					// _, err = engine.addrdb.Exec("INSERT INTO address (`address`, `deposit`, `last_block`) VALUES"+
					// 	"(?, 1, ?) ON DUPLICATE KEY UPDATE `deposit` = `deposit` + 1", addrs[0].EncodeAddress(), height)
					// dbT2 := time.Now()
					// if err != nil {
					// 	exitFunc(err)
					// }
					// dbT += dbT2.Sub(dbT1).Seconds()
				}
			}

			if len(args) > 0 {
				buf.WriteString("ON DUPLICATE KEY UPDATE `deposit` = `deposit` + 1")

				dbT1 := time.Now()
				engine.addrdb.MustExec(exbytes.ToString(buf.Bytes()), args...)
				dbT += time.Now().Sub(dbT1).Seconds()
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
		str := fmt.Sprintf("Handled block hash: %s, height: %d, txs: %d(%d ins <> %d outs), total_time(s): %f, rpc_time(s): %f, coindb_time(s): %f, db_time(s): %f",
			blockHash.String(), height, len(block.Transactions), ins, outs, end.Sub(start).Seconds(), rpcT2.Sub(rpcT1).Seconds(), coinDBT, dbT)
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

	// db, err := models.NewAddressDB(&models.DBConfig{
	// 	User:   config.DB.User,
	// 	PassWD: config.DB.Pass,
	// 	Host:   config.DB.Host,
	// 	DBName: config.DB.DBName,
	// })
	// if err != nil {
	// 	return nil, err
	// }
	// db.SetMaxIdleConns(10)
	// db.SetMaxOpenConns(30)
	db, err := sqlx.Connect("mysql", "root:646689abc@tcp(127.0.0.1:3306)/bitaddr?parseTime=true&charset=utf8&loc=Local")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(3)
	db.SetMaxIdleConns(2)

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
