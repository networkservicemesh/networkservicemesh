package logger

import "github.com/sirupsen/logrus"

var defaultConfigEntry = ConfigEntry{
	Level:    logrus.InfoLevel,
	Selector: nil,
}

var defaultConfig = Config{
	ConfigEntries: []ConfigEntry{defaultConfigEntry},
}

// Config consists of:
//    ConfigEntries - A list of ConfigEntry objects.
type Config struct {
	ConfigEntries []ConfigEntry
}

// ConfigEntry - Each Config Entry specifies
//     Selector: A map of key, value pairs to match against
//               logger.Plugin.Fields.  A Selector matches iff
//               all key,value pairs in the selector are also
//               present in config.Plugin.Fields
//     Level:    Log level to set if the ConfigEntry is selected
//               if the config entry has the highest len(ConfigEntry.Selector)
//               of all Config.ConfigEntries.  In the event of two
//               ConfigEntry with the same len(ConfigEntry.Selector)
//               in the same Config.ConfigEntries, the first occurence is
//               selected
type ConfigEntry struct {
	Level    logrus.Level
	Selector map[string]string
}
