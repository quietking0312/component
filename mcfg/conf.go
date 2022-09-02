package mcfg

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"os"
	"time"
)

// viper bug OnConfigChange 会回调2次
// https://github.com/spf13/viper/issues?q=OnConfigChange
var lastChangeTime time.Time

func init() {
	lastChangeTime = time.Now()
}

func InitViperConfigFile(cfgFile string, cfg interface{}, tagName string) error {
	if tagName == "" {
		tagName = "viper"
	}
	cfgObj := viper.New()
	_, err := os.Stat(cfgFile)
	if err == nil {
		cfgObj.SetConfigFile(cfgFile)
	} else {
		cfgObj.SetConfigFile(cfgFile)
		cfgObj.AddConfigPath(".")
	}
	if err := cfgObj.ReadInConfig(); err != nil {
		return fmt.Errorf("read config failed: %v", err)
	}
	if err := cfgObj.Unmarshal(cfg, func(c *mapstructure.DecoderConfig) {
		c.TagName = tagName
	}); err != nil {
		return fmt.Errorf("config unmarshal filed: %v", err)
	}
	cfgObj.WatchConfig()
	cfgObj.OnConfigChange(func(in fsnotify.Event) {
		if time.Now().Sub(lastChangeTime).Seconds() >= 1 {
			if err := cfgObj.Unmarshal(cfg, func(c *mapstructure.DecoderConfig) {
				c.TagName = tagName
			}); err != nil {
				fmt.Println(fmt.Errorf("config unmarshal filed: %v", err))
			}
			lastChangeTime = time.Now()
			fmt.Printf("配置信息： %+v\n", cfg)
		}
	})
	fmt.Printf("配置信息：%+v\n", cfg)
	return nil
}
