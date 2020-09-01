// +build darwin

package macutmpx

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/anthonybm/Orion/datawriter"
	"github.com/anthonybm/Orion/instance"
	"github.com/anthonybm/Orion/util"
	"go.uber.org/zap"
)

type MacUtmpxModule struct {
}

var (
	moduleName  = "MacUtmpxModule"
	mode        = "mac"
	version     = "1.0"
	description = `
	read and parse the utmpx file located in /private/var/run/utmpx
	`
	author = "Anthony Martinez, martinez.anthonyb@gmail.com"
)

var (
	header = []string{
		"login_name",
		"id",
		"tty_name",
		"pid",
		"logon_type",
		"timestamp",
		"hostname",
	}
	filepathsUtmpx = []string{
		"private/var/run/utmpx",
	}
	utmpxLineSize = 628
)

func (m MacUtmpxModule) Start(inst instance.Instance) error {
	err := m.utmpx(inst)
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
	}
	return err
}

func (m MacUtmpxModule) utmpx(inst instance.Instance) error {
	mw, err := datawriter.NewOrionWriter(moduleName, inst.GetOrionRuntime(), inst.GetOrionOutputFormat(), inst.GetOrionOutputFilepath())
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
		return err
	}

	values := [][]string{}

	// Start Parsing
	vals, err := m.parseUtmpx(inst)
	if err != nil {
		if strings.HasSuffix(err.Error(), " were found") {
			zap.L().Warn("Error parsing - "+err.Error(), zap.String("module", moduleName))
		} else {
			zap.L().Error("Error parsing UTMPX files: "+err.Error(), zap.String("module", moduleName))
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

func (m MacUtmpxModule) parseUtmpx(inst instance.Instance) ([][]string, error) {
	values := [][]string{}
	count := 0

	utmpxFilepaths := util.Multiglob(filepathsUtmpx, inst.GetTargetPath())
	if len(utmpxFilepaths) == 0 {
		return [][]string{}, errors.New("no UTMPX files were found")
	}

	for _, path := range utmpxFilepaths {
		var valmap = make(map[string]string)
		valmap = util.InitializeMapToEmptyString(valmap, header)

		// Open utmpx file
		f, err := os.Open(path)
		if err != nil {
			zap.L().Error(fmt.Sprintf("could not open '%s': %s", path, err.Error()))
			continue
		}
		defer f.Close()

		// https://opensource.apple.com/source/Libc/Libc-1158.50.2/include/NetBSD/utmpx.h.auto.html
		// https://github.com/libyal/dtformats/blob/master/documentation/Utmp%20login%20records%20format.asciidoc
		type utmpxData struct {
			LoginName [256]uint8
			ID        [4]uint8
			TtyName   [32]uint8
			Pid       int32
			LogonType int16
			Padding   [2]byte
			Epoch     int32
			Usec      int32
			Hostname  [256]uint8
			Reserved  [64]byte
		}

		// Read header of utmpx file, head of reader will point to next segment after this
		headerBuff := make([]byte, utmpxLineSize)
		headerBytesRead, err := f.Read(headerBuff)
		if err != nil {
			zap.L().Error(fmt.Sprintf("failed to read utmpx header from '%s': %s", path, err.Error()))
			continue
		}
		if headerBytesRead != utmpxLineSize {
			zap.L().Error(fmt.Sprintf("utmpx header smaller than expected- expected '%d' got '%d'", utmpxLineSize, headerBytesRead))
			continue
		}
		// fmt.Println(string(headerBuff[:])) // Debug print header - uncomment

		for {
			utmpxBuff := make([]byte, utmpxLineSize)
			utmpxBuffRead, err := io.ReadFull(f, utmpxBuff)
			if err != nil {
				break // Done reading
			}
			if utmpxBuffRead != utmpxLineSize {
				break // Done reading
			}

			utmpxEntry := utmpxData{}
			err = binary.Read(bytes.NewReader(utmpxBuff), binary.LittleEndian, &utmpxEntry)
			if err == io.EOF {
				break
			}

			timestampNum := utmpxEntry.Epoch + utmpxEntry.Usec
			timestamp := time.Unix(int64(timestampNum), 0)

			hostName := util.GetPrintableString(string(utmpxEntry.Hostname[:])) // vars still have non-printable characters due to unpacking

			if hostName == "" {
				hostName = "localhost"
			}

			valmap["login_name"] = util.GetPrintableString(string(utmpxEntry.LoginName[:]))
			valmap["id"] = util.GetPrintableString(string(utmpxEntry.ID[:]))
			valmap["tty_name"] = util.GetPrintableString(string(utmpxEntry.TtyName[:]))
			valmap["pid"] = strconv.FormatInt(int64(utmpxEntry.Pid), 10)
			valmap["logon_type"] = strconv.FormatInt(int64(utmpxEntry.LogonType), 10)
			valmap["timestamp"] = timestamp.UTC().Format(time.RFC3339)
			valmap["hostname"] = hostName

			entry, err := util.UnsafeEntryFromMap(valmap, header)
			if err != nil {
				zap.L().Error("Failed to convert map to entry: "+err.Error(), zap.String("module", moduleName))
				continue
			}
			values = append(values, entry)
			count++
		}
	}
	zap.L().Debug(fmt.Sprintf("Parsed [%d] UTMPX entries", count), zap.String("module", moduleName))

	return values, nil
}
