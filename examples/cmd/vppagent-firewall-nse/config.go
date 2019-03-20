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

func initConfig() {
	viper.SetConfigName("config")                  // name of config file (without extension)
	viper.AddConfigPath("/etc/vppagent-firewall/") // path to look for the config file in

	viper.SetDefault(aclRules, defaultAclRules)

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		logrus.Errorf("Error config file: %s \n", err)
	}

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config file changed:", e.Name)
	})
	logrus.Infof("Firewall config finished")
}

func getAclRulesConfig() map[string]string {
	logrus.Infof("ACL Rules: \n %v", viper.GetStringMapString(aclRules))
	return viper.GetStringMapString(aclRules)
}
