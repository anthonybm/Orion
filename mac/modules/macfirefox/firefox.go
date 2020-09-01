// +build darwin

package macfirefox

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthonybm/Orion/datawriter"
	"github.com/anthonybm/Orion/instance"
	"github.com/anthonybm/Orion/util"
	"go.uber.org/zap"
)

// MacFirefoxModule wraps the methods for the module to run
type MacFirefoxModule struct {
}

var (
	moduleName  = "MacFirefoxModule"
	mode        = "mac"
	version     = "1.0"
	description = `
	read and parse the firefox history database for each user on disk
	`
	author = "Anthony Martinez, martinez.anthonyb@gmail.com"
)

var (
	filepathTemporary           = "FirefoxTemp"
	filepathFirefoxLocationGlob = []string{"Users/*/Library/Application Support/Firefox/Profiles/*.*"}
	historyHeader               = []string{"user", "profile", "visit_date", "title", "url", "visit_count", "last_visit_date", "typed", "description"}
	downloadHeader              = []string{"user", "profile", "download_url", "download_path", "download_started", "download_finished", "download_totalbytes"}
	extensionHeader             = []string{"user", "profile", "name", "id", "creator", "description", "updateURL", "installDate", "updateDate", "sourceURI", "homepageURL"}
)

// Start starts the MacFirefoxModule, should not be manually called
func (m MacFirefoxModule) Start(inst instance.Instance) error {
	err := m.firefox(inst)
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
	}
	return err
}

func (m MacFirefoxModule) firefox(inst instance.Instance) error {
	mw, err := datawriter.NewOrionWriter(moduleName, inst.GetOrionRuntime(), inst.GetOrionOutputFormat(), inst.GetOrionOutputFilepath())
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
		return err
	}
	// forensicMode, err := mc.IsForensicMode()
	// if err != nil {
	// 	return err
	// }

	downloadValues := [][]string{}
	historyValues := [][]string{}
	extensionValues := [][]string{}

	// Start Parsing
	firefoxLocations := util.Multiglob(filepathFirefoxLocationGlob, inst.GetTargetPath())
	if len(firefoxLocations) == 0 {
		zap.L().Debug(fmt.Sprintf("No firefox files were found in %s", filepathFirefoxLocationGlob), zap.String("module", moduleName))
	} else {
		for _, firefoxLocation := range firefoxLocations {
			username := util.GetUsernameFromPath(firefoxLocation)
			profile := strings.Split(firefoxLocation, "/")[len(strings.Split(firefoxLocation, "/"))-1]
			zap.L().Debug(fmt.Sprintf("Started parsing for Firefox user %s", username), zap.String("module", moduleName))

			dbfilepath := filepath.Join(firefoxLocation, "places.sqlite")

			parseVisitHistoryValues, err := m.parseVisitHistory(dbfilepath, username, profile)
			if err != nil {
				if strings.Contains(err.Error(), "found no columns") || strings.Contains(err.Error(), "no such table") {
					zap.L().Debug(fmt.Sprintf("firefox visit history - %s", err.Error()), zap.String("module", moduleName))
				} else {
					zap.L().Error(fmt.Sprintf("firefox visit history - %s", err.Error()), zap.String("module", moduleName))
				}
			} else {
				historyValues = append(historyValues, parseVisitHistoryValues...)
			}
			parseDownloadHistoryValues, err := m.parseDownloadHistory(dbfilepath, username, profile)
			if err != nil {
				if strings.Contains(err.Error(), "found no columns") || strings.Contains(err.Error(), "no such table") {
					zap.L().Debug(fmt.Sprintf("firefox download history - %s", err.Error()), zap.String("module", moduleName))
				} else {
					zap.L().Error(fmt.Sprintf("firefox download history - %s", err.Error()), zap.String("module", moduleName))
				}
			} else {
				downloadValues = append(downloadValues, parseDownloadHistoryValues...)
			}
			parseExtensionsValues, err := m.parseExtensionsValues(filepath.Join(firefoxLocation, "extensions.json"), username, profile)
			if err != nil {
				if strings.Contains(err.Error(), "found no columns") || strings.Contains(err.Error(), "no such table") {
					zap.L().Debug(fmt.Sprintf("firefox extensions history - %s", err.Error()), zap.String("module", moduleName))
				} else {
					zap.L().Error(fmt.Sprintf("firefox extensions history - %s", err.Error()), zap.String("module", moduleName))
				}
			} else {
				extensionValues = append(extensionValues, parseExtensionsValues...)
			}
			// remove temp created files

		}
	}

	// End Parsing

	// Start Write to output
	// firefox download output
	downloadOrionWriter, err := datawriter.NewOrionWriter(moduleName+"-downloads", mw.GetOrionRuntime(), mw.GetOutputType(), filepath.Dir(mw.GetOutfilePath()))
	if err != nil {
		zap.L().Error(err.Error(), zap.String("module", moduleName))
	} else {
		err := downloadOrionWriter.WriteOutput(downloadHeader, downloadValues)
		if err != nil {
			zap.L().Error(fmt.Sprintf("while writing download output - %s", err.Error()), zap.String("module", moduleName))
		}
	}
	// firefox history output
	historyOrionWriter, err := datawriter.NewOrionWriter(moduleName+"-history", mw.GetOrionRuntime(), mw.GetOutputType(), filepath.Dir(mw.GetOutfilePath()))
	if err != nil {
		zap.L().Error(err.Error(), zap.String("module", moduleName))
	} else {
		err := historyOrionWriter.WriteOutput(historyHeader, historyValues)
		if err != nil {
			zap.L().Error(fmt.Sprintf("while writing history output - %s", err.Error()), zap.String("module", moduleName))
		}
	}
	// firefox extensions output
	extensionsOrionWriter, err := datawriter.NewOrionWriter(moduleName+"-extensions", mw.GetOrionRuntime(), mw.GetOutputType(), filepath.Dir(mw.GetOutfilePath()))
	if err != nil {
		zap.L().Error(err.Error(), zap.String("module", moduleName))
	} else {
		err := extensionsOrionWriter.WriteOutput(extensionHeader, extensionValues)
		if err != nil {
			zap.L().Error(fmt.Sprintf("while writing extension output - %s", err.Error()), zap.String("module", moduleName))
		}
	}
	// End Write to output

	// Remove general orionwriter
	err = mw.SelfDestruct()
	if err != nil {
		zap.L().Error(fmt.Sprintf("while deleting general orionwriter - %s", err.Error()), zap.String("module", moduleName))
	}

	// Delete folder temp
	err = os.Remove(filepathTemporary)
	if err != nil {
		zap.L().Error(err.Error(), zap.String("module", moduleName))
	}

	return nil
}

