package mcfg

import (
	"fmt"
	"testing"
	"time"
)

type Cfg struct {
	Server struct {
		Mode string `viper:"mode"`
		Port int    `viper:"port"`
	} `viper:"server"`
	DB struct {
		Host string `viper:"host"`
	} `viper:"db"`
}

func TestInitViperConfigFile(t *testing.T) {
	var cfg = new(Cfg)
	err := InitViperConfigFile("test.toml", cfg, "viper")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(cfg.DB.Host)
	time.Sleep(30 * time.Second)
	fmt.Println(cfg.DB.Host)
}
