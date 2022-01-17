package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

func InitConfig(ProjectName, configureSuggestion string) {
	if viper.GetString("config") == "" {
		viper.AddConfigPath(fmt.Sprintf("$HOME/%s/configs", ProjectName))
		viper.AddConfigPath(".")
		viper.AddConfigPath("./configs")
		viper.SetConfigName(fmt.Sprintf("%s.config", ProjectName))
		viper.SetConfigType("json")
	} else {
		viper.SetConfigType("json")
		viper.SetConfigFile(viper.GetString("config"))
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error: %v\n%s", err, configureSuggestion)
		os.Exit(1)
	}

	fmt.Println(viper.Get("domain"))

	fmt.Println(viper.ConfigFileUsed())
	// check flag, if no flag, check default, otherwise recommend config command
}
