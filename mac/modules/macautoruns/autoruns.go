// +build darwin

package macautoruns

import (
	"bufio"
	"crypto/md5"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/anthonybm/Orion/datawriter"
	"github.com/anthonybm/Orion/instance"
	"github.com/anthonybm/Orion/util"
	"github.com/anthonybm/Orion/util/machelpers"
	"go.uber.org/zap"
)

type MacAutorunsModule struct {
}

var (
	moduleName  = "MacAutorunsModule"
	mode        = "mac"
	version     = "1.0"
	description = `
	Reads and parses various persistent and auto-start programs, daemons, services
	Tries to parse plist configuration files and check code signatures on programs

	- Cron 
	- Kernel Extentions
	- LaunchAgents and LaunchDaemons
	- Login Items
	- Login Restart Apps
	- Periodic Items/ RC Items / emond Items
	- Sandboxed Login Items
	- Startup Items
	- Scripting Additions
	`
	author = "Anthony Martinez, martinez.anthonyb@gmail.com"
)

var (
	header = []string{
		"mtime",
		"atime",
		"ctime",
		"btime",
		"source_name",
		"source_file",
		"program_name",
		"program",
		"arguments",
		"code_signatures",
		"sha256",
		"md5",
		"extras",
	}
	filepathsCron = []string{
		"private/var/at/tabs/*",
	}
	filepathsKernelExtentions = []string{
		"System/Library/Extensions/*/Contents/Info.plist",
		"Library/Extensions/*/Contents/Info.plist",
	}
	filepathsLaunchAgents = []string{
		"System/Library/LaunchAgents/*",
		"Library/LaunchAgents/*",
	}
	filepathsLaunchDaemons = []string{
		"System/Library/LaunchDaemons",
		"Library/LaunchDaemons",
	}
	filepathsLoginItems = []string{
		"Users/*/Library/Preferences/com.apple.loginitems.plist",
		"private/var/*/Library/Preferences/com.apple.loginitems.plist",
	}
	filepathsLoginRestartApps = []string{
		"Users/*/Library/Preferences/ByHost/com.apple.loginwindow.*.plist",
	}
	filepathsSandboxLoginItemsGlob = []string{
		"var/db/com.apple.xpc.launchd/disabled.*.plist",
	}
	filepathScriptingAdditions = []string{
		"System/Library/ScriptingAdditions/*.osax",
		"Library/ScriptingAdditions/*.osax",
	}
	filepathsStartupItems = []string{
		"System/Library/StartupItems",
		"Library/StartupItems",
	}
	filepathsPeriodic = []string{
		"private/etc/periodic/daily",
		"private/etc/periodic/weekly",
		"private/etc/periodic/monthly",
	}
)

func (m MacAutorunsModule) Start(inst instance.Instance) error {
	err := m.autoruns(inst)
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
	}
	return err
}

func (m MacAutorunsModule) autoruns(inst instance.Instance) error {
	mw, err := datawriter.NewOrionWriter(moduleName, inst.GetOrionRuntime(), inst.GetOrionOutputFormat(), inst.GetOrionOutputFilepath())
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
		return err
	}
	values := [][]string{}

	// Start Parsing
	vals, err := m.cron(inst)
	if err != nil {
		if strings.HasSuffix(err.Error(), " were found") {
			zap.L().Warn("Error parsing - "+err.Error(), zap.String("module", moduleName))
		} else {
			zap.L().Error("Error parsing Cron: "+err.Error(), zap.String("module", moduleName))
		}
	}
	values = util.AppendToDoubleSlice(values, vals)

	vals, err = m.kernelExtentions(inst)
	if err != nil {
		if strings.HasSuffix(err.Error(), " were found") {
			zap.L().Warn("Error parsing - "+err.Error(), zap.String("module", moduleName))
		} else {
			zap.L().Error("Error parsing Kernel Extentions: "+err.Error(), zap.String("module", moduleName))
		}
	}
	values = util.AppendToDoubleSlice(values, vals)

	vals, err = m.launchAgentsDaemons(inst)
	if err != nil {
		if strings.HasSuffix(err.Error(), " were found") {
			zap.L().Warn("Error parsing - "+err.Error(), zap.String("module", moduleName))
		} else {
			zap.L().Error("Error parsing Launch Agents and Daemons: "+err.Error(), zap.String("module", moduleName))
		}
	}
	values = util.AppendToDoubleSlice(values, vals)

	vals, err = m.loginItems(inst)
	if err != nil {
		if strings.HasSuffix(err.Error(), " were found") {
			zap.L().Warn("Error parsing - "+err.Error(), zap.String("module", moduleName))
		} else {
			zap.L().Error("Error parsing Login Items: "+err.Error(), zap.String("module", moduleName))
		}
	}
	values = util.AppendToDoubleSlice(values, vals)

	vals, err = m.loginRestartApps(inst)
	if err != nil {
		if strings.HasSuffix(err.Error(), " were found") {
			zap.L().Warn("Error parsing - "+err.Error(), zap.String("module", moduleName))
		} else {
			zap.L().Error("Error parsing Login Restart Apps: "+err.Error(), zap.String("module", moduleName))
		}
	}
	values = util.AppendToDoubleSlice(values, vals)

	vals, err = m.periodicItems(inst)
	if err != nil {
		if strings.HasSuffix(err.Error(), " were found") {
			zap.L().Warn("Error parsing - "+err.Error(), zap.String("module", moduleName))
		} else {
			zap.L().Error("Error parsing Periodic Items: "+err.Error(), zap.String("module", moduleName))
		}
	}
	values = util.AppendToDoubleSlice(values, vals)

	vals, err = m.sandboxedLoginItems(inst)
	if err != nil {
		if strings.HasSuffix(err.Error(), " were found") {
			zap.L().Warn("Error parsing - "+err.Error(), zap.String("module", moduleName))
		} else {
			zap.L().Error("Error parsing Sandboxed Login Items: "+err.Error(), zap.String("module", moduleName))
		}
	}
	values = util.AppendToDoubleSlice(values, vals)

	vals, err = m.startupItems(inst)
	if err != nil {
		if strings.HasSuffix(err.Error(), " were found") {
			zap.L().Warn("Error parsing - "+err.Error(), zap.String("module", moduleName))
		} else {
			zap.L().Error("Error parsing Startup Items: "+err.Error(), zap.String("module", moduleName))
		}
	}
	values = util.AppendToDoubleSlice(values, vals)

	vals, err = m.scriptingAdditions(inst)
	if err != nil {
		if strings.HasSuffix(err.Error(), " were found") {
			zap.L().Warn("Error parsing - "+err.Error(), zap.String("module", moduleName))
		} else {
			zap.L().Error("Error parsing Scripting Additions: "+err.Error(), zap.String("module", moduleName))
		}
	}
	values = util.AppendToDoubleSlice(values, vals)
	// End Parsing

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

func (m MacAutorunsModule) kernelExtentions(inst instance.Instance) ([][]string, error) {
	values := [][]string{}
	count := 0

	kernelExtentionsPaths := util.Multiglob(filepathsKernelExtentions, inst.GetTargetPath())

	for _, path := range kernelExtentionsPaths {
		fi, err := os.Stat(path)
		if err != nil {
			zap.L().Error("Failed to get stats for "+path+": "+err.Error(), zap.String("module", moduleName))
			continue
		}
		if fi.IsDir() {
			continue
		}

		var valmap = make(map[string]string)
		valmap = util.InitializeMapToEmptyString(valmap, header)

		// Add metadata to valmap
		metadata := machelpers.FileTimestamps(path, moduleName)
		valmap["mtime"] = metadata["mtime"]
		valmap["atime"] = metadata["atime"]
		valmap["ctime"] = metadata["ctime"]
		valmap["btime"] = metadata["btime"]

		// Set source info to valmap
		valmap["source_file"] = path
		valmap["source_name"] = "kernel_extentions"

		// Parse plist/bplist
		data, err := machelpers.DecodePlist(path, inst.GetTargetPath())
		if err != nil {
			zap.L().Error("could not parse plist '"+path+"': "+err.Error(), zap.String("module", moduleName))
			continue
		}
		for _, item := range data {
			if val, ok := item["CFBundleName"].(string); ok {
				valmap["program_name"] = strings.TrimSpace(val)
			}
			extra, err := json.Marshal(item)
			if err != nil {
				valmap["extras"] = "<kext>" + strings.TrimSpace(fmt.Sprint(item)) + "</kext>"
			} else {
				valmap["extras"] = strings.TrimSpace(string(extra))
			}

			entry, err := util.UnsafeEntryFromMap(valmap, header)
			if err != nil {
				zap.L().Error("Failed to convert map to entry: "+err.Error(), zap.String("module", moduleName))
				continue
			}
			values = append(values, entry)
			count++
		}
	}

	zap.L().Debug(fmt.Sprintf("Parsed [%d] Kernel Extentions entries", count), zap.String("module", moduleName))
	return values, nil
}

