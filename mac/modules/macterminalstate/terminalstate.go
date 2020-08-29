// +build darwin

package macterminalstate

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/tonythetiger06/Orion/datawriter"
	"github.com/tonythetiger06/Orion/instance"
	"github.com/tonythetiger06/Orion/util"
	"github.com/tonythetiger06/Orion/util/machelpers"
	"go.uber.org/zap"
)

var (
	moduleName  = "MacTerminalStateModule"
	mode        = "mac"
	version     = "1.0"
	description = `
	read and parse the savedState files for the Terminal application for each user on disk
	`
	author = "Anthony Martinez, martinez.anthonyb@gmail.com"
)

var (
	filepathTerminalStateLocationGlob = []string{"Users/*/Library/Saved Application State/com.apple.Terminal.savedState", "private/var/*/Library/Saved Application State/com.apple.Terminal.savedState"}
	terminalStateHeader               = []string{"user", "window_id", "datablock", "window_title", "tab_working_directory_url", "tab_working_directory_url_string", "line_index", "line"}
)

// MacTerminalStateModule wraps Module methods
type MacTerminalStateModule struct{}

// Start starts the MacTerminalStateModule, should not be manually called
func (m MacTerminalStateModule) Start(inst instance.Instance) error {
	err := m.terminalstate(inst)
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
	}
	return err
}

