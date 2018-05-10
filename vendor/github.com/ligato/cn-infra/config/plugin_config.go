package config

import (
	"os"
	"path"
	"strings"
	"sync"

	"github.com/ligato/cn-infra/logging/logrus"
	"github.com/namsral/flag"
)

// FlagSuffix is added to plugin name while loading plugins configuration.
const FlagSuffix = "-config"

// EnvSuffix is added to plugin name while loading plugins configuration from ENV variable.
const EnvSuffix = "_CONFIG"

// DirFlag as flag name (see implementation in declareFlags())
// is used to define default directory where config files reside.
// This flag name is derived from the name of the plugin.
const DirFlag = "config-dir"

// DirDefault holds a default value "." for flag, which represents current working directory.
const DirDefault = "."

// DirUsage used as a flag (see implementation in declareFlags()).
const DirUsage = "Location of the configuration files; also set via 'CONFIG_DIR' env variable."

// PluginConfig is API for plugins to access configuration.
//
// Aim of this API is to let a particular plugin to bind it's configuration
// without knowing a particular key name. The key name is injected in flavor (Plugin Name).
type PluginConfig interface {
	// GetValue parses configuration for a plugin and stores the results in data.
	// The argument data is a pointer to an instance of a go structure.
	GetValue(data interface{}) (found bool, err error)

	// GetConfigName returns config name derived from plugin name:
	// flag = PluginName + FlagSuffix (evaluated most often as absolute path to a config file)
	GetConfigName() string
}

// ForPlugin returns API that is injectable to a particular Plugin
// and is used to read it's configuration.
//
// It tries to lookup `plugin + "-config"` in flags and declare
// the flag if it still not exists. It uses the following
// opts (used to define flag (if it was not already defined)):
// - default value
// - usage
func ForPlugin(pluginName string, opts ...string) PluginConfig {
	flgName := pluginName + FlagSuffix
	flg := flag.CommandLine.Lookup(flgName)
	if flg == nil && len(opts) > 0 {
		var flagDefault, flagUsage string

		if len(opts) > 0 && opts[0] != "" {
			flagDefault = opts[0]
		} else {
			flagDefault = pluginName + ".conf"
		}
		if len(opts) > 1 && opts[1] != "" {
			flagUsage = opts[1]
		} else {
			flagUsage = "Location of the " + pluginName +
				" Client configuration file; also set via '" +
				strings.ToUpper(pluginName) + EnvSuffix + "' env variable."
		}
		flag.String(flgName, flagDefault, flagUsage)
	}

	return &pluginConfig{pluginName: pluginName}
}

type pluginConfig struct {
	pluginName string
	access     sync.Mutex
	cfg        string
}

// GetValue binds the configuration to config method argument.
func (p *pluginConfig) GetValue(config interface{}) (found bool, err error) {
	cfgName := p.GetConfigName()
	if cfgName == "" {
		return false, nil
	}
	err = ParseConfigFromYamlFile(cfgName, config) //TODO switch to Viper (possible to have one huge config file)
	if err != nil {
		return false, err
	}

	return true, nil
}

// GetConfigName looks up flag value and uses it to:
// 1. Find config in flag value location.
// 2. Alternatively, it tries to find it in config dir
// (see also Dir() comments).
func (p *pluginConfig) GetConfigName() string {
	p.access.Lock()
	defer p.access.Unlock()
	if p.cfg == "" {
		p.cfg = p.getConfigName()
	}

	return p.cfg
}

func (p *pluginConfig) getConfigName() string {
	flgName := p.pluginName + FlagSuffix
	flg := flag.CommandLine.Lookup(flgName)
	if flg != nil {
		flgVal := flg.Value.String()

		if flgVal != "" {
			// if exist value from flag
			if _, err := os.Stat(flgVal); !os.IsNotExist(err) {
				return flgVal
			}
			cfgDir, err := Dir()
			if err != nil {
				logrus.DefaultLogger().Error(err)
				return ""
			}
			// if exist flag value in config dir
			flgValInConfigDir := path.Join(cfgDir, flgVal)
			if _, err := os.Stat(flgValInConfigDir); !os.IsNotExist(err) {
				return flgValInConfigDir
			}
		}
	}

	return ""
}

// Dir evaluates the flag DirFlag. It interprets "." as current working directory.
func Dir() (string, error) {
	flg := flag.CommandLine.Lookup(DirFlag)
	if flg != nil {
		val := flg.Value.String()
		if strings.HasPrefix(val, ".") {
			cwd, err := os.Getwd()
			if err != nil {
				return cwd, err
			}

			if len(val) > 1 {
				return cwd + val[1:], nil
			}
			return cwd, nil
		}

		return val, nil
	}

	return "", nil
}