func (m MacAutorunsModule) launchAgentsDaemons(inst instance.Instance) ([][]string, error) {
	values := [][]string{}
	count := 0

	launchPaths := util.Multiglob(filepathsLaunchAgents, inst.GetTargetPath())
	launchPaths = append(launchPaths, util.Multiglob(filepathsLaunchDaemons, inst.GetTargetPath())...)

	for _, path := range launchPaths {
		fi, err := os.Stat(path)
		if err != nil {
			zap.L().Error("Failed to get stats for "+path+": "+err.Error(), zap.String("module", moduleName))
			continue
		}
		if fi.IsDir() {
			continue
		}

		var valmap = make(map[string]string)
		valmap = util.InitializeMapToEmptyString(valmap, header)

		// Add metadata to valmap
		metadata := machelpers.FileTimestamps(path, moduleName)
		valmap["mtime"] = metadata["mtime"]
		valmap["atime"] = metadata["atime"]
		valmap["ctime"] = metadata["ctime"]
		valmap["btime"] = metadata["btime"]

		// Set source info to valmap
		valmap["source_file"] = path
		valmap["source_name"] = "launch_items"

		// Parse plist/bplist
		data, err := machelpers.DecodePlist(path, inst.GetTargetPath())
		if err != nil {
			zap.L().Error("could not parse plist '"+path+"': "+err.Error(), zap.String("module", moduleName))
			continue
		}
		for _, item := range data {
			if val, ok := item["Label"].(string); ok {
				valmap["program_name"] = val
			}

			if val, ok := item["Program"].(string); ok {
				valmap["program"] = val
			}

			if val, ok := item["ProgramArguments"].([]interface{}); ok {
				if len(val) > 1 {
					valmap["program_arguments"] = fmt.Sprint(val[1:])
				}
			}

			entry, err := util.UnsafeEntryFromMap(valmap, header)
			if err != nil {
				zap.L().Error("Failed to convert map to entry: "+err.Error(), zap.String("module", moduleName))
				continue
			}
			values = append(values, entry)
			count++
		}
	}

	zap.L().Debug(fmt.Sprintf("Parsed [%d] Launch Agents and Daemons entries", count), zap.String("module", moduleName))
	return values, nil
}

