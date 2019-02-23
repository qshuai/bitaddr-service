package models

import (
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
)

// Specification:
// key:value
// hash + index : address
// 32 bytes + 4 bytes : 1+

type Tuple struct {
	Key []byte
	Value []byte
}

type CoinDB struct {
	DB *leveldb.DB
}

func (db *CoinDB)Batch(set []Tuple)  error {
	var err error

	batch := new(leveldb.Batch)
	for _,item := range set {
		batch.Put(item.Key, item.Value)
	}

	err = db.DB.Write(batch, nil)
	if err != nil {
		db.Close()
		return err
	}

	return nil
}

func (db *CoinDB)Close() {
	err := db.DB.Close()
	if err != nil {
		fmt.Printf("close leveldb failed: %s\n", err)
	}
}

func NewCoinDB(path string) (*CoinDB, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}

	return &CoinDB{
		DB:db,
	}, nil
}