package config

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	Port string `mapstructure:"port"`
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
	return &c, nil
}