func (m MacAutorunsModule) loginItems(inst instance.Instance) ([][]string, error) {
	values := [][]string{}

	// Glob LoginRestartApps Items
	loginItemsPlistPaths := util.Multiglob(filepathsLoginItems, inst.GetTargetPath())
	if len(loginItemsPlistPaths) == 0 {
		return [][]string{}, errors.New("no Login Items were found")
	}

	count := 0
	for _, path := range loginItemsPlistPaths {
		var valmap = make(map[string]string)
		valmap = util.InitializeMapToEmptyString(valmap, header)

		// Add metadata to valmap
		metadata := machelpers.FileTimestamps(path, moduleName)
		valmap["mtime"] = metadata["mtime"]
		valmap["atime"] = metadata["atime"]
		valmap["ctime"] = metadata["ctime"]
		valmap["btime"] = metadata["btime"]

		// Set source info to valmap
		valmap["source_file"] = path
		valmap["source_name"] = "login_items"

		// Parse plist/bplist
		data, err := machelpers.DecodePlist(path, inst.GetTargetPath())
		if err != nil {
			zap.L().Error("could not parse plist '"+path+"': "+err.Error(), zap.String("module", moduleName))
			continue
		}

		for _, item := range data {
			machelpers.PrintPlistAsJSON(item)
			zap.L().Warn(fmt.Sprintf("EXIT UNIMPLEMENTED PARSING OF: %s \n, had no test data", item), zap.String("module", moduleName))
			// if val, ok := item["TALAppsToRelaunchAtLogin"].(interface{}); ok {
			// 	for _, i := range val.([]interface{}) {
			// 		valmap["program_name"] = fmt.Sprint(i.(map[string]interface{})["BundleId"])
			// 		valmap["program"] = fmt.Sprint(i.(map[string]interface{})["Path"])

			// 		entry, err := util.UnsafeEntryFromMap(valmap, header)
			// 		if err != nil {
			// 			zap.L().Error("Failed to convert map to entry: "+err.Error(), zap.String("module", moduleName))
			// 			continue
			// 		}
			// 		values = append(values, entry)
			// 		count++
			// 	}
			// }
		}
	}
	zap.L().Debug(fmt.Sprintf("Parsed [%d] Login Item entries", count), zap.String("module", moduleName))

	return values, nil
}