func (m MacFirefoxModule) parseVisitHistory(dbfilepath, username, profile string) ([][]string, error) {
	// copy db to temp
	var firefoxDBPathTemp string
	firefoxDBPathTemp = filepathTemporary + "/" + "places.sqlite-tmp"
	err := util.CopyFile(dbfilepath, firefoxDBPathTemp, os.FileMode(int(0777)))
	if err != nil {
		zap.L().Error("Failed to copy Firefox History to temp: " + err.Error())
		return [][]string{}, err
	}
	// var firefoxDBShmPathTemp string
	// firefoxDBShmPathTemp = filepathTemporary + "/" + "places.sqlite-tmp-shm"
	// err = util.CopyFile(dbfilepath, firefoxDBShmPathTemp, os.FileMode(int(0777)))
	// if err != nil {
	// 	zap.L().Error("Failed to copy Firefox History to temp: " + err.Error())
	// 	return [][]string{}, err
	// }
	// var firefoxDBWalPathTemp string
	// firefoxDBWalPathTemp = filepathTemporary + "/" + "places.sqlite-tmp-wal"
	// err = util.CopyFile(dbfilepath, firefoxDBWalPathTemp, os.FileMode(int(0777)))
	// if err != nil {
	// 	zap.L().Error("Failed to copy Firefox History to temp: " + err.Error())
	// 	return [][]string{}, err
	// }

	// query and query header
	values := [][]string{}
	wantedCols := []string{"visit_date", "title", "url", "visit_count", "typed", "last_visit_date", "description"}
	actualCols := []string{} // compared to wanted
	queryCols := []string{}  // actually sent to query
	mozPlacesCols, err := util.DBColumnNames(dbfilepath, "moz_places")
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			zap.L().Debug(err.Error(), zap.String("module", moduleName))
		} else {
			zap.L().Error(err.Error(), zap.String("module", moduleName))
		}
	} else {
		actualCols = append(actualCols, mozPlacesCols...)
	}
	mozAnnosCols, err := util.DBColumnNames(dbfilepath, "moz_historyvisits")
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			zap.L().Debug(err.Error(), zap.String("module", moduleName))
		} else {
			zap.L().Error(err.Error(), zap.String("module", moduleName))
		}
	} else {
		actualCols = append(actualCols, mozAnnosCols...)
	}
	for _, col := range wantedCols {
		if util.SliceContainsString(actualCols, col) {
			queryCols = append(queryCols, fmt.Sprintf("COALESCE(%s,'') as %s", col, col))
		}
	}

	if len(queryCols) == 0 {
		return [][]string{}, fmt.Errorf("found no columns for '%s'", dbfilepath)
	}

	// send query to db
	query := fmt.Sprintf("SELECT %s FROM moz_historyvisits left join moz_places on moz_places.id = moz_historyvisits.place_id", strings.Join(queryCols, ", "))
	entries, err := util.UnsafeQueryDBToMap(firefoxDBPathTemp, query)
	if err != nil {
		return [][]string{}, err
	}

	// parse entries
	count := 0
	for _, e := range entries {
		var valmap = make(map[string]string)
		util.InitializeMapToEmptyString(valmap, historyHeader)

		valmap["user"] = username
		valmap["profile"] = profile
		for k, v := range e {
			val, err := util.InterfaceToString(v)
			if err != nil {
				zap.L().Error(err.Error(), zap.String("module", moduleName))
				valmap[k] = "ERR"
			} else {
				valmap[k] = strings.TrimSpace(val)
			}
		}

		// Convert valmap to entry and append to values
		entry, err := util.UnsafeEntryFromMap(valmap, historyHeader)
		if err != nil {
			zap.L().Debug("Error formatting valmap as entry: "+err.Error(), zap.String("module", moduleName))
			continue
		}
		values = append(values, entry)
		count++
	}
	zap.L().Debug(fmt.Sprintf("Parsed [%d] visit history entries from '%s'", count, dbfilepath), zap.String("module", moduleName))

	// Delete copy from Temp
	err = os.Remove(firefoxDBPathTemp)
	if err != nil {
		zap.L().Error(err.Error(), zap.String("module", moduleName))
	}

	return values, nil
}

