package utilities

var defaultConfigEntry = ConfigEntry{}

var defaultConfig = Config{
	ConfigEntries: []ConfigEntry{defaultConfigEntry},
}

// Config consists of:
//    ConfigEntries - A list of ConfigEntry objects.
type Config struct {
	ConfigEntries []ConfigEntry
}

// ConfigEntry - Each Config Entry specifies
type ConfigEntry struct{}
