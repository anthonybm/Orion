// +build darwin

package macspotlight

import (
	"errors"
	"fmt"

	"github.com/anthonybm/Orion/datawriter"
	"github.com/anthonybm/Orion/instance"
	"github.com/anthonybm/Orion/util"
	"github.com/anthonybm/Orion/util/machelpers"
	"go.uber.org/zap"
)

type MacSpotlightShortcutsModule struct {
}

var (
	moduleName  = "MacSpotlightShortcutsModule"
	mode        = "mac"
	version     = "1.0"
	description = `
	Parses the com.apple.spotlight.Shortcuts.plist file
	Contains a record of every application opened with Spotlight 
	and associated timestamp of when it was last opened.
	`
	author = "Anthony Martinez, martinez.anthonyb@gmail.com"
)

var (
	header = []string{
		"user",
		"shortcut",
		"display_name",
		"last_used",
		"url",
	}
	filepathsSpotlightShortcutsPlists = []string{
		"Users/*/Library/Application Support/com.apple.spotlight/com.apple.spotlight.Shortcuts",
		"private/var/*/Library/Application Support/com.apple.spotlight/com.apple.spotlight.Shortcuts",
	}
)

func (m MacSpotlightShortcutsModule) Start(inst instance.Instance) error {

	err := m.shortcuts(inst)
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
	}
	return err
}

func (m MacSpotlightShortcutsModule) shortcuts(inst instance.Instance) error {
	values := [][]string{}
	mw, err := datawriter.NewOrionWriter(moduleName, inst.GetOrionRuntime(), inst.GetOrionOutputFormat(), inst.GetOrionOutputFilepath())
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
		return err
	}

	spotlightShortcutPlistPaths := util.Multiglob(filepathsSpotlightShortcutsPlists, inst.GetTargetPath())
	if len(spotlightShortcutPlistPaths) == 0 {
		return errors.New("no spotlight shortcuts plists were found")
	}

	count := 0
	for _, path := range spotlightShortcutPlistPaths {
		var valmap = make(map[string]string)
		valmap = util.InitializeMapToEmptyString(valmap, header)

		zap.L().Debug(fmt.Sprintf("Parsing Spotlight Shortcuts plist for %s", util.GetUsernameFromPath(path)), zap.String("module", moduleName))
		data, err := machelpers.DecodePlist(path, inst.GetTargetPath())
		if err != nil {
			zap.L().Error("could not parse plist '"+path+"': "+err.Error(), zap.String("module", moduleName))
			continue
		}

		for _, item := range data {
			for k, v := range item {
				valmap["display_name"], _ = util.InterfaceToString(v.(map[string]interface{})["DISPLAY_NAME"])
				valmap["last_used"], _ = util.InterfaceToString(v.(map[string]interface{})["LAST_USED"])
				valmap["url"], _ = util.InterfaceToString(v.(map[string]interface{})["URL"])
				valmap["user"] = util.GetUsernameFromPath(path)
				val, err := util.InterfaceToString(k)
				if err != nil {
					val = "ERROR"
				}
				valmap["shortcut"] = val

				entry, err := util.UnsafeEntryFromMap(valmap, header)
				if err != nil {
					zap.L().Error("Failed to convert map to entry: "+err.Error(), zap.String("module", moduleName))
					continue
				}
				values = append(values, entry)
				count++

			}
		}
	}
	zap.L().Debug(fmt.Sprintf("Parsed [%d] Spotlight Shortcuts entries", count), zap.String("module", moduleName))

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
