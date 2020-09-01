// +build darwin

package macinstallhistory

import (
	"fmt"
	"time"

	"github.com/anthonybm/Orion/datawriter"
	"github.com/anthonybm/Orion/instance"
	"github.com/anthonybm/Orion/util/machelpers"
	"go.uber.org/zap"
)

type MacInstallHistoryModule struct {
}

var (
	moduleName  = "MacInstallHistoryModule"
	mode        = "mac"
	version     = "1.0"
	description = "Parses the InstallHistory.plist file"
	author      = "Anthony Martinez, martinez.anthonyb@gmail.com"
)

var (
	filepathInstallHistoryPlist = "Library/Receipts/InstallHistory.plist" // TODO fix path to be configurable
)

type installHistoryItem struct {
	Date               time.Time `plist:"date"`
	ContentType        string    `plist:"contentType"`
	DisplayName        string    `plist:"displayName"`
	DisplayVersion     string    `plist:"displayVersion"`
	PackageIdentifiers []string  `plist:"packageIdentifiers"`
	ProcessName        string    `plist:"processName"`
}

func (m MacInstallHistoryModule) Start(inst instance.Instance) error {
	err := m.installHistory(inst)
	if err != nil {
		zap.L().Error("Error running "+moduleName+": "+err.Error(), zap.String("module", moduleName))
	}
	return err
}

func (m MacInstallHistoryModule) installHistory(inst instance.Instance) error {
	zap.L().Debug("Parsing InstallHistory.plist file from "+filepathInstallHistoryPlist, zap.String("module", moduleName))
	mw, err := datawriter.NewOrionWriter(moduleName, inst.GetOrionRuntime(), inst.GetOrionOutputFormat(), inst.GetOrionOutputFilepath())
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
		return err
	}

	header := []string{
		"timestamp",
		"content_type",
		"display_name",
		"display_version",
		"package_identifiers",
		"process_name",
	}
	values := [][]string{}

	// Read and parse InstallHistory.plist
	data, err := machelpers.DecodePlist(filepathInstallHistoryPlist, inst.GetTargetPath())
	if err != nil {
		return err
	}

	for _, item := range data {
		var result installHistoryItem
		if val, ok := item["date"]; ok {
			// fmt.Println(reflect.TypeOf(val).String())
			result.Date = val.(time.Time)
		}
		if val, ok := item["contentType"]; ok {
			// fmt.Println(reflect.TypeOf(val).String())
			result.ContentType = val.(string)
		}
		if val, ok := item["displayName"]; ok {
			// fmt.Println(reflect.TypeOf(val).String())
			result.DisplayName = val.(string)
		}
		if val, ok := item["displayVersion"]; ok {
			// fmt.Println(reflect.TypeOf(val).String())
			result.DisplayName = val.(string)
		}
		if val, ok := item["packageIdentifiers"]; ok {
			// fmt.Println(reflect.TypeOf(val).String())
			concreteVal := make([]string, len(val.([]interface{})))
			for i, v := range val.([]interface{}) {
				concreteVal[i] = fmt.Sprint(v)
			}
			result.PackageIdentifiers = concreteVal
		}
		if val, ok := item["processName"]; ok {
			// fmt.Println(reflect.TypeOf(val).String())
			result.ProcessName = val.(string)
		}

		entry := []string{
			result.Date.UTC().Format(time.RFC3339),
			result.ContentType,
			result.DisplayName,
			result.DisplayVersion,
			result.PackageIdentifiers[0],
			result.ProcessName,
		}
		values = append(values, entry)
	}

	// Write to output
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
