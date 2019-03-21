package main

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	aclRules = "aclRules"
)

var defaultAclRules = map[string]string{
	"Allow All": "action=reflect",
}

var viperConfig *viper.Viper

func initConfig() {
	viperConfig = viper.New()
	viperConfig.SetConfigName("config")
	viperConfig.AddConfigPath("/etc/vppagent-firewall/")

	viperConfig.SetDefault(aclRules, defaultAclRules)

	err := viperConfig.ReadInConfig()
	if err != nil {
		logrus.Errorf("Error reading the config file: %s \n", err)
	}

	viperConfig.WatchConfig()
	viperConfig.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config file changed:", e.Name)
	})
	logrus.Infof("Firewall config finished")
}

func getAclRulesConfig() map[string]string {
	return viperConfig.GetStringMapString(aclRules)
}
