package main

type AppConfig struct {
	// leveldb
	LevelDB struct{
		DBPath string `mapstructure:"dbpath"`
	}

	// mysql
	DB struct{
		User   string `mapstructure:"user"`
		Pass   string `mapstructure:"pass"`
		Host   string `mapstructure:"host"`
		DBName string `mapstructure:"db"`
	}

	// rpc
	RPC struct {
		RPCUser string `mapstructure:"user"`
		RPCPass string `mapstructure:"pass"`
		RPCHost string `mapstructure:"host"`
	}
}
