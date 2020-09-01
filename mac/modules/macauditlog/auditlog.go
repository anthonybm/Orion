// +build darwin

package macauditlog

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"

	"github.com/anthonybm/Orion/datawriter"
	"github.com/anthonybm/Orion/instance"
	"github.com/anthonybm/Orion/util"
	"github.com/beevik/etree"
	"go.uber.org/zap"
)

type MacAuditLogModule struct{}

var (
	moduleName  = "MacAuditLogModule"
	mode        = "mac"
	version     = "1.0"
	description = `
	Reads and parses audit log files on disk
	`
	author = "Anthony Martinez, martinez.anthonyb@gmail.com"
)

var (
	header = []string{
		"source_file",
		"timestamp",
		"version",
		"event",
		"modifier",
		"msec",
		"audit_uid",
		"uid",
		"gid",
		"ruid",
		"rgid",
		"pid",
		"sid",
		"tid",
		"errval",
		"retval",
		"text_fields",
	}
	filepathsAuditLogs = []string{
		"private/var/audit/*",
	}
)

func (m MacAuditLogModule) Start(inst instance.Instance) error {
	err := m.auditlog(inst)
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
	}
	return err
}

func (m MacAuditLogModule) auditlog(inst instance.Instance) error {
	zap.L().Warn("Experimental module - no test data was used to generate - verify results!", zap.String("module", moduleName))

	mw, err := datawriter.NewOrionWriter(moduleName, inst.GetOrionRuntime(), inst.GetOrionOutputFormat(), inst.GetOrionOutputFilepath())
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
		return err
	}

	values := [][]string{}

	// Start Parsing
	vals, err := m.parseAuditLogs(inst)
	if err != nil {
		if strings.HasSuffix(err.Error(), " were found") {
			zap.L().Warn("Error parsing - "+err.Error(), zap.String("module", moduleName))
		} else {
			zap.L().Error("Error parsing audit logs: "+err.Error(), zap.String("module", moduleName))
		}
	} else {
		values = util.AppendToDoubleSlice(values, vals)
	}

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

func (m MacAuditLogModule) parseAuditLogs(inst instance.Instance) ([][]string, error) {
	values := [][]string{}
	count := 0

	auditLogPaths := util.Multiglob(filepathsAuditLogs, inst.GetTargetPath())
	if len(auditLogPaths) == 0 {
		return [][]string{}, errors.New("no audit log files were found")
	}

	for _, path := range auditLogPaths {
		var valmap = make(map[string]string)
		valmap = util.InitializeMapToEmptyString(valmap, header)

		v, err := m.parseAuditLogFile(path)
		if err != nil {
			zap.L().Error("failed to parse '"+path+"': "+err.Error(), zap.String("module", moduleName))
		} else if len(v) == 0 {
			zap.L().Debug(fmt.Sprintf("Audit log file '%s' had no records that could be parsed", path), zap.String("module", moduleName))
		} else {
			for _, entry := range v {
				values = append(values, entry)
				count++
			}
		}
	}
	zap.L().Debug(fmt.Sprintf("Parsed [%d] audit log entries", count), zap.String("module", moduleName))

	return values, nil
}

