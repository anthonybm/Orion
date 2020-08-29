// +build darwin

package macsample

import (
	"os"
	"path/filepath"

	"github.com/tonythetiger06/Orion/datawriter"
	"github.com/tonythetiger06/Orion/instance"

	"go.uber.org/zap"
	"howett.net/plist"
)

type MacSampleModule struct {
}

type systemVersionHeader struct {
	ProductBuildVersion       string `plist:"ProductBuildVersion"`
	ProductCopyright          string `plist:"ProductCopyright"`
	ProductName               string `plist:"ProductName"`
	ProductUserVisibleVersion string `plist:"ProductUserVisibleVersion"`
	ProductVersion            string `plist:"ProductVersion"`
}

var (
	moduleName  = "MacSampleModule"
	mode        = "mac"
	version     = "1.0"
	description = ""
	author      = ""
)

var (
	filepathSystemVersionPlist = "System/Library/CoreServices/SystemVersion.plist"
	keyProductVersion          = "ProductVersion"
)

func (m MacSampleModule) Start(inst instance.Instance) error {
	err := m.osVersion(inst)
	if err != nil {
		zap.L().Error("Error running MacSampleModule: " + err.Error())
	}
	return err
}

func (m MacSampleModule) osVersion(inst instance.Instance) error {
	zap.L().Debug("Grabbing OS version from " + filepathSystemVersionPlist)

	dw, err := datawriter.NewOrionWriter(moduleName, inst.GetOrionRuntime(), inst.GetOrionOutputFormat(), inst.GetOrionOutputFilepath())
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
		return err
	}

	// Read SystemVersion.plist
	f, err := os.Open(filepath.Join(inst.GetTargetPath(), filepathSystemVersionPlist))
	if err != nil {
		return err
	}
	p := plist.NewDecoder(f)

	// Grab val from key 'ProductVersion'
	var data systemVersionHeader
	err = p.Decode(&data)
	if err != nil {
		return err
	}

	zap.L().Debug("Got OS version " + data.ProductVersion)

	header := []string{"OS Version"}
	values := []string{data.ProductVersion}
	err = dw.WriteHeader(header)
	if err != nil {
		return err
	}
	err = dw.Write(values)
	if err != nil {
		return err
	}
	err = dw.Close()
	if err != nil {
		return err
	}

	return nil
}
