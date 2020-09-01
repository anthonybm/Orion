// +build darwin

package maclivepslist

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"regexp"
	"strings"

	"github.com/anthonybm/Orion/datawriter"
	"github.com/anthonybm/Orion/instance"
	"go.uber.org/zap"
)

type MacLivePslistModule struct{}

var (
	moduleName  = "MacLivePslistModule"
	mode        = "mac"
	version     = "1.0"
	description = `
	Records current process listing when run on a live system. 
	`
	author = "Anthony Martinez, martinez.anthonyb@gmail.com"
)

var (
	pslistHeader = []string{"pid", "ppid", "user", "state", "proc_start", "runtime", "cmd"}
)

func (m MacLivePslistModule) Start(inst instance.Instance) error {
	err := m.pslist(inst)
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
	}
	return err
}

func (m MacLivePslistModule) pslist(inst instance.Instance) error {
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

	pslistCmd := exec.Command("ps", "-Ao", "pid,ppid,user,stat,lstart,time,command")
	pslistOut, outerr := pslistCmd.StdoutPipe()
	pslistErr, errerr := pslistCmd.StderrPipe()
	if outerr != nil {
		return errors.New("pslistOut error - could not parse pslist query results: " + outerr.Error())
	}
	if errerr != nil {
		return errors.New("pslistErr error - could not parse pslist query results: " + errerr.Error())
	}
	pslistCmd.Start()
	pslistOutBytes, outbyteserr := ioutil.ReadAll(pslistOut)
	pslistErrorBytes, errbyteserr := ioutil.ReadAll(pslistErr)
	if outbyteserr != nil {
		return errors.New("pslistOutBytes error - could not parse pslist query results: " + outbyteserr.Error())
	}
	if errbyteserr != nil {
		return errors.New("pslistErrorBytes error - could not parse pslist query results: " + errbyteserr.Error())
	}
	waiterr := pslistCmd.Wait()
	if waiterr != nil {
		return errors.New("pslistCmd wait error - could not parse pslist query results: " + waiterr.Error())
	}

	pslistOutString := string(pslistOutBytes)
	pslistErrorString := string(pslistErrorBytes)

	if len(pslistErrorString) > 0 {
		if !strings.Contains(pslistErrorString, "NOTE:") {
			return errors.New(fmt.Sprintf("pslistErrorString not empty - could not parse: %s", pslistErrorString))
		}
	}

	cont, err := m.parsePslistString(pslistOutString)
	if err != nil {
		return err
	}

	for _, item := range cont {
		if len(item) > 0 && !strings.Contains(item, "PID") {
			values = append(values, m.parsePslistEntry(item))
			count++
		}
	}

	zap.L().Debug(fmt.Sprintf("Parsed %d pslist entries ", count), zap.String("module", moduleName))

	// Write to output
	err = mw.WriteHeader(pslistHeader)
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

func (m MacLivePslistModule) parsePslistString(pslistOutString string) ([]string, error) {
	return strings.Split(pslistOutString, "\n"), nil
}

func (m MacLivePslistModule) parsePslistEntry(item string) []string {
	r := regexp.MustCompile("[^\\s]+")
	data := strings.Split(strings.Join(r.FindAllString(item, -1), " "), " ")
	// fmt.Println(data)
	processStartTime := strings.Join(data[5:9], " ")
	// fmt.Println(strings.Join(data[5:9], " "))

	entry := []string{
		data[0], // pid
		data[1], // ppid
		data[2], // user
		data[3], // state
		processStartTime,
		data[9],                              // runtime
		string(strings.Join(data[10:], " ")), // cmd
	}
	return entry
}