func (m MacFirefoxModule) parseDownloadHistory(dbfilepath, username, profile string) ([][]string, error) {
	// copy db to temp
	var firefoxDBPathTemp string
	firefoxDBPathTemp = filepathTemporary + "/" + "places.sqlite-tmp"
	err := util.CopyFile(dbfilepath, firefoxDBPathTemp, os.FileMode(int(0777)))
	if err != nil {
		zap.L().Error("Failed to copy Firefox History to temp: " + err.Error())
		return [][]string{}, err
	}

	// query and query header
	values := [][]string{}
	wantedCols := []string{"url", "content", "dateAdded"}
	actualCols := []string{} // compared to wanted
	queryCols := []string{}  // actually sent to query
	mozPlacesCols, err := util.DBColumnNames(dbfilepath, "moz_places")
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			zap.L().Debug(err.Error(), zap.String("module", moduleName))
		} else {
			zap.L().Error(err.Error(), zap.String("module", moduleName))
		}
	} else {
		actualCols = append(actualCols, mozPlacesCols...)
	}
	mozAnnosCols, err := util.DBColumnNames(dbfilepath, "moz_annos")
	if err != nil {
		if strings.Contains(err.Error(), "no such table") {
			zap.L().Debug(err.Error(), zap.String("module", moduleName))
		} else {
			zap.L().Error(err.Error(), zap.String("module", moduleName))
		}
	} else {
		actualCols = append(actualCols, mozAnnosCols...)
	}
	for _, col := range wantedCols {
		if util.SliceContainsString(actualCols, col) {
			queryCols = append(queryCols, fmt.Sprintf("COALESCE(%s,'') as %s", col, col))
		}
	}

	if len(queryCols) == 0 {
		return [][]string{}, fmt.Errorf("found no columns for '%s'", dbfilepath)
	}

	// send query to db
	query := `
	SELECT url,group_concat(content),dateAdded 
	FROM moz_annos
    LEFT JOIN moz_places ON moz_places.id = moz_annos.place_id
    GROUP BY place_id`
	entries, err := util.UnsafeQueryDBToMap(firefoxDBPathTemp, query)
	if err != nil {
		return [][]string{}, err
	}

	// parse entries
	count := 0
	for _, e := range entries {
		var valmap = make(map[string]string)
		util.InitializeMapToEmptyString(valmap, downloadHeader)

		valmap["user"] = username
		valmap["profile"] = profile
		for k, v := range e {
			val, err := util.InterfaceToString(v)
			if err != nil {
				zap.L().Error(err.Error(), zap.String("module", moduleName))
				valmap[k] = "ERR"
			} else {
				valmap[k] = strings.TrimSpace(val)
			}
		}

		// Convert valmap to entry and append to values
		entry, err := util.UnsafeEntryFromMap(valmap, downloadHeader)
		if err != nil {
			zap.L().Debug("Error formatting valmap as entry: "+err.Error(), zap.String("module", moduleName))
			continue
		}
		values = append(values, entry)
		count++
	}
	zap.L().Debug(fmt.Sprintf("Parsed [%d] download entries from '%s'", count, dbfilepath), zap.String("module", moduleName))

	// Delete copy from Temp
	err = os.Remove(firefoxDBPathTemp)
	if err != nil {
		zap.L().Error(err.Error(), zap.String("module", moduleName))
	}

	zap.L().Warn("No test data used for firefox download history - VERIFY and update :) ", zap.String("module", moduleName))
	return values, nil
}
func (m MacFirefoxModule) parseExtensionsValues(extensionFilepath, username, profile string) ([][]string, error) {
	values := [][]string{}
	count := 0

	extensionFilepathExists, err := util.Exists(extensionFilepath)
	if err != nil {
		return [][]string{}, fmt.Errorf("failed to find file - %s", err.Error())
	}
	if extensionFilepathExists {
		extensionsContents, err := ioutil.ReadFile(extensionFilepath)
		if err != nil {
			return [][]string{}, fmt.Errorf("failed to read file - %s", err.Error())
		}
		// TODO: format struct instead of interface unmarshalling
		var extensionsData interface{}
		err = json.Unmarshal(extensionsContents, &extensionsData)
		if err != nil {
			return [][]string{}, fmt.Errorf("failed to unmarshall data - %s", err.Error())
		}

		// proceed with json data from file
		addons, err := util.JSONGetValueFromKey(extensionsData, "addons")
		if err != nil {
			return [][]string{}, fmt.Errorf("failed to get 'addons' values - %s", err.Error())
		}
		for _, addon := range addons.([]interface{}) {
			var valmap = make(map[string]string)
			valmap = util.InitializeMapToEmptyString(valmap, extensionHeader)
			if defaultLocale, ok := addon.(map[string]interface{})["defaultLocale"]; ok {
				if nameVal, ok := defaultLocale.(map[string]interface{})["name"]; ok {
					if val, err := util.InterfaceToString(nameVal); err != nil {
						valmap["name"] = "ERR"
						zap.L().Debug(err.Error(), zap.String("module", moduleName))
					} else {
						valmap["name"] = val
					}
				}
				if creatorVal, ok := defaultLocale.(map[string]interface{})["creator"]; ok {
					if val, err := util.InterfaceToString(creatorVal); err != nil {
						valmap["creator"] = "ERR"
						zap.L().Debug(err.Error(), zap.String("module", moduleName))
					} else {
						valmap["creator"] = val
					}
				}
				if descriptionVal, ok := defaultLocale.(map[string]interface{})["description"]; ok {
					if val, err := util.InterfaceToString(descriptionVal); err != nil {
						valmap["description"] = "ERR"
						zap.L().Debug(err.Error(), zap.String("module", moduleName))
					} else {
						valmap["description"] = val
					}
				}
				if homepageVal, ok := defaultLocale.(map[string]interface{})["homepage"]; ok {
					if val, err := util.InterfaceToString(homepageVal); err != nil {
						valmap["homepage"] = "ERR"
						zap.L().Debug(err.Error(), zap.String("module", moduleName))
					} else {
						valmap["homepage"] = val
					}
				}
			}

			valmap["user"] = username
			valmap["profile"] = profile
			if val, err := util.InterfaceToString(addon.(map[string]interface{})["id"]); err != nil {
				valmap["id"] = "ERR"
				zap.L().Debug(err.Error(), zap.String("module", moduleName))
			} else {
				valmap["id"] = val
			}
			if val, err := util.InterfaceToString(addon.(map[string]interface{})["updateURL"]); err != nil {
				valmap["updateURL"] = "ERR"
				zap.L().Debug(err.Error(), zap.String("module", moduleName))
			} else {
				valmap["updateURL"] = val
			}
			if addon.(map[string]interface{})["installDate"] != nil {
				val := addon.(map[string]interface{})["installDate"].(float64)
				valmap["installDate"] = time.Unix(int64(val)/1000, 0).UTC().Format(time.RFC3339)
			}

			if addon.(map[string]interface{})["updateDate"] != nil {
				val := addon.(map[string]interface{})["updateDate"].(float64)
				valmap["updateDate"] = time.Unix(int64(val)/1000, 0).UTC().Format(time.RFC3339)
			}
			if val, err := util.InterfaceToString(addon.(map[string]interface{})["sourceURI"]); err != nil {
				valmap["sourceURI"] = "ERR"
				zap.L().Debug(err.Error(), zap.String("module", moduleName))
			} else {
				valmap["sourceURI"] = val
			}
			// Convert valmap to entry and append to values
			entry, err := util.EntryFromMap(valmap, extensionHeader)
			if err != nil {
				zap.L().Debug("Error formatting valmap as entry: "+err.Error(), zap.String("module", moduleName))
				continue
			}
			values = append(values, entry)
			count++
		}
	}

	zap.L().Debug(fmt.Sprintf("Parsed [%d] firefox extensions entries", count), zap.String("module", moduleName))

	return values, nil
}
