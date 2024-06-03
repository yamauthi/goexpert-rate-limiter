package configs

import "github.com/spf13/viper"

type Config struct {
	DefaultLimitType       int    `mapstructure:"DEFAULT_LIMIT_TYPE"`
	DefaultRequestsLimit   int    `mapstructure:"DEFAULT_REQUESTS_LIMIT"`
	DefaultClientBlockTime int    `mapstructure:"DEFAULT_CLIENT_BLOCK_TIME"`
	DBHost                 string `mapstructure:"DB_HOST"`
	DBPort                 string `mapstructure:"DB_PORT"`
	DBPassword             string `mapstructure:"DB_PASSWORD"`
}

func LoadConfig(path string) (*Config, error) {
	var conf *Config

	viper.SetConfigName("app_config")
	viper.SetConfigType("env")
	viper.AddConfigPath(path)
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	err := viper.ReadInConfig()

	if err != nil {
		return nil, err
	}

	err = viper.Unmarshal(&conf)
	if err != nil {
		return nil, err
	}

	return conf, err
}
