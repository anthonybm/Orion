// +build darwin

package maclivelsof

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"

	"github.com/anthonybm/Orion/datawriter"
	"github.com/anthonybm/Orion/instance"
	"go.uber.org/zap"
)

type MacLiveLsofModule struct{}

var (
	moduleName  = "MacLiveLsofModule"
	mode        = "mac"
	version     = "1.0"
	description = `
	Records current file handles open when run on a live system.
	`
	author = "Anthony Martinez, martinez.anthonyb@gmail.com"
)

var (
	lsofHeader = []string{"cmd", "pid", "user", "file_descriptor", "type", "device", "size", "node", "name"}
)

func (m MacLiveLsofModule) Start(inst instance.Instance) error {
	err := m.lsof(inst)
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
	}
	return err
}

func (m MacLiveLsofModule) lsof(inst instance.Instance) error {
	mw, err := datawriter.NewOrionWriter(moduleName, inst.GetOrionRuntime(), inst.GetOrionOutputFormat(), inst.GetOrionOutputFilepath())
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
		return err
	}

	values := [][]string{}
	count := 0

	if inst.ForensicMode() {
		return errors.New("running live module in forensic mode")
	}

	lsofCmd := exec.Command("lsof", "-n", "-P", "-F", "pcuftDsin")
	/*
		p pid
		c command
		u user
		f file descriptor
		t type
		D device_no
		s size
		i inode
		n name
	*/
	lsofOut, outerr := lsofCmd.StdoutPipe()
	lsofErr, errerr := lsofCmd.StderrPipe()
	if outerr != nil {
		return errors.New("lsofOut error - could not parse lsof query results: " + outerr.Error())
	}
	if errerr != nil {
		return errors.New("lsofErr error - could not parse lsof query results: " + errerr.Error())
	}
	lsofCmd.Start()
	lsofOutBytes, outbyteserr := ioutil.ReadAll(lsofOut)
	lsofErrorBytes, errbyteserr := ioutil.ReadAll(lsofErr)
	if outbyteserr != nil {
		return errors.New("lsofOutBytes error - could not parse lsof query results: " + outbyteserr.Error())
	}
	if errbyteserr != nil {
		return errors.New("lsofErrorBytes error - could not parse lsof query results: " + errbyteserr.Error())
	}
	waiterr := lsofCmd.Wait()
	if waiterr != nil {
		return errors.New("lsofCmd wait error - could not parse lsof query results: " + waiterr.Error())
	}

	lsofOutString := string(lsofOutBytes)
	lsofErrorString := string(lsofErrorBytes)

	if len(lsofErrorString) > 0 {
		if !strings.Contains(lsofErrorString, "NOTE:") {
			return errors.New(fmt.Sprintf("lsofErrorString not empty - could not parse: %s", lsofErrorString))
		}
	}

	cont, err := m.parseLsofString(lsofOutString)
	if err != nil {
		return err
	}

	entry := make([]string, len(lsofHeader))
	for _, item := range cont {
		if entry == nil {
			entry = make([]string, len(lsofHeader))
		}
		// lsofHeader = []string{"cmd","pid", "user","file_descriptor","type","device","size","node","name"}
		if len(item) > 0 {
			// []string{"cmd","pid", "user","file_descriptor","type","device","size","node","name"}
			switch string(item[0]) {
			case "p":
				entry[1] = string(item[1:])
			case "c":
				entry[0] = string(item[1:])
			case "u":
				entry[2] = string(item[1:])
			case "f":
				entry[3] = string(item[1:])
			case "t":
				entry[4] = string(item[1:])
			case "D":
				entry[5] = string(item[1:])
			case "s":
				entry[6] = string(item[1:])
			case "i":
				entry[7] = string(item[1:])
			case "n":
				entry[8] = string(item[1:])
				values = append(values, entry)
				count++
				entry = nil
			}
		}
	}

	zap.L().Debug(fmt.Sprintf("Parsed %d lsof entries ", count), zap.String("module", moduleName))

	// Write to output
	err = mw.WriteHeader(lsofHeader)
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

func (m MacLiveLsofModule) parseLsofString(lsofOutString string) ([]string, error) {
	return strings.Split(lsofOutString, "\n"), nil
}