func (m MacAutorunsModule) loginRestartApps(inst instance.Instance) ([][]string, error) {
	values := [][]string{}

	// Glob LoginRestartApps Items
	loginRestartAppsPlistPath := util.Multiglob(filepathsLoginRestartApps, inst.GetTargetPath())
	if len(loginRestartAppsPlistPath) == 0 {
		return [][]string{}, errors.New("no Login Restart Apps were found")
	}

	count := 0
	for _, path := range loginRestartAppsPlistPath {
		var valmap = make(map[string]string)
		valmap = util.InitializeMapToEmptyString(valmap, header)

		// Add metadata to valmap
		metadata := machelpers.FileTimestamps(path, moduleName)
		valmap["mtime"] = metadata["mtime"]
		valmap["atime"] = metadata["atime"]
		valmap["ctime"] = metadata["ctime"]
		valmap["btime"] = metadata["btime"]

		// Set source info to valmap
		valmap["source_file"] = path
		valmap["source_name"] = "login_restart"

		// Parse plist/bplist
		data, err := machelpers.DecodePlist(path, inst.GetTargetPath())
		if err != nil {
			// return [][]string{}, errors.New("failed to decode '" + path + "': " + err.Error())
			zap.L().Error("could not parse plist '"+path+"': "+err.Error(), zap.String("module", moduleName))
			continue
		}

		for _, item := range data {
			if val, ok := item["TALAppsToRelaunchAtLogin"].(interface{}); ok {
				for _, i := range val.([]interface{}) {
					valmap["program_name"] = fmt.Sprint(i.(map[string]interface{})["BundleId"])
					valmap["program"] = fmt.Sprint(i.(map[string]interface{})["Path"])

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
	}
	zap.L().Debug(fmt.Sprintf("Parsed [%d] Login Restart Apps entries", count), zap.String("module", moduleName))

	return values, nil
}

func (m MacAutorunsModule) cron(inst instance.Instance) ([][]string, error) {
	values := [][]string{}

	// Glob Cron
	cronPaths := util.Multiglob(filepathsCron, inst.GetTargetPath())
	if len(cronPaths) == 0 {
		// return [][]string{}, errors.New("no cron items were found")
		zap.L().Debug("No cron items were found", zap.String("module", moduleName))
		return [][]string{}, nil
	}

	count := 0
	for _, path := range cronPaths {
		var valmap = make(map[string]string)
		valmap = util.InitializeMapToEmptyString(valmap, header)

		// Add metadata to valmap
		metadata := machelpers.FileTimestamps(path, moduleName)
		valmap["mtime"] = metadata["mtime"]
		valmap["atime"] = metadata["atime"]
		valmap["ctime"] = metadata["ctime"]
		valmap["btime"] = metadata["btime"]

		// Set source info to valmap
		valmap["source_file"] = path
		valmap["source_name"] = "cron"

		// Parse cron item
		cronFile, err := os.Open(path)
		if err != nil {
			zap.L().Debug("Could not open '"+path+"': "+err.Error(), zap.String("module", moduleName))
			continue
		}
		defer cronFile.Close()

		scanner := bufio.NewScanner(cronFile)
		for scanner.Scan() {
			text := strings.TrimSpace(scanner.Text())
			if !strings.HasPrefix(text, "# ") {
				valmap["program"] = text
				entry, err := util.UnsafeEntryFromMap(valmap, header)
				if err != nil {
					zap.L().Error("Failed to convert map to entry: "+err.Error(), zap.String("module", moduleName))
					continue
				}
				values = append(values, entry)
			}
		}
	}
	zap.L().Debug(fmt.Sprintf("Parsed [%d] Sandboxed Login Items entries", count), zap.String("module", moduleName))
	return values, nil
}

func (m MacAutorunsModule) periodicItems(inst instance.Instance) ([][]string, error) {
	values := [][]string{}

	// Glob Periodic Items
	periodicItemsPaths := util.Multiglob(filepathsPeriodic, inst.GetTargetPath())
	if len(periodicItemsPaths) == 0 {
		return [][]string{}, errors.New("no periodic items were found")
	}

	count := 0
	for _, path := range periodicItemsPaths {
		var valmap = make(map[string]string)
		valmap = util.InitializeMapToEmptyString(valmap, header)

		// Add metadata to valmap
		metadata := machelpers.FileTimestamps(path, moduleName)
		valmap["mtime"] = metadata["mtime"]
		valmap["atime"] = metadata["atime"]
		valmap["ctime"] = metadata["ctime"]
		valmap["btime"] = metadata["btime"]

		// Set source info to valmap
		valmap["source_file"] = path
		valmap["source_name"] = "periodic_items"

		entry, err := util.UnsafeEntryFromMap(valmap, header)
		if err != nil {
			zap.L().Error("Failed to convert map to entry: "+err.Error(), zap.String("module", moduleName))
			continue
		}
		values = append(values, entry)
		count++
	}
	zap.L().Debug(fmt.Sprintf("Parsed [%d] Sandboxed Login Items entries", count), zap.String("module", moduleName))

	return values, nil
}

func (m MacAutorunsModule) sandboxedLoginItems(inst instance.Instance) ([][]string, error) {
	values := [][]string{}

	// Glob Sandboxed Login Items
	sandboxLoginItemsPaths := util.Multiglob(filepathsSandboxLoginItemsGlob, inst.GetTargetPath())
	if len(sandboxLoginItemsPaths) == 0 {
		return [][]string{}, errors.New("no sandbox login items were found")
	}

	sandboxedLoginItemsCount := 0
	for _, path := range sandboxLoginItemsPaths {
		var valmap = make(map[string]string)
		valmap = util.InitializeMapToEmptyString(valmap, header)

		// Add metadata to valmap
		metadata := machelpers.FileTimestamps(path, moduleName)
		valmap["mtime"] = metadata["mtime"]
		valmap["atime"] = metadata["atime"]
		valmap["ctime"] = metadata["ctime"]
		valmap["btime"] = metadata["btime"]

		// Set source info to valmap
		valmap["source_file"] = path
		valmap["source_name"] = "sandboxed_login_items"

		// Parse plist/bplist
		data, err := machelpers.DecodePlist(path, inst.GetTargetPath())
		if err != nil {
			return [][]string{}, errors.New("failed to decode '" + path + "': " + err.Error())
		}

		// Read data from plist/bplist
		// Format of this requires that we loop over key,vals and write entries for each False item

		for _, i := range data {
			for k, v := range i {
				if val, ok := v.(bool); ok {
					if val == false {
						valmap["program_name"] = k
						entry, err := util.UnsafeEntryFromMap(valmap, header)
						if err != nil {
							zap.L().Error("Failed to convert map to entry: "+err.Error(), zap.String("module", moduleName))
							continue
						}
						values = append(values, entry)
						sandboxedLoginItemsCount++
					}
				}
			}
		}
	}
	zap.L().Debug(fmt.Sprintf("Parsed [%d] Sandboxed Login Items entries", sandboxedLoginItemsCount), zap.String("module", moduleName))

	return values, nil
}

func (m MacAutorunsModule) scriptingAdditions(inst instance.Instance) ([][]string, error) {
	values := [][]string{}
	count := 0

	scriptingAdditionsPaths := util.Multiglob(filepathScriptingAdditions, inst.GetTargetPath())
	if len(scriptingAdditionsPaths) == 0 {
		// return [][]string{}, errors.New("no cron items were found")
		zap.L().Debug("No Scripting Additions were found", zap.String("module", moduleName))
		return [][]string{}, nil
	}

	for _, path := range scriptingAdditionsPaths {
		fi, err := os.Stat(path)
		if err != nil {
			zap.L().Error("Failed to get stats for "+path+": "+err.Error(), zap.String("module", moduleName))
			continue
		}
		if fi.IsDir() {
			continue
		}

		var valmap = make(map[string]string)
		valmap = util.InitializeMapToEmptyString(valmap, header)

		// Add metadata to valmap
		metadata := machelpers.FileTimestamps(path, moduleName)
		valmap["mtime"] = metadata["mtime"]
		valmap["atime"] = metadata["atime"]
		valmap["ctime"] = metadata["ctime"]
		valmap["btime"] = metadata["btime"]

		// Set source info to valmap
		valmap["source_file"] = path
		valmap["source_name"] = "scripting_additions"
		valmap["code_signatures"] = fmt.Sprint(machelpers.GetCodesignatures(path))

		entry, err := util.UnsafeEntryFromMap(valmap, header)
		if err != nil {
			zap.L().Error("Failed to convert map to entry: "+err.Error(), zap.String("module", moduleName))
			continue
		}
		values = append(values, entry)
		count++

	}

	zap.L().Debug(fmt.Sprintf("Parsed [%d] Scripting Additions entries", count), zap.String("module", moduleName))

	return values, nil
}

func (m MacAutorunsModule) startupItems(inst instance.Instance) ([][]string, error) {
	values := [][]string{}
	count := 0

	startupItemsPaths := util.Multiglob(filepathScriptingAdditions, inst.GetTargetPath())
	if len(startupItemsPaths) == 0 {
		// return [][]string{}, errors.New("no cron items were found")
		zap.L().Debug("No Startup Items were found", zap.String("module", moduleName))
		return [][]string{}, nil
	}

	for _, path := range startupItemsPaths {
		fi, err := os.Stat(path)
		if err != nil {
			zap.L().Error("Failed to get stats for "+path+": "+err.Error(), zap.String("module", moduleName))
			continue
		}
		if fi.IsDir() {
			continue
		}

		var valmap = make(map[string]string)
		valmap = util.InitializeMapToEmptyString(valmap, header)

		// Add metadata to valmap
		metadata := machelpers.FileTimestamps(path, moduleName)
		valmap["mtime"] = metadata["mtime"]
		valmap["atime"] = metadata["atime"]
		valmap["ctime"] = metadata["ctime"]
		valmap["btime"] = metadata["btime"]

		// Set source info to valmap
		valmap["source_file"] = path
		valmap["source_name"] = "startup_items"

		entry, err := util.UnsafeEntryFromMap(valmap, header)
		if err != nil {
			zap.L().Error("Failed to convert map to entry: "+err.Error(), zap.String("module", moduleName))
			continue
		}
		values = append(values, entry)
		count++

	}

	zap.L().Debug(fmt.Sprintf("Parsed [%d] Startup Items entries", count), zap.String("module", moduleName))

	return values, nil
}

func fileSHA256(fp string) (string, error) {
	f, err := os.Open(fp)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func fileMD5(fp string) (string, error) {
	f, err := os.Open(fp)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
