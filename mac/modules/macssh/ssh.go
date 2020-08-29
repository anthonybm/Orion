// +build darwin

package macssh

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"

	"github.com/tonythetiger06/Orion/datawriter"
	"github.com/tonythetiger06/Orion/instance"
	"github.com/tonythetiger06/Orion/util"
	"go.uber.org/zap"
)

type MacSSHModule struct {
}

var (
	moduleName  = "MacSSHModule"
	mode        = "mac"
	version     = "1.0"
	description = `
	Reads and parses the SSH known_hosts and authorized_keys on disk
	`
	author = "Anthony Martinez, martinez.anthonyb@gmail.com"
)

var (
	filepathSSHLocations = []string{
		"Users/*/.ssh/known_hosts",
		"Users/*/.ssh/authorized_keys",
		"private/var/*/.ssh/known_hosts",
		"private/var/*/.ssh/authorized_keys",
	}
)

func (m MacSSHModule) Start(inst instance.Instance) error {
	err := m.ssh(inst)
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
	}
	return err
}

func (m MacSSHModule) ssh(inst instance.Instance) error {
	header := []string{
		"source_name",
		"user",
		"bits",
		"fingerprint",
		"host",
		"keytype",
	}
	values := [][]string{}

	mw, err := datawriter.NewOrionWriter(moduleName, inst.GetOrionRuntime(), inst.GetOrionOutputFormat(), inst.GetOrionOutputFilepath())
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
		return err
	}

	// get all ssh files from locations
	filenames := util.Multiglob(filepathSSHLocations, inst.GetTargetPath())

	if len(filenames) == 0 {
		zap.L().Error("Module exiting, files not found in: '"+strings.Join(filepathSSHLocations, " OR ")+"'.", zap.String("module", moduleName))
		return nil // Do not throw error for this
	}

	// parse each ssh file
	count := 0
	countEntries := 0
	for _, file := range filenames {
		v, err := m.parseSSHFile(file)
		if err != nil {
			zap.L().Error("failed to parse '"+file+"': "+err.Error(), zap.String("module", moduleName))
		} else {
			count++
			for _, entry := range v {
				values = append(values, entry)
				countEntries++
			}
		}
	}
	zap.L().Debug(fmt.Sprintf("Parsed %d entries from %d of %d .ssh files", countEntries, count, len(filenames)))

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

func (m MacSSHModule) parseSSHFile(fp string) ([][]string, error) {
	sshCmd := exec.Command("ssh-keygen", "-l", "-f", fp)
	sshOut, outerr := sshCmd.StdoutPipe()
	sshErr, errerr := sshCmd.StderrPipe()
	if outerr != nil {
		return [][]string{}, errors.New("sshOut error - could not parse ssh log: " + fp + ": " + outerr.Error())
	}
	if errerr != nil {
		return [][]string{}, errors.New("sshErr error - could not parse ssh log: " + fp + ": " + errerr.Error())
	}
	sshCmd.Start()
	sshOutBytes, outbyteserr := ioutil.ReadAll(sshOut)
	sshErrorBytes, errbyteserr := ioutil.ReadAll(sshErr)
	if outbyteserr != nil {
		return [][]string{}, errors.New("sshOutBytes error - could not parse ssh log: " + fp + ": " + outbyteserr.Error())
	}
	if errbyteserr != nil {
		return [][]string{}, errors.New("sshErrorBytes error - could not parse ssh log: " + fp + ": " + errbyteserr.Error())
	}
	waiterr := sshCmd.Wait()
	if waiterr != nil {
		return [][]string{}, errors.New("sshCmd wait error - could not parse ssh log: " + fp + ": " + waiterr.Error())
	}

	sshOutString := string(sshOutBytes)
	sshErrorString := string(sshErrorBytes)

	if len(sshErrorString) > 0 {
		if !strings.Contains(sshErrorString, "NOTE:") {
			return [][]string{}, errors.New(fmt.Sprintf("sshErrorString not empty - could not parse '%s': %s", fp, sshErrorString))
		}
	}
	if strings.Contains(sshOutString, "is not a public key file") {
		return [][]string{}, errors.New(fmt.Sprintf("could not parse " + fp + ": " + sshOutString))
	}

	cont, err := m.parseSSHString(sshOutString)
	if err != nil {
		return [][]string{}, err
	}
	var entries [][]string

	count := 0
	for i, item := range cont {
		if len(item) > 0 {
			entries = append(entries, m.parseSSHEntry(item, fp))
			count = i
		}
	}

	zap.L().Debug("parsed ["+strconv.Itoa(count)+"] items from '"+fp+"'", zap.String("module", moduleName))
	return entries, nil
}

func (m MacSSHModule) parseSSHString(sshOutString string) ([]string, error) {
	return strings.Split(sshOutString, "\n"), nil
}

func (m MacSSHModule) parseSSHEntry(item string, fp string) []string {
	data := strings.Split(item, " ")
	entry := []string{
		fp,
		data[0],
		data[1],
		data[2],
		data[3],
	}
	return entry
}
