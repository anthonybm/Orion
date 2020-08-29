package instance

import (
	"errors"
	"os"

	"github.com/tonythetiger06/Orion/configs"
	"github.com/tonythetiger06/Orion/util"
	"go.uber.org/zap"
)

/* Orion Globals */

// TargetPath specifies the root target path to reference artifacts from - i.e. <target>/pathToPlist.plist
var TargetPath string

type InstanceInterface interface {
	// GetOrionWriter() datawriter.OrionWriter
	GetOrionConfig() configs.Config
	GetOrionRuntime() string
	GetOrionModules() []string
	Logger() *zap.Logger
	NoMultithreading() bool
}

type Instance struct {
	// orionwriter  datawriter.OrionWriter
	orionconfig      configs.Config
	orionlogger      *zap.Logger
	orionlogfile     *os.File
	orionruntime     string
	noMultithreading bool
	outputformat     string
	outputpath       string
	targetpath       string
	forensicMode     bool
	mode             string
}

// NewInstance returns a new instance struct based on arguments, should only be called once per run
func NewInstance(targetpath string, outputformat string, outputPath string, orionRuntime string, loglevel string, configpath string, mode string, noMultithreading bool, forensicMode bool) (Instance, error) {
	// Instantiate logger and handle any errors
	logger, logfile, err := util.NewOrionLogger(loglevel, orionRuntime, outputPath)
	if err != nil {
		return Instance{}, err
	}
	logger.Named(mode)

	// Parse the config file
	config, err := configs.Parse(configpath, mode)
	if err != nil {
		logger.Error("Failed to parse config file: ", zap.String("error", err.Error()))
		return Instance{}, errors.New("failed to parse config file")
	}

	inst := Instance{
		// orionwriter: orionWriter,
		orionconfig:      config,
		orionlogger:      logger,
		orionlogfile:     logfile,
		orionruntime:     orionRuntime,
		noMultithreading: noMultithreading,
		outputformat:     outputformat,
		outputpath:       outputPath,
		targetpath:       targetpath,
		forensicMode:     forensicMode,
		mode:             mode,
	}

	return inst, nil
}

func (i Instance) CloseLogger() error {
	return i.orionlogfile.Close()
}

// GetOrionRuntime returns the name of the Orion runtime
func (i Instance) GetOrionRuntime() string {
	return i.orionruntime
}

func (i Instance) GetOrionMode() string {
	return i.mode
}

// GetOrionOutputFormat returns the name of the output file type (csv, xlsx, etc.)
func (i Instance) GetOrionOutputFormat() string {
	return i.outputformat
}

// GetOrionOutputFilepath returns the string filepath where output is written, relative to Orion
func (i Instance) GetOrionOutputFilepath() string {
	return i.outputpath
}

// GetOrionModules returns string slice of modules to run from config file
func (i Instance) GetOrionModules() ([]string, error) {
	conf := i.GetOrionConfig()
	mods, err := conf.GetModulesToExecute()
	return mods, err
}

// GetOrionConfig returns the Config for this Instance of Orion
func (i Instance) GetOrionConfig() configs.Config {
	return i.orionconfig
}

// NoMultithreading exposes argument flag for running Orion with goroutines for each module
func (i Instance) NoMultithreading() bool {
	return i.noMultithreading
}

// ForensicMode exposes argument flag for running Orion with forensic mode active for each module
func (i Instance) ForensicMode() bool {
	return i.forensicMode
}

// GetTargetPath returns the string representing the path to the target for modules
func (i Instance) GetTargetPath() string {
	return i.targetpath
}
