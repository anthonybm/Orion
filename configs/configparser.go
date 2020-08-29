package configs

import (
	"errors"
	"fmt"
	"math"

	"github.com/BurntSushi/toml"
	"go.uber.org/zap"
)

type configInterface interface {
	GetConfigType() string
	GetModules() []string
}

type Config struct {
	configpath        string
	macconfig         MacConfig
	macconfigbool     bool
	windowsconfig     WindowsConfig
	windowsconfigbool bool
}

type MacConfig struct {
	ForensicMode              bool
	Verbose                   bool
	Modules                   []string // modules to execute
	DirlistExcludedDirs       []string // Folders to exclude
	DirlistExcludedExts       []string // Extentions to exclude
	DirlistRootWalkDir        string
	DirlistHashSizeLimitBytes int
	DirlistDoHashMD5          bool
	DirlistDoHashSHA256       bool
}

type WindowsConfig struct {
	ForensicMode              bool
	Verbose                   bool
	Modules                   []string // modules to execute
	DirlistExcludedDrives     []string // Windows drives to exclude
	DirlistExcludedDirs       []string // Folders to exclude
	DirlistExcludedExts       []string // Extentions to exclude
	DirlistRootWalkDir        string
	DirlistHashSizeLimitBytes int
	DirlistDoHashMD5          bool
	DirlistDoHashSHA256       bool
}

// configTypeError defines an error occuring with Orion not ready to parse that config type.
type configTypeError struct {
	arg  string
	prob string
}

func (conf Config) GetModulesToExecute() ([]string, error) {
	switch conf.GetConfigType() {
	case "mac":
		return conf.macconfig.Modules, nil
	case "windows":
		return conf.windowsconfig.Modules, nil
	}
	return []string{}, errors.New("cannot get modules for config of type " + conf.GetConfigType())
}

func (conf Config) GetDirlistExcludedDrives() ([]string, error) {
	switch conf.GetConfigType() {
	case "windows":
		return conf.windowsconfig.DirlistExcludedDrives, nil
	}
	return []string{}, errors.New("could not read dirlist excluded drives key for config of type " + conf.GetConfigType())
}

func (conf Config) GetDirlistExcludedDirs() ([]string, error) {
	switch conf.GetConfigType() {
	case "mac":
		return conf.macconfig.DirlistExcludedDirs, nil
	case "windows":
		return conf.windowsconfig.DirlistExcludedDirs, nil
	}
	return []string{}, errors.New("could not read dirlist excluded dirs key for config of type " + conf.GetConfigType())
}

func (conf Config) GetDirlistExcludedExts() ([]string, error) {
	switch conf.GetConfigType() {
	case "mac":
		return conf.macconfig.DirlistExcludedExts, nil
	case "windows":
		return conf.windowsconfig.DirlistExcludedExts, nil
	}
	return []string{}, errors.New("could not read dirlist excluded exts key for config of type " + conf.GetConfigType())
}

func (conf Config) GetDirlistRootWalkDir() (string, error) {
	switch conf.GetConfigType() {
	case "mac":
		return conf.macconfig.DirlistRootWalkDir, nil
	case "windows":
		return conf.windowsconfig.DirlistRootWalkDir, nil
	}
	return "", errors.New("could not read dirlist root walk dir key for config of type " + conf.GetConfigType())
}

func (conf Config) GetDirlistHashSizeLimitBytes() (int, error) {
	switch conf.GetConfigType() {
	case "mac":
		return conf.macconfig.DirlistHashSizeLimitBytes, nil
	case "windows":
		return conf.windowsconfig.DirlistHashSizeLimitBytes, nil
	}
	return math.MaxInt64, errors.New("could not read dirlist hash size limit key for config of type " + conf.GetConfigType())
}

func (conf Config) GetDirlistDoHashMD5() (bool, error) {
	switch conf.GetConfigType() {
	case "mac":
		return conf.macconfig.DirlistDoHashMD5, nil
	case "windows":
		return conf.windowsconfig.DirlistDoHashMD5, nil
	}
	return true, errors.New("could not read dirlist md5hash key for config of type " + conf.GetConfigType())
}

func (conf Config) GetDirlistDohashSHA256() (bool, error) {
	switch conf.GetConfigType() {
	case "mac":
		return conf.macconfig.DirlistDoHashSHA256, nil
	case "windows":
		return conf.windowsconfig.DirlistDoHashSHA256, nil
	}
	return true, errors.New("could not read dirlist sha256hash key for config of type " + conf.GetConfigType())
}

func (conf Config) IsForensicMode() (bool, error) {
	switch conf.GetConfigType() {
	case "mac":
		return conf.macconfig.ForensicMode, nil
	case "windows":
		return conf.windowsconfig.ForensicMode, nil
	}
	return true, errors.New("could not read forensicMode key for config of type " + conf.GetConfigType())
}

