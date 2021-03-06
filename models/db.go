package models

import (
	"github.com/go-xorm/xorm"

	_ "github.com/go-sql-driver/mysql"
)

type Address struct {
	Address   string `xorm:"unique not null varchar(50)"`
	Deposit   int    `xorm:"default(0)"`
	Withdraw  int    `xorm:"default(0)"`
	LastBlock int32  `xorm:"last_block not null"`
}

type DBConfig struct {
	User   string
	PassWD string
	Host   string // ip + port
	DBName string
}

func NewAddressDB(config *DBConfig) (*xorm.Engine, error) {
	dataSource := config.User + ":" + config.PassWD + "@tcp(" + config.Host + ")/" + config.DBName + "?charset=utf8"
	engine, err := xorm.NewEngine("mysql", dataSource)
	if err != nil {
		return nil, err
	}

	// initial table if not exist
	err = engine.Sync2(new(Address))
	if err != nil {
		return nil, err
	}

	return engine, nil
}
