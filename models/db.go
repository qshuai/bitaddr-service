package models

import (
	"github.com/go-xorm/xorm"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type Address struct {
	Id int64
	Address string `xorm:"unique index not null varchar(50)"`
	Deposit int	`xorm:"default(0)"`
	Withdraw int `xorm:"default(0)"`
	LastUpdate time.Time `xorm:"last_update updated not null"`
}

type DBConfig struct {
	User string
	PassWD string
	Host string  // ip + port
	DBName string
}

func NewAddressDB(config *DBConfig) (*xorm.Engine, error) {
	dataSource := config.User +":"+ config.PassWD +"@tcp(" + config.Host +")/"+config.DBName +"?charset=utf8"
	engine, err :=  xorm.NewEngine("mysql", dataSource)
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