func (m MacTerminalStateModule) terminalstate(inst instance.Instance) error {
	mw, err := datawriter.NewOrionWriter(moduleName, inst.GetOrionRuntime(), inst.GetOrionOutputFormat(), inst.GetOrionOutputFilepath())
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
		return err
	}

	values := [][]string{}
	count := 0

	// forensicMode, err := mc.IsForensicMode()
	// if err != nil {
	// 	return err
	// }

	// Start Parsing
	terminalStateLocations := util.Multiglob(filepathTerminalStateLocationGlob, inst.GetTargetPath())
	if len(terminalStateLocations) == 0 {
		zap.L().Debug(fmt.Sprintf("No TerminalState files were found in %s", filepathTerminalStateLocationGlob), zap.String("module", moduleName))
		return nil
	}

	for _, terminalStateLocation := range terminalStateLocations {
		username := util.GetUsernameFromPath(terminalStateLocation)
		zap.L().Debug(fmt.Sprintf("Parsing Terminal State data for user '%s' under '%s'", username, terminalStateLocation), zap.String("module", moduleName))

		// Check if windows.plist and data.data exist under user profiles
		windows, err := filepath.Glob(filepath.Join(terminalStateLocation, "windows.plist"))
		if err != nil {
			zap.L().Error(fmt.Sprintf("when globbing %s - %s", filepath.Join(terminalStateLocation, "windows.plist"), err.Error(), zap.String("module", moduleName)))
			continue
		}
		if len(windows) <= 0 {
			zap.L().Debug(fmt.Sprintf("Required file windows.plist not found, cannot parse Terminal saved state data for '%s'", username), zap.String("module", moduleName))
			continue
		}
		dataLoc, err := filepath.Glob(filepath.Join(terminalStateLocation, "data.data"))
		if err != nil {
			zap.L().Error(fmt.Sprintf("when globbing %s - %s", filepath.Join(terminalStateLocation, "data.data"), err.Error(), zap.String("module", moduleName)))
			continue
		}
		if len(dataLoc) <= 0 {
			zap.L().Debug(fmt.Sprintf("Required file data.data not found, cannot parse Terminal saved state data for '%s'", username), zap.String("module", moduleName))
			continue
		}

		// Check if file header for data.data is NSCR1000
		// open file
		data, err := os.Open(dataLoc[0])
		if err != nil {
			zap.L().Error(fmt.Sprintf("could not open %s - %s", dataLoc[0], err.Error()), zap.String("module", moduleName))
			continue
		}
		// read header
		headerBuff := make([]byte, len("NSCR1000"))
		_, err = data.Read(headerBuff)
		if err != nil {
			zap.L().Error(fmt.Sprintf("could not read %s - %s", dataLoc[0], err.Error()), zap.String("module", moduleName))
			continue
		}
		if string(headerBuff[:]) != "NSCR1000" {
			zap.L().Debug(fmt.Sprintf("Bad file header for data.data - cannot parse further - %s", dataLoc[0]), zap.String("module", moduleName))
			continue
		}

		// Try to read XML and binary style windows.plist files
		windowsPlist, err := machelpers.DecodePlist(windows[0], inst.GetTargetPath()) // is this right?
		if err != nil {
			zap.L().Error(fmt.Sprintf("could not decode %s - %s", windows[0], err.Error()), zap.String("module", moduleName))
			continue
		}

		// Get NSWindowID and NSDataKey values from windows.plist, for each window available
		var windowsData = make(map[uint64]interface{})
		for _, m := range windowsPlist {
			windowID := m["NSWindowID"]

			var decryptionKey interface{}
			if _, ok := m["NSDataKey"]; ok {
				decryptionKey = m["NSDataKey"]
			} else {
				zap.L().Debug(fmt.Sprintf("Could not find decryption key in windows.plist for WindowID %s", windowID), zap.String("module", moduleName))
				decryptionKey = nil
			}

			windowsData[windowID.(uint64)] = decryptionKey
		}
		zap.L().Debug("Num WindowIDs and keys found:"+strconv.Itoa(len(windowsData)), zap.String("module", moduleName))

		// Parse each NSCR1000 block
		dataChunks, err := ioutil.ReadAll(data)
		if err != nil {
			zap.L().Error(fmt.Sprintf("could not read data chunks - %s", err.Error()), zap.String("module", moduleName))
			continue
		}
		for index, dataChunk := range bytes.Split(dataChunks, []byte("NSCR1000")) {
			nsWindowID := binary.BigEndian.Uint32(dataChunk[0:4]) // Big Endian unsigned int
			blocksize := binary.BigEndian.Uint32(dataChunk[4:8])  // Big Endian unsigned int
			available := uint32(len(dataChunk)) + uint32(8)

			// fmt.Println(index, nsWindowID, "BS ", blocksize, "Avail ", available)
			if available == blocksize {
				dataBlock := dataChunk[8 : blocksize-8]

				// check if key was seen
				if _, ok := windowsData[uint64(nsWindowID)]; !ok {
					zap.L().Debug(fmt.Sprintf("Key not found in windows.plist for WindowID %d (datablock %d)", nsWindowID, index), zap.String("module", moduleName))
					continue
				}

				// decrypt data block
				var iv = []byte{35, 46, 57, 24, 85, 35, 24, 74, 87, 35, 88, 98, 66, 32, 14, 05}
				key := windowsData[uint64(nsWindowID)]
				if key == nil {
					zap.L().Debug(fmt.Sprintf("Key found was nil in windows.plist for WindowID %d (datablock %d)", nsWindowID, index), zap.String("module", moduleName))
					continue
				}
				block, err := aes.NewCipher(key.([]byte))
				if err != nil {
					zap.L().Error("Failed to create cipher - "+err.Error(), zap.String("module", moduleName))
					continue
				}
				mode := cipher.NewCBCDecrypter(block, iv)
				mode.CryptBlocks(dataBlock, dataBlock)
				// fmt.Println("DATABLOCK: ", dataBlock)

				// Carve and parse each binary plist from the decrypted blocks
				if bytes.Contains(dataBlock, []byte("bplist")) {
					headerOffset := bytes.Index(dataBlock, []byte("bplist"))
					plistSize := binary.BigEndian.Uint32(dataBlock[headerOffset-4 : headerOffset]) // Big Endian unsigned int
					plistData := dataBlock[headerOffset : headerOffset+int(plistSize)]
					// fmt.Println(string(plistData))
					plistParsedData, err := machelpers.UnarchiveNSKeyedArchiverBytes(plistData)
					// _, err := machelpers.UnarchiveNSKeyedArchiverBytes(plistData)
					if err != nil {
						zap.L().Error("Failed to decode plist - "+err.Error(), zap.String("module", moduleName))
						continue
					}
					for _, plistParsedDataItem := range plistParsedData {
						if !strings.Contains(fmt.Sprint(plistParsedDataItem), "null") {
							if m, ok := plistParsedDataItem.(map[string]interface{}); ok {
								var valmap = make(map[string]string)
								valmap["user"] = username
								valmap["window_id"] = fmt.Sprint(nsWindowID)
								valmap["datablock"] = fmt.Sprint(index)
								valmap["window_title"] = ">>UNIMPLEMENTED<<"

								// fmt.Println(plistParsedDataItem)

								// for k, _ := range m {
								// 	fmt.Println(k)
								// }
								// panic("E")
								// fmt.Println(reflect.TypeOf(m["Window Settings"].([]interface{})[0]).String())
								if val, ok := m["Window Settings"].([]interface{})[0].(map[string]interface{})["Tab Working Directory URL"]; ok {
									// fmt.Println(val)
									valmap["tab_working_directory_url"], _ = util.InterfaceToString(val)
								}
								if val, ok := m["Window Settings"].([]interface{})[0].(map[string]interface{})["Tab Working Directory URL String"]; ok {
									// fmt.Println(val)
									valmap["tab_working_directory_url_string"], _ = util.InterfaceToString(val)
								}
								if val, ok := m["Window Settings"].([]interface{})[0].(map[string]interface{})["Tab Contents v2"]; ok {
									for i, v := range val.([]interface{}) {
										// fmt.Println(fmt.Sprintf("%d - %s", i, v))
										valmap["line"], _ = util.InterfaceToString(v)
										valmap["line_index"] = fmt.Sprint(i)
										// Convert valmap to entry and append to values
										entry, err := util.EntryFromMap(valmap, terminalStateHeader)
										if err != nil {
											zap.L().Debug("Error formatting valmap as entry: "+err.Error(), zap.String("module", moduleName))
											continue
										}
										values = append(values, entry)
										count++
									}
								}
							} else {
								fmt.Println(reflect.TypeOf(plistParsedDataItem).String())
							}
						}
					}
				}
			}
		}
	}

	// End Parsing

	// Write Output
	err = mw.WriteOutput(terminalStateHeader, values)
	if err != nil {
		zap.L().Error(fmt.Sprintf("while writing download output - %s", err.Error()), zap.String("module", moduleName))
	}

	zap.L().Debug(fmt.Sprintf("Parsed [%d] terminal state entries", count), zap.String("module", moduleName))
	return nil
}
