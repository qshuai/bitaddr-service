package main

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/qshuai/tcolor"
	"github.com/spf13/viper"
	"os"
)

const configFile string = "./conf/app.yml"

func main() {
	// parse configuration file
	viper.SetConfigType("yaml")
	f, err := os.OpenFile(configFile, os.O_RDONLY, 0666)
	if err != nil {
		fmt.Println(tcolor.WithColor(tcolor.Red, "Open configuration file failed: " + err.Error()))
		os.Exit(1)
	}

	err = viper.ReadConfig(f)
	if err != nil {
		fmt.Println(tcolor.WithColor(tcolor.Red, "Viper parse configuration file failed: " + err.Error()))
		os.Exit(1)
	}
	var config AppConfig
	err = viper.Unmarshal(&config)
	if err != nil {
		fmt.Println(tcolor.WithColor(tcolor.Red, "Viper reflect AppConfig failed: " + err.Error()))
		os.Exit(1)
	}

	// initialize Engine
	engine ,err := NewEngine(&config)
	if err != nil {
		fmt.Println(tcolor.WithColor(tcolor.Red, "Initialize engine failed: " + err.Error()))
		os.Exit(1)
	}

	spew.Dump(engine.client.GetInfo())

	engine.start()
}

