// +build darwin

package macquarantines

import (
	"errors"
	"os"
	"strconv"

	"github.com/anthonybm/Orion/datawriter"
	"github.com/anthonybm/Orion/instance"
	"github.com/anthonybm/Orion/util"
	"go.uber.org/zap"
	"howett.net/plist"
)

var (
	moduleName  = "MacQuarantinesModule"
	mode        = "mac"
	version     = "1.0"
	description = `
	Parses the QuarantineEventsV2 databases and GateKeeper .LastGKReject file (not seen in 10.13?)
	`
	author = "Anthony Martinez, martinez.anthonyb@gmail.com"
)

type MacQuarantinesModule struct {
}

var (
	quarantineEventsV2Filepaths = []string{
		"Users/*/Library/Preferences/com.apple.LaunchServices.QuarantineEventsV2",
		"private/var/*/Library/Preferences/com.apple.LaunchServices.QuarantineEventsV2",
	}
	gatekeeperFilepaths = []string{
		"private/var/db/.LastGKReject",
	}
)

type lastGKRejectPlist struct {
}

func (m MacQuarantinesModule) Start(inst instance.Instance) error {
	err := m.quarantines(inst)
	if err != nil {
		zap.L().Error("Error running "+moduleName+": "+err.Error(), zap.String("module", moduleName))
	}
	return err
}

func (m MacQuarantinesModule) quarantines(inst instance.Instance) error {
	quarantineheader := []string{
		"user",
		"EventIdentifier",       //TEXT
		"TimeStamp",             // REAL
		"AgentBundleIdentifier", //TEXT
		"AgentName",             //TEXT
		"DataURLString",         //TEXT
		"SenderName",            //TEXT
		"SenderAddress",         //TEXT
		"TypeNumber",            //INTEGER
		"OriginTitle",           //TEXT
		"OriginURLString",       //TEXT
		"OriginAlias",           //BLOB
	}

	mw, err := datawriter.NewOrionWriter(moduleName, inst.GetOrionRuntime(), inst.GetOrionOutputFormat(), inst.GetOrionOutputFilepath())
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
		return err
	}

	// gatekeeperheader := []string{
	// 	"BookmarkData",
	// 	"TimeStamp",
	// 	"XProtectMalwareType",
	// }

	quarantineValues := [][]string{}
	// gatekeeperValues := [][]string{}

	// get all QuarantineEventsV2 files
	quarantineEventsV2filenames, err := m.getQuarantineEventsV2Filenames(inst)
	if err != nil {
		zap.L().Warn("Error grabbing QuarantineEventsV2 files: "+err.Error(), zap.String("module", moduleName))
	}

	// // get all GateKeeper last reject files
	// gatekeeperLastRejectFilenames, err := m.getGatekeeperLastRejectFilenames()
	// if err != nil {
	// 	zap.L().Warn("Error grabbing Gatekeeper files: "+err.Error(), zap.String("module", moduleName))
	// }

	// goroutine to parse each file
	qcount := 0
	for _, file := range quarantineEventsV2filenames {
		v, err := m.parseQuarantineEventsV2Database(file)
		if err != nil {
			zap.L().Error("failed to parse '"+file+"': "+err.Error(), zap.String("module", moduleName))
		} else {
			for _, entry := range v {
				quarantineValues = append(quarantineValues, entry)
				qcount++
			}
		}
	}

	// gcount := 0
	// gatekeeperMW := datawriter.NewOrionWriter(moduleName, mw.GetOrionRuntime(), mw.GetOutputType(), mw.GetOutfilePath())
	// for _, file := range gatekeeperLastRejectFilenames {
	// 	v, err := m.parseGatekeeperLastRejectFile(file)
	// 	if err != nil {
	// 		zap.L().Debug("failed to parse '"+file+"': "+err.Error(), zap.String("module", moduleName))
	// 	} else {
	// 		for _, entry := range v {
	// 			gatekeeperValues = append(gatekeeperValues, entry)
	// 			gcount++
	// 		}
	// 	}
	// }
	zap.L().Debug("Parsed ["+strconv.Itoa(qcount)+"] quarantine artifacts" /*"  and "+strconv.Itoa(gcount)+" gatekeeper artifacts"*/, zap.String("module", moduleName))

	// Write to output
	err = mw.WriteHeader(quarantineheader)
	if err != nil {
		return err
	}
	err = mw.WriteAll(quarantineValues)
	if err != nil {
		return err
	}
	err = mw.Close()
	if err != nil {
		return err
	}
	return nil
}

