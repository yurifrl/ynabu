package config

import (
	"os"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type YNABConfig struct {
	AccountID string `mapstructure:"account_id"`
	Token     string `mapstructure:"token"`
}

type Config struct {
	Port     string     `mapstructure:"port"`
	LogLevel string     `mapstructure:"log_level"`
	YNAB     YNABConfig `mapstructure:"ynab"`
}

// Load initialises a Viper instance, reads the config file (if any) and returns it.
// No defaults or unmarshalling are performed here – this keeps I/O in one place.
func Load(cfgFile string) (*viper.Viper, error) {
	v := viper.New()
	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
	}

	// Ignore not-found error; caller can decide if it's fatal.
	_ = v.ReadInConfig()
	return v, nil
}

func Build(cfgFile string, fs *pflag.FlagSet) (*Config, error) {
	v, err := Load(cfgFile)
	if err != nil {
		return nil, err
	}
	if fs != nil {
		_ = v.BindPFlags(fs)
	}
	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, err
	}
	if c.Port == "" {
		c.Port = "3000"
	}

	c.YNAB.Token = os.ExpandEnv(c.YNAB.Token)

	return &c, nil
}
