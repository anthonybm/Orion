// +build darwin

package maclivenetstat

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

type MacLiveNetstat struct {
}

var (
	moduleName  = "MacLiveNetstat"
	mode        = "mac"
	version     = "1.0"
	description = `
	Records current and past network connections on a live system.
	`
	author = "Anthony Martinez, martinez.anthonyb@gmail.com"
)

var (
	netstatHeader = []string{
		"protocol",
		"recv_q",
		"send_q",
		"source_ip",
		"source_port",
		"dest_ip",
		"dest_port",
		"state",
	}
)

func (m MacLiveNetstat) Start(inst instance.Instance) error {
	err := m.netstat(inst)
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
	}
	return err
}

func (m MacLiveNetstat) netstat(inst instance.Instance) error {
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

	netstatCmd := exec.Command("netstat", "-f", "inet", "-n")
	netstatOut, outerr := netstatCmd.StdoutPipe()
	netstatErr, errerr := netstatCmd.StderrPipe()
	if outerr != nil {
		return errors.New("netstatOut error - could not parse netstat query results: " + outerr.Error())
	}
	if errerr != nil {
		return errors.New("netstatErr error - could not parse netstat query results: " + errerr.Error())
	}
	netstatCmd.Start()
	netstatOutBytes, outbyteserr := ioutil.ReadAll(netstatOut)
	netstatErrorBytes, errbyteserr := ioutil.ReadAll(netstatErr)
	if outbyteserr != nil {
		return errors.New("netstatOutBytes error - could not parse netstat query results: " + outbyteserr.Error())
	}
	if errbyteserr != nil {
		return errors.New("netstatErrorBytes error - could not parse netstat query results: " + errbyteserr.Error())
	}
	waiterr := netstatCmd.Wait()
	if waiterr != nil {
		return errors.New("netstatCmd wait error - could not parse netstat query results: " + waiterr.Error())
	}

	netstatOutString := string(netstatOutBytes)
	netstatErrorString := string(netstatErrorBytes)

	if len(netstatErrorString) > 0 {
		if !strings.Contains(netstatErrorString, "NOTE:") {
			return errors.New(fmt.Sprintf("netstatErrorString not empty - could not parse: %s", netstatErrorString))
		}
	}

	cont, err := m.parseNetstatString(netstatOutString)
	if err != nil {
		return err
	}

	for _, item := range cont {
		if len(item) > 0 && !strings.HasPrefix(item, "Active") && !strings.HasPrefix(item, "Proto") {
			values = append(values, m.parseNetstatEntry(item))
			count++
		}
	}

	zap.L().Debug(fmt.Sprintf("Parsed %d netstat entries ", count), zap.String("module", moduleName))

	// Write to output
	err = mw.WriteHeader(netstatHeader)
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

func (m MacLiveNetstat) parseNetstatString(netstatOutString string) ([]string, error) {
	return strings.Split(netstatOutString, "\n"), nil
}

func (m MacLiveNetstat) parseNetstatEntry(item string) []string {
	r := regexp.MustCompile("[^\\s]+")
	data := strings.Split(strings.Join(r.FindAllString(item, -1), " "), " ")
	stateVar := ""
	if len(data) == 6 {
		stateVar = data[5]
	}
	sourceIPVar, sourcePortVar := splitIPandPort(data[3])
	destIPVar, destPortVar := splitIPandPort(data[4])

	entry := []string{
		data[0],
		data[1],
		data[2],
		sourceIPVar,
		sourcePortVar,
		destIPVar,
		destPortVar,
		stateVar,
	}
	return entry
}

func splitIPandPort(str string) (string, string) {
	ip := "ERROR"
	port := "ERROR"
	temp := strings.Split(str, ".")

	// format is either *.PORT or 0.0.0.0.PORT or *.*
	if len(temp) == 0 {
		return ip, port
	}
	if len(temp) == 2 {
		// either *.PORT or *.*
		return temp[0], temp[1]
	}
	return strings.Join(temp[0:len(temp)-2], "."), temp[len(temp)-1]
}