func (m MacQuarantinesModule) getQuarantineEventsV2Filenames(inst instance.Instance) ([]string, error) {
	quarantineEventsV2filenames := util.Multiglob(quarantineEventsV2Filepaths, inst.GetTargetPath())
	if len(quarantineEventsV2filenames) <= 0 {
		return quarantineEventsV2filenames, errors.New("no QuarantineEventsV2 files were found")
	}
	return quarantineEventsV2filenames, nil
}

func (m MacQuarantinesModule) getGatekeeperLastRejectFilenames(inst instance.Instance) ([]string, error) {
	gatekeeperLastRejectFilenames := util.Multiglob(gatekeeperFilepaths, inst.GetTargetPath())
	if len(gatekeeperLastRejectFilenames) <= 0 {
		return gatekeeperLastRejectFilenames, errors.New("no .LastGKReject files were found")
	}
	return gatekeeperLastRejectFilenames, nil
}

func (m MacQuarantinesModule) parseQuarantineEventsV2Database(dbpath string) ([][]string, error) {
	var entries [][]string

	q := `
	SELECT
		LSQuarantineEventIdentifier,
		COALESCE(LSQuarantineTimeStamp, '') as LSQuarantineTimeStamp,
		COALESCE(LSQuarantineAgentBundleIdentifier, '') as LSQuarantineAgentBundleIdentifier,
		COALESCE(LSQuarantineAgentName, '') as LSQuarantineAgentName,
		COALESCE(LSQuarantineDataURLString, '') as LSQuarantineDataURLString,
		COALESCE(LSQuarantineSenderName, '') as LSQuarantineSenderName,
		COALESCE(LSQuarantineSenderAddress, '') as LSQuarantineSenderAddress,
		COALESCE(LSQuarantineTypeNumber, '') as LSQuarantineTypeNumber,
		COALESCE(LSQuarantineOriginTitle, '') as LSQuarantineOriginTitle,
		COALESCE(LSQuarantineOriginURLString, '') as LSQuarantineOriginURLString,
		COALESCE(LSQuarantineOriginAlias, '') as LSQuarantineOriginAlias
	FROM LSQuarantineEvent`
	dbheaders := []string{
		"LSQuarantineEventIdentifier",       //TEXT
		"LSQuarantineTimeStamp",             // REAL
		"LSQuarantineAgentBundleIdentifier", //TEXT
		"LSQuarantineAgentName",             //TEXT
		"LSQuarantineDataURLString",         //TEXT
		"LSQuarantineSenderName",            //TEXT
		"LSQuarantineSenderAddress",         //TEXT
		"LSQuarantineTypeNumber",            //INTEGER
		"LSQuarantineOriginTitle",           //TEXT
		"LSQuarantineOriginURLString",       //TEXT
		"LSQuarantineOriginAlias",           //BLOB
	}

	entries, err := util.QueryDB(dbpath, q, dbheaders, false)
	if err != nil {
		return [][]string{}, err
	}

	// We have to modify timestamp and prepend user
	timestampIndex := 1
	for _, e := range entries {
		tmp := e[timestampIndex]
		f, err := strconv.ParseFloat(e[timestampIndex], 64)
		if err != nil {
			e[timestampIndex] = tmp + "<FAILED TO CONVERT>"
		}
		e[timestampIndex], err = util.CocoaTime(int64(f))
		if err != nil {
			e[timestampIndex] = tmp + "<FAILED TO CONVERT>"
		}

		e = util.Prepend(e, util.GetUsernameFromPath(dbpath))
	}

	return entries, nil
}

func (m MacQuarantinesModule) parseGatekeeperLastRejectFile(fp string) ([][]string, error) {
	var entries [][]string

	// lastGKRejectMalwareTypes := make(map[int]string)
	// lastGKRejectMalwareTypes[2] = "Unsigned app/program"
	// lastGKRejectMalwareTypes[3] = "Modified Bundle"
	// lastGKRejectMalwareTypes[5] = "Signed App"
	// lastGKRejectMalwareTypes[7] = "Modified App"

	return entries, nil
}

func (m MacQuarantinesModule) readLastGKRejectPlist(filepathLastGKReject string) error {

	// Read plist
	f, err := os.Open(filepathLastGKReject)
	if err != nil {
		return errors.New(moduleName + ": could not read " + filepathLastGKReject + ": " + err.Error())
	}
	p := plist.NewDecoder(f)

	// Grab contents of plist
	var data []interface{}
	err = p.Decode(&data)
	if err != nil {
		return errors.New(moduleName + ": could not decode " + filepathLastGKReject + ": " + err.Error())
	}

	return nil
}