func (m MacAuditLogModule) parseAuditLogFile(fp string) ([][]string, error) {
	auditLogCmd := exec.Command("praudit", "-x", "-l", fp)
	auditLogOut, outerr := auditLogCmd.StdoutPipe()
	auditLogErr, errerr := auditLogCmd.StderrPipe()
	if outerr != nil {
		return [][]string{}, errors.New("auditLogOut error - could not parse audit log: " + fp + ": " + outerr.Error())
	}
	if errerr != nil {
		return [][]string{}, errors.New("auditLogErr error - could not parse audit log: " + fp + ": " + errerr.Error())
	}
	auditLogCmd.Start()
	auditLogOutBytes, outbyteserr := ioutil.ReadAll(auditLogOut)
	auditLogErrorBytes, errbyteserr := ioutil.ReadAll(auditLogErr)
	if outbyteserr != nil {
		return [][]string{}, errors.New("auditLogOutBytes error - could not parse audit log: " + fp + ": " + outbyteserr.Error())
	}
	if errbyteserr != nil {
		return [][]string{}, errors.New("auditLogErrorBytes error - could not parse audit log: " + fp + ": " + errbyteserr.Error())
	}
	waiterr := auditLogCmd.Wait()
	if waiterr != nil {
		return [][]string{}, errors.New("auditLogCmd wait error - could not parse audit log: " + fp + ": " + waiterr.Error())
	}

	auditLogOutString := string(auditLogOutBytes)
	auditLogErrorString := string(auditLogErrorBytes)

	if len(auditLogErrorString) > 0 {
		return [][]string{}, fmt.Errorf("auditLogErrorString not empty - could not parse '%s': %s", fp, auditLogErrorString)
	}

	cont, err := m.parseAuditString(auditLogOutString)
	if err != nil {
		return [][]string{}, err
	}
	if len(cont) == 0 {
		return [][]string{}, nil
	}
	var entries [][]string

	// Parse audit log items
	count := 0
	for _, item := range cont {
		if len(item) > 0 {
			entry, err := m.parseAuditEntry(item, fp)
			if err != nil {
				zap.L().Error(fmt.Sprintf("Error reading audit log item: %s", err.Error()), zap.String("module", moduleName))
			}
			entries = append(entries, entry)
			// fmt.Println(item)
			count++
		}
	}

	zap.L().Debug("parsed ["+strconv.Itoa(count)+"] items from '"+fp+"'", zap.String("module", moduleName))
	return entries, nil
}

func (m MacAuditLogModule) parseAuditString(auditOutString string) ([]string, error) {
	return strings.Split(auditOutString, "\n"), nil
}

func (m MacAuditLogModule) parseAuditEntry(item, fp string) ([]string, error) {
	var entry []string
	var valmap = make(map[string]string)

	doc := etree.NewDocument()
	if err := doc.ReadFromString(item); err != nil {
		return []string{}, err
	}

	// Get root attributes
	rootKeys := []string{"time", "modifier", "msec", "version", "event"}
	for _, key := range rootKeys {
		valmap[key] = doc.Root().SelectAttrValue(key, "ERROR-DNE")
	}

	// Get subject attributes and text attributes from XML record
	textFields := []string{}
	subjectValues := []etree.Attr{}
	returnValues := []etree.Attr{}
	for _, child := range doc.Root().ChildElements() {
		if child.Tag == "subject" {
			subjectValues = child.Attr
		} else if child.Tag == "text" {
			textFields = append(textFields, child.Text())
		} else if child.Tag == "return" {
			returnValues = child.Attr
		}
	}

	// Parse subject values
	if len(subjectValues) == 0 {
		zap.L().Debug(fmt.Sprintf("XML record %s from '%s' does not contain 'subject' key", item, fp), zap.String("module", moduleName))
	} else {
		subjectKeys := []string{"audit-uid", "uid", "gid", "ruid", "rgid", "pid", "sid", "tid"}
		for _, key := range subjectKeys {
			var val string
			if key == "audit-uid" {
				for _, attr := range subjectValues {
					if attr.Key == key {
						val = attr.Value
					}
				}
				valmap["audit-uid"] = val
			} else {
				for _, attr := range subjectValues {
					if attr.Key == key {
						val = attr.Value
					}
				}
				valmap[key] = val
			}
		}
	}

	// Parse return values
	if len(returnValues) == 0 {
		zap.L().Debug(fmt.Sprintf("XML record %s from '%s' does not contain 'return' key", item, fp), zap.String("module", moduleName))
	} else {
		returnKeys := []string{"errval", "retval"}
		for _, key := range returnKeys {
			var val string
			for _, attr := range returnValues {
				if attr.Key == key {
					val = attr.Value
				}
			}
			valmap[key] = val
		}
	}

	// Write text fields
	valmap["text_fields"] = strings.Join(textFields, " ")

	return entry, nil
}
