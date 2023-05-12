package config

import (
	"fmt"

	"github.com/spf13/viper"
)

func EUProdViperInit() {
	viper.SetConfigName("euprodconfig")
	viper.AddConfigPath("../")
	viper.AutomaticEnv()
	viper.SetConfigType("yml")

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading config file, %s", err)
	}

}
