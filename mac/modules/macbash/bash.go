// +build darwin

package macbash

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	"github.com/anthonybm/Orion/datawriter"
	"github.com/anthonybm/Orion/instance"
	"github.com/anthonybm/Orion/util"
	"github.com/anthonybm/Orion/util/machelpers"
	"go.uber.org/zap"
)

type MacBashModule struct {
}

var (
	moduleName  = "MacBashModule"
	mode        = "mac"
	version     = "1.0"
	description = `
	Reads and parses the .*_history and .bash_sessions on disk
	`
	author = "Anthony Martinez, martinez.anthonyb@gmail.com"
)

var (
	filepathBashLocations = []string{
		"Users/*/.*_history",
		"Users/*/.bash_sessions/*",
		"private/var/*/.*_history",
		"private/var/*/.bash_sessions/*",
	}
)

func (m MacBashModule) Start(inst instance.Instance) error {
	err := m.bash(inst)
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
	}
	return err
}

func (m MacBashModule) bash(inst instance.Instance) error {
	header := []string{
		"mtime",
		"atime",
		"ctime",
		"btime",
		"src_file",
		"user",
		"item_index",
		"cmd",
	}

	mw, err := datawriter.NewOrionWriter(moduleName, inst.GetOrionRuntime(), inst.GetOrionOutputFormat(), inst.GetOrionOutputFilepath())
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
		return err
	}

	values := [][]string{}

	// get users
	files := util.Multiglob(filepathBashLocations, inst.GetTargetPath())
	if len(files) <= 0 {
		zap.L().Warn("No .*_history and .bash_sessions were found.", zap.String("module", moduleName))
	} else {
		zap.L().Debug("Parsing ["+strconv.Itoa(len(files))+"] bash items", zap.String("module", moduleName))
	}

	userlist := []string{}
	// get all bash and history files
	parsedfilecount := 0
	parsedentrycount := 0
	for _, fp := range files {
		user := util.GetUsernameFromPath(fp)
		userlist = append(userlist, user)

		// parse files
		fileMetadata, err := machelpers.FileMetadata(fp, moduleName)
		if err != nil {
			zap.L().Debug("Could not get metadata for '"+fp+"': "+err.Error(), zap.String("module", moduleName))
		}
		file, err := os.Open(fp)
		if err != nil {
			zap.L().Debug("Could not open '"+fp+"': "+err.Error(), zap.String("module", moduleName))
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		index := 0
		for scanner.Scan() {
			line := scanner.Text()

			entry := []string{
				fileMetadata["mtime"],
				fileMetadata["atime"],
				fileMetadata["ctime"],
				fileMetadata["btime"],
				fp,
				user,
				strconv.Itoa(index),
				strings.TrimSpace(line),
			}
			values = append(values, entry)
			parsedentrycount++
		}
		if err := scanner.Err(); err != nil {
			zap.L().Debug("error reading input: "+err.Error(), zap.String("module", moduleName))
		}
		parsedfilecount++
	}

	zap.L().Debug("Parsed ["+strconv.Itoa(parsedentrycount)+"] entries from "+strconv.Itoa(parsedfilecount)+" files", zap.String("module", moduleName))

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
