// +build darwin

package macsystemlog

import (
	"bufio"
	"compress/gzip"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/anthonybm/Orion/datawriter"
	"github.com/anthonybm/Orion/instance"
	"go.uber.org/zap"
)

type MacSystemLogModule struct {
}

var (
	moduleName  = "MacSystemLogModule"
	mode        = "mac"
	version     = "1.0"
	description = `
	Reads and parses the system.log files on disk

	Currently does not parse multiline entries such as: 
	Jun 22 00:51:07 ML-C02YW1L4LVDQ syslogd[114]: Configuration Notice:
		ASL Module "com.apple.cdscheduler" claims selected messages.
		Those messages may not appear in standard system log files or in the ASL database.
	`
	author = "Anthony Martinez, martinez.anthonyb@gmail.com"
)

var (
	filepathSystemLogLocation = "private/var/log/system.log*"
)

func (m MacSystemLogModule) Start(inst instance.Instance) error {
	zap.L().Warn("Does not parse multi-line system.log entries", zap.String("module", moduleName))
	err := m.systemLog(inst)
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
	}
	return err
}

func (m MacSystemLogModule) systemLog(inst instance.Instance) error {
	header := []string{
		"source_file",
		"timestamp",
		"system_name",
		"process_name",
		"pid",
		"message",
	}
	values := [][]string{}

	mw, err := datawriter.NewOrionWriter(moduleName, inst.GetOrionRuntime(), inst.GetOrionOutputFormat(), inst.GetOrionOutputFilepath())
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
		return err
	}

	// get all system.log files in path
	files, _ := filepath.Glob(filepath.Join(inst.GetTargetPath(), filepathSystemLogLocation))
	if len(files) == 0 {
		zap.L().Debug("files not found in: '"+filepathSystemLogLocation+"'.", zap.String("module", moduleName))
		return nil // Do not throw error for this
	}

	// parse each system.log file
	for _, item := range files {
		v, e := m.parseSystemLogFile(item)
		if e != nil {
			zap.L().Error("failed to parse '"+item+"': "+e.Error(), zap.String("module", moduleName))
		} else {
			for _, entry := range v {
				values = append(values, entry)
			}
		}
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

func (m MacSystemLogModule) parseSystemLogFile(fp string) ([][]string, error) {
	var entries [][]string

	cont, err := m.openSystemLogFileIntoMemory(fp)
	if err != nil {
		return [][]string{}, err
	}
	text := string(cont)
	var re = regexp.MustCompile(`(?m)(?P<month>^[A-Za-z]{3}) (?P<day>[0-9]{2}) (?P<time>\d\d:\d\d:\d\d) (?P<system_name>.*?) (?P<process_name>.*?)\[(?P<pid>[0-9]+)\].*?:\s{0,1}(?P<message>.*(\n	(.*)?\n	(.*)?)?)`)
	count := 0
	for _, match := range re.FindAllString(text, -1) {
		// zap.L().Debug("parsed item from "+fp, zap.String("module", moduleName), zap.String("contents", match))
		entries = append(entries, m.parseSystemLogEntry(match, fp))
		count++
	}
	zap.L().Debug("parsed ["+strconv.Itoa(count)+"] items from '"+fp+"'", zap.String("module", moduleName))

	return entries, nil
}

func (m MacSystemLogModule) openSystemLogFileIntoMemory(fp string) ([]byte, error) {
	if strings.HasSuffix(fp, ".gz") {
		file, err := os.Open(fp)
		if err != nil {
			zap.L().Debug("failed to open '"+fp+"': "+err.Error(), zap.String("module", moduleName))
			return nil, err
		}

		gz, err := gzip.NewReader(file)
		if err != nil {
			zap.L().Debug("failed to open as gzip '"+fp+"': "+err.Error(), zap.String("module", moduleName))
			return nil, err
		}

		defer file.Close()
		defer gz.Close()

		cont, err := ioutil.ReadAll(gz)
		return cont, err
	}

	file, err := os.Open(fp)
	if err != nil {
		zap.L().Debug("failed to open '"+fp+"': "+err.Error(), zap.String("module", moduleName))
		return nil, err
	}

	defer file.Close()

	cont, err := ioutil.ReadAll(file)
	return cont, err
}

func (m MacSystemLogModule) parseSystemLogEntry(item string, fp string) []string {
	// expr := regexp.MustCompile(`(?P<month>^[A-Za-z]{3}) (?P<day>[0-9]{2}) (?P<time>\d\d:\d\d:\d\d) (?P<system_name>.*?) (?P<process_name>.*?)\[(?P<pid>[0-9]+)\].*?:\s{0,1}(?P<message>.*)`)
	// below expression acts on single line reads from scanner
	// multiline expression that would be captured from whole file is not
	expr := regexp.MustCompile(`(?m)(?P<month>^[A-Za-z]{3}) (?P<day>[0-9]{2}) (?P<time>\d\d:\d\d:\d\d) (?P<system_name>.*?) (?P<process_name>.*?)\[(?P<pid>[0-9]+)\].*?:\s{0,1}(?P<message>.*(\n	(.*)?\n	(.*)?)?)`)
	res := expr.FindStringSubmatch(item)
	if len(res) > 0 {
		entry := []string{
			fp,
			res[1] + " " + res[2] + " " + res[3],
			res[4],
			res[5],
			res[6],
			res[7] + " " + res[9] + " " + res[10],
		}
		return entry
	}
	return []string{}
}

func (m MacSystemLogModule) openSystemLogFile(fp string) (*bufio.Scanner, error) {
	if strings.HasSuffix(fp, ".gz") {
		file, err := os.Open(fp)
		if err != nil {
			zap.L().Debug("failed to open '"+fp+"': "+err.Error(), zap.String("module", moduleName))
			return nil, err
		}

		gz, err := gzip.NewReader(file)
		if err != nil {
			zap.L().Debug("failed to open as gzip '"+fp+"': "+err.Error(), zap.String("module", moduleName))
			return nil, err
		}

		defer file.Close()
		defer gz.Close()

		scanner := bufio.NewScanner(gz)
		return scanner, nil
	}
	file, err := os.Open(fp)
	if err != nil {
		zap.L().Debug("failed to open '"+fp+"': "+err.Error(), zap.String("module", moduleName))
		return nil, err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	return scanner, nil
}