func (conf Config) IsVerbose() (bool, error) {
	switch conf.GetConfigType() {
	case "mac":
		return conf.macconfig.ForensicMode, nil
	case "windows":
		return conf.windowsconfig.ForensicMode, nil
	}
	return true, errors.New("could not read verbose key for config of type " + conf.GetConfigType())
}

func (e *configTypeError) Error() string {
	return fmt.Sprintf("%d - %s", e.arg, e.prob)
}

func (conf Config) GetConfigType() string {
	if conf.macconfigbool {
		return "mac"
	} else if conf.windowsconfigbool {
		return "windows"
	}
	return ""
}

// initConfig takes in the device parsing type and returns a configured Config type
func initConfig(configpath string, mode string) (Config, error) {
	if configpath == "" {
		return Config{}, errors.New("configparser: empty filepath for config provided")
	}

	var conf Config
	switch mode {
	case "mac":
		conf.macconfigbool = true
		conf.macconfig = MacConfig{}
		conf.configpath = configpath
		return conf, nil
	case "windows":
		conf.windowsconfigbool = true
		conf.windowsconfig = WindowsConfig{}
		conf.configpath = configpath
		return conf, nil
	}
	err := &configTypeError{mode, "- Orion cannot use this config type."}
	return Config{}, err
}

// Parse takes in the relative path to a config file and the device parsing type
func Parse(configpath string, mode string) (Config, error) {
	conf, err := initConfig(configpath, mode)
	if conf.configpath == "" {
		return Config{}, errors.New("configparser: struct has empty filepath for config")
	}
	if err != nil {
		zap.L().Error("configparser: Orion cannot parse this type of config: ", zap.String("error", err.Error()))
		return conf, err
	}

	switch conf.GetConfigType() {
	case "mac":
		conf, err := parseMacConfig(conf)
		if err != nil {
			zap.L().Error("configparser: syntax error in given mac config file: ", zap.String("error", err.Error()))
			return conf, err
		}
		return conf, nil
	case "windows":
		conf, err := parseWindowsConfig(conf)
		if err != nil {
			zap.L().Error("configparser: syntax error in given windows config file: ", zap.String("error", err.Error()))
			return conf, err
		}
		return conf, nil
	}
	err = errors.New("configparser: syntax error in given config file")
	return conf, err
}

// parseMacConfig takes in an initialized Config type and returns it configured for mac
func parseMacConfig(conf Config) (Config, error) {
	var tomlConf MacConfig
	if _, err := toml.DecodeFile(conf.configpath, &tomlConf); err != nil {
		msg := "configparser: cannot parse mac config toml file: '" + conf.configpath + "'"
		zap.L().Error(msg, zap.String("error", err.Error()))
		return conf, err
	}

	// parse fields from TOML file
	conf.macconfig.ForensicMode = tomlConf.ForensicMode
	conf.macconfig.Verbose = tomlConf.Verbose
	conf.macconfig.Modules = tomlConf.Modules
	conf.macconfig.DirlistExcludedDirs = tomlConf.DirlistExcludedDirs
	conf.macconfig.DirlistExcludedExts = tomlConf.DirlistExcludedExts
	conf.macconfig.DirlistDoHashMD5 = tomlConf.DirlistDoHashMD5
	conf.macconfig.DirlistDoHashSHA256 = tomlConf.DirlistDoHashSHA256
	conf.macconfig.DirlistHashSizeLimitBytes = tomlConf.DirlistHashSizeLimitBytes

	return conf, nil
}

// parseWindowsConfig takes in an initialized Config type and returns it configured for Windows
func parseWindowsConfig(conf Config) (Config, error) {
	var tomlConf WindowsConfig
	if _, err := toml.DecodeFile(conf.configpath, &tomlConf); err != nil {
		msg := "configparser: cannot parse Windows config toml file: '" + conf.configpath + "'"
		zap.L().Error(msg, zap.String("error", err.Error()))
		return conf, err
	}

	// parse fields from TOML file
	conf.windowsconfig.ForensicMode = tomlConf.ForensicMode
	conf.windowsconfig.Verbose = tomlConf.Verbose
	conf.windowsconfig.Modules = tomlConf.Modules
	conf.windowsconfig.DirlistExcludedDirs = tomlConf.DirlistExcludedDirs
	conf.windowsconfig.DirlistExcludedExts = tomlConf.DirlistExcludedExts
	conf.windowsconfig.DirlistDoHashMD5 = tomlConf.DirlistDoHashMD5
	conf.windowsconfig.DirlistDoHashSHA256 = tomlConf.DirlistDoHashSHA256
	conf.windowsconfig.DirlistHashSizeLimitBytes = tomlConf.DirlistHashSizeLimitBytes
	conf.windowsconfig.DirlistExcludedDrives = tomlConf.DirlistExcludedDrives

	return conf, nil
}
