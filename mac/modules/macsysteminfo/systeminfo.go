// +build darwin

package macsysteminfo

import (
	"errors"

	"github.com/anthonybm/Orion/datawriter"
	"github.com/anthonybm/Orion/instance"
	"github.com/anthonybm/Orion/util"
	"github.com/anthonybm/Orion/util/machelpers"
	"go.uber.org/zap"
)

var (
	moduleName  = "MacSystemInfoModule"
	mode        = "mac"
	version     = "1.0"
	description = `
	Grabs and writes the following:
	- OS Version
	- Build version
	- Serial Number
	- Volume created
	- Model and Computer name
	- Hostname and local hostname
	- Timezone
	- Last logged in user
	- Volume information
	- IP Address
	- Gatekeeper status
	- FVDE Status
	- SIP status
	`
	author = "Anthony Martinez, martinez.anthonyb@gmail.com"
)

var (
	filepathGlobalPreferencesPlist              = "Library/Preferences/.GlobalPreferences.plist"
	filepathSystemConfigurationPreferencesPlist = "Library/Preferences/SystemConfiguration/preferences.plist"
	filepathSystemVersionPlist                  = "System/Library/CoreServices/SystemVersion.plist"
)

type systemVersionPlist struct {
	ProductBuildVersion       string `plist:"ProductBuildVersion"`
	ProductCopyright          string `plist:"ProductCopyright"`
	ProductName               string `plist:"ProductName"`
	ProductUserVisibleVersion string `plist:"ProductUserVisibleVersion"`
	ProductVersion            string `plist:"ProductVersion"`
}

type MacSystemInfoModule struct {
}

func (m MacSystemInfoModule) Start(inst instance.Instance) error {
	err := m.systeminfo(inst)
	if err != nil {
		zap.L().Error("Error running "+moduleName+": "+err.Error(), zap.String("module", moduleName))
	}
	return err
}

func (m MacSystemInfoModule) systeminfo(inst instance.Instance) error {
	mw, err := datawriter.NewOrionWriter(moduleName, inst.GetOrionRuntime(), inst.GetOrionOutputFormat(), inst.GetOrionOutputFilepath())
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
		return err
	}

	header := []string{
		"local_hostname",
		"computer_name",
		"hostname",
		"model",
		"os_version",
		"os_build_version",
		// "serial_number",
		// "volume_created",
		// "system_timezone",
		// "ipaddress",
		// "fvde_status",
		// "gatekeeper_status",
		// "sip_status",
	}
	headermap := make(map[string]string)

	// // Read and parse .GlobalPreferences.plist
	// data, err := machelpers.DecodePlist(filepathGlobalPreferencesPlist, inst.GetTargetPath())
	// if err != nil {
	// 	return errors.New("failed to decode '" + filepathGlobalPreferencesPlist + "': " + err.Error())
	// }
	// headermap["system_timezone"]

	// Read and parse preferences.plist
	data, err := machelpers.DecodePlist(filepathSystemConfigurationPreferencesPlist, inst.GetTargetPath())

	headermap["local_hostname"], err = machelpers.GetSingleValueFromPlist(data, "LocalHostName")
	if err != nil {
		headermap["local_hostname"] = "ERROR"
		zap.L().Error("Error grabbing local hostname: "+err.Error(), zap.String("module", moduleName))
	}
	headermap["hostname"], err = machelpers.GetSingleValueFromPlist(data, "HostName")
	if err != nil {
		headermap["hostname"] = "ERROR"
		zap.L().Error("Error grabbing hostname: "+err.Error(), zap.String("module", moduleName))
	}
	headermap["model"], err = machelpers.GetSingleValueFromPlist(data, "Model")
	if err != nil {
		headermap["model"] = "ERROR"
		zap.L().Error("Error grabbing model: "+err.Error(), zap.String("module", moduleName))
	}
	headermap["computer_name"], err = machelpers.GetSingleValueFromPlist(data, "ComputerName")
	if err != nil {
		headermap["computer_name"] = "ERROR"
		zap.L().Error("Error grabbing computer name: "+err.Error(), zap.String("module", moduleName))
	}

	// Read and parse SystemVersion.plist
	data, err = machelpers.DecodePlist(filepathSystemVersionPlist, inst.GetTargetPath())
	if err != nil {
		return errors.New("failed to decode '" + filepathSystemVersionPlist + "': " + err.Error())
	}
	headermap["os_version"], err = machelpers.GetSingleValueFromPlist(data, "ProductVersion")
	if err != nil {
		headermap["os_version"] = "ERROR"
	}
	headermap["os_build_version"], err = machelpers.GetSingleValueFromPlist(data, "ProductBuildVersion")
	if err != nil {
		headermap["os_build_version"] = "ERROR"
	}

	// Write to output
	entry, err := util.EntryFromMap(headermap, header)
	if err != nil {
		return err
	}

	values := [][]string{}
	values = append(values, entry)

	err = mw.WriteHeader(header)
	if err != nil {
		return err
	}
	err = mw.WriteAll(values)
	if err != nil {
		return err
	}
	err = mw.Close()
	if err != nil {
		return err
	}
	return nil
}
