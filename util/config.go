package util

import "github.com/spf13/viper"

type Config struct {
	MongoURL string `mapstructure:"MONGO_URL"`
	PgURL    string `mapstructure:"PG_URL"`
}

// LoadConfig read configuration from file or env variables
func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}
