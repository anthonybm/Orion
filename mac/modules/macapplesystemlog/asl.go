// +build darwin

package macapplesystemlog

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/anthonybm/Orion/datawriter"
	"github.com/anthonybm/Orion/instance"
	"go.uber.org/zap"
)

type MacAppleSystemLogModule struct {
}

var (
	moduleName  = "MacAppleSystemLogModule"
	mode        = "mac"
	version     = "1.0"
	description = `
	Reads and parses the .asl files on disk
	`
	author = "Anthony Martinez, martinez.anthonyb@gmail.com"
)

var (
	filepathAslLocation = "private/var/log/asl/*.asl"
)

// Start executes the module with Config instructions and writes to OrionWriter
func (m MacAppleSystemLogModule) Start(inst instance.Instance) error {
	zap.L().Warn("Does not parse multi-line asl entries", zap.String("module", moduleName))
	err := m.asl(inst)
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
		return err
	}
	return err
}

func (m MacAppleSystemLogModule) asl(inst instance.Instance) error {
	header := []string{
		"source_file",
		"timestamp",
		"system_name",
		"process_name",
		"pid",
		"message",
	}

	mw, err := datawriter.NewOrionWriter(moduleName, inst.GetOrionRuntime(), inst.GetOrionOutputFormat(), inst.GetOrionOutputFilepath())
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
		return err
	}

	values := [][]string{}

	// get all .asl files in path
	files, _ := filepath.Glob(filepath.Join(inst.GetTargetPath(), filepathAslLocation))
	if len(files) == 0 {
		zap.L().Debug("files not found in: '"+filepath.Join(inst.GetTargetPath(), filepathAslLocation)+"'.", zap.String("module", moduleName))
		return nil // Do not throw error for this
	}

	// parse each .asl file
	for _, file := range files {
		v, err := m.parseAslFile(file)
		if err != nil {
			zap.L().Error("failed to parse '"+file+"': "+err.Error(), zap.String("module", moduleName))
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

func (m MacAppleSystemLogModule) parseAslFile(fp string) ([][]string, error) {
	aslCmd := exec.Command("syslog", "-f", fp, "-T", "utc.3")
	aslOut, outerr := aslCmd.StdoutPipe()
	aslErr, errerr := aslCmd.StderrPipe()
	if outerr != nil {
		return [][]string{}, errors.New("aslOut error - could not parse ASL log: " + fp + ": " + outerr.Error())
	}
	if errerr != nil {
		return [][]string{}, errors.New("aslErr error - could not parse ASL log: " + fp + ": " + errerr.Error())
	}
	aslCmd.Start()
	aslOutBytes, outbyteserr := ioutil.ReadAll(aslOut)
	aslErrorBytes, errbyteserr := ioutil.ReadAll(aslErr)
	if outbyteserr != nil {
		return [][]string{}, errors.New("aslOutBytes error - could not parse ASL log: " + fp + ": " + outbyteserr.Error())
	}
	if errbyteserr != nil {
		return [][]string{}, errors.New("aslErrorBytes error - could not parse ASL log: " + fp + ": " + errbyteserr.Error())
	}
	waiterr := aslCmd.Wait()
	if waiterr != nil {
		return [][]string{}, errors.New("aslCmd wait error - could not parse ASL log: " + fp + ": " + waiterr.Error())
	}

	aslOutString := string(aslOutBytes)
	aslErrorString := string(aslErrorBytes)

	if len(aslErrorString) > 0 {
		if !strings.Contains(aslErrorString, "NOTE:") {
			return [][]string{}, errors.New(fmt.Sprintf("aslErrorString not empty - could not parse '%s': %s", fp, aslErrorString))
		}
	}
	if strings.Contains(aslOutString, "Invalid Data Store") {
		return [][]string{}, errors.New(fmt.Sprintf("could not parse '%s'. Invalid Data Store error reported - file may be corrupted.", fp))
	}

	cont, err := m.openAslFileFromSyslog(aslOutString)
	if err != nil {
		return [][]string{}, err
	}
	var entries [][]string

	count := 0
	for i, item := range cont {
		if !strings.Contains(item, "last message repeated") {
			entries = append(entries, m.parseAslEntry(item, fp))
			count = i
		}
	}
	zap.L().Debug("parsed ["+strconv.Itoa(count)+"] items from '"+fp+"'", zap.String("module", moduleName))
	return entries, nil
}

func (m MacAppleSystemLogModule) parseAslEntry(item string, fp string) []string {
	expr := regexp.MustCompile(`^(?P<datetime>\d{4}\-\d{2}\-\d{2} \w\w:\w\w:\w\w\.\d{3}Z) (?P<systemname>.*?) (?P<processName>.*?)\[(?P<PID>[0-9]+)\].*?:\s{0,1}(?P<message>.*(\n	(.*)?\n	(.*)?)?)`)
	res := expr.FindStringSubmatch(item)
	if len(res) > 0 {
		entry := []string{
			fp,
			res[1],
			res[2],
			res[3],
			res[4],
			res[5],
		}
		return entry
	}
	return []string{}
}

func (m MacAppleSystemLogModule) openAslFileFromSyslog(aslOutString string) ([]string, error) {
	return strings.Split(aslOutString, "\n"), nil
}
