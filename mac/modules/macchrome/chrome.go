package macchrome

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/anthonybm/Orion/datawriter"
	"github.com/anthonybm/Orion/instance"
	"github.com/anthonybm/Orion/util"
	"go.uber.org/zap"
)

var (
	moduleName  = "MacChromeModule"
	mode        = "mac"
	version     = "1.0"
	description = `
	read and parse the Chrome history database for each user on disk
	`
	author = "Anthony Martinez, martinez.anthonyb@gmail.com"
)

var (
	filepathTemporary           = "ChromeTemp"
	filepathsChromeLocationGlob = []string{"Users/*/Library/Application Support/Google/Chrome/"}
	profileHeader               = []string{
		"user",
		"profile",
		"active_time",
		"is_using_default_avatar",
		"avatar_icon",
		"last_downloaded_gaia_picture_url_with_size",
		"hosted_domain",
		"first_account_name_hash",
		"name",
		"gaia_picture_file_name",
		"user_name",
		"gaia_name",
		"local_auth_credentials",
		"is_consented_primary_account",
		"managed_user_id",
		"gaia_id",
		"background_apps",
		"is_omitted_from_profile_list",
		"gaia_given_name",
		"is_using_default_name",
		"is_ephemeral",
		"metrics_bucket_index",
		"account_categories",
	}
	urlHeader = []string{
		"user",
		"profile",
		"visit_time",
		"title",
		"url",
		"visit_count",
		"last_visit_time",
		"typed_count",
		"visit_duration",
		"search_term",
	}
	downloadHeader = []string{
		"user",
		"profile",
		"download_path",
		"current_path",
		"download_started",
		"download_finished",
		"danger_type",
		"opened",
		"last_modified",
		"referrer",
		"tab_url",
		"tab_referrer_url",
		"download_url",
		"url",
	}
	extensionHeader = []string{
		"user",
		"profile",
		"name",
		"permissions",
		"author",
		"description",
		"scripts",
		"persistent",
		"version",
	}
)

// MacChromeModule wraps the methods for the module to run
type MacChromeModule struct{}

// Start starts the MacChromeModule, should not be manually called
func (m MacChromeModule) Start(inst instance.Instance) error {
	err := m.chrome(inst)
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
	}
	return err
}

func (m MacChromeModule) chrome(inst instance.Instance) error {
	mw, err := datawriter.NewOrionWriter(moduleName, inst.GetOrionRuntime(), inst.GetOrionOutputFormat(), inst.GetOrionOutputFilepath())
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
		return err
	}

	forensicMode, err := inst.GetOrionConfig().IsForensicMode()
	if err != nil {
		return err
	}

	profileValues := [][]string{}
	downloadValues := [][]string{}
	historyValues := [][]string{}
	extensionValues := [][]string{}

	// Start Parsing

	// for all chrome dirs on disk parse their local state files
	chromeLocations := util.Multiglob(filepathsChromeLocationGlob, inst.GetTargetPath())
	if len(chromeLocations) == 0 {
		zap.L().Debug(fmt.Sprintf("No chrome files were found in %s", filepathsChromeLocationGlob), zap.String("module", moduleName))
	}
	for _, chromeLocation := range chromeLocations {
		username := util.GetUsernameFromPath(chromeLocation)
		zap.L().Debug(fmt.Sprintf("Parsing Chrome Local State data under '%s' user", username))

		localStateFilePath := filepath.Join(chromeLocation, "Local State")
		localStateFilePathExists, err := util.Exists(localStateFilePath)
		if err != nil {
			zap.L().Debug(fmt.Sprintf("local state file error - %s", err.Error()), zap.String("module", moduleName))
		}
		if localStateFilePathExists {
			localStateFileContents, err := ioutil.ReadFile(localStateFilePath)
			if err != nil {
				zap.L().Error(fmt.Sprintf("local state file error - %s", err.Error()), zap.String("module", moduleName))
			} else {
				// proceed with parsing from local state file for user
				// TODO: format struct instead of interface unmarshalling
				var localStateFileData interface{}
				err := json.Unmarshal(localStateFileContents, &localStateFileData)
				if err != nil {
					zap.L().Error(fmt.Sprintf("local state file error unmarshalling json- %s", err.Error()), zap.String("module", moduleName))
				} else {
					// proceed with json data from file
					// chrome version
					chromeVersion, err := util.JSONGetValueFromKey(localStateFileData, "stats_version")
					if err != nil {
						zap.L().Error(fmt.Sprintf("reading chrome version - %s", err.Error()), zap.String("module", moduleName))
					} else {
						zap.L().Debug(fmt.Sprintf("read chrome version - %s", chromeVersion), zap.String("module", moduleName))
					}

					// profile data info cache
					profileData, err := util.JSONGetValueFromKey(localStateFileData, "info_cache")
					if err != nil {
						zap.L().Error(fmt.Sprintf("reading chrome profile data info cache - %s", err.Error()), zap.String("module", moduleName))
					} else {
						zap.L().Debug(fmt.Sprintf("read chrome profile data info cache - %s", profileData), zap.String("module", moduleName))
					}

					for k, v := range profileData.(map[string]interface{}) {
						var valmap = make(map[string]string)
						util.InitializeMapToEmptyString(valmap, profileHeader)

						valmap["user"] = username
						valmap["profile"] = k
						for key, val := range v.(map[string]interface{}) {
							if itemexists(profileHeader, key) {
								if strings.Contains(key, "time") {
									valmap[key] = time.Unix(int64(val.(float64)), 0).UTC().String()
								} else {
									t, err := util.InterfaceToString(val)
									valmap[key] = t
									if err != nil {
										zap.L().Error(err.Error(), zap.String("module", moduleName))
									}
								}
								// Convert valmap to entry and append to values
								entry, err := util.EntryFromMap(valmap, profileHeader)
								if err != nil {
									ht := []string{}
									for k := range valmap {
										ht = append(ht, k)
									}
									zap.L().Error(fmt.Sprintf("Error formatting profile valmap as entry: %s - valmap headers: %s - profile headers %s", err.Error(), ht, profileHeader), zap.String("module", moduleName))
									continue
								}
								profileValues = append(profileValues, entry)
							}
						}
					}
				}
			}
		}
	}

	// make a full list of all chrome profiles under all chrome dirs
	chromeProfiles := []string{}
	chromeProfileGlobs := []string{"Default", "Profile *", "Guest Profile"}
	for _, chromeProfileGlob := range chromeProfileGlobs {
		for _, filepathChromeLocationGlob := range filepathsChromeLocationGlob {
			chromeProfilesGlobbed, err := filepath.Glob(filepath.Join(filepath.Join(inst.GetTargetPath(), filepathChromeLocationGlob), chromeProfileGlob))
			if err != nil {
				zap.L().Error(fmt.Sprintf("while globbing chrome profile '%s' from '%s' - %s", chromeProfileGlob, filepathChromeLocationGlob, err.Error()), zap.String("module", moduleName))
			}
			chromeProfiles = append(chromeProfiles, chromeProfilesGlobbed...)
		}
	}

	for _, chromeProfile := range chromeProfiles {
		username := util.GetUsernameFromPath(chromeProfile)
		zap.L().Debug(fmt.Sprintf("Starting parsing for Chrome history under '%s' user", username), zap.String("module", moduleName))

		// parse Chrome history
		parseChromeVisitHistoryValues, err := m.parseChromeVisitHistoryValues(username, chromeProfile, filepath.Join(chromeProfile, "History"), forensicMode)
		if err != nil {
			zap.L().Error(fmt.Sprintf("chrome visit history - %s", err.Error()), zap.String("module", moduleName))
		} else {
			historyValues = append(historyValues, parseChromeVisitHistoryValues...)
		}
		parseChromeDownloadValues, err := m.parseChromeDownloadValues(username, chromeProfile, filepath.Join(chromeProfile, "History"), forensicMode)
		if err != nil {
			zap.L().Error(fmt.Sprintf("chrome download history - %s", err.Error()), zap.String("module", moduleName))
		} else {
			downloadValues = append(downloadValues, parseChromeDownloadValues...)
		}

		// parse Chrome extensions
		chromeExtensions, err := filepath.Glob(filepath.Join(chromeProfile, "Extensions/*/*/manifest.json"))
		if err != nil {
			zap.L().Error(fmt.Sprintf("chrome extensions glob error - %s", err.Error()), zap.String("module", moduleName))
		} else {
			if len(chromeExtensions) == 0 {
				zap.L().Debug(fmt.Sprintf("No chrome extension files were found in %s", filepath.Join(chromeProfile, "Extensions/*/*/Manifest.json")), zap.String("module", moduleName))
			} else {
				parseChromeExtensionsValues, err := m.parseChromeExtensionsValues(chromeExtensions, username, chromeProfile, forensicMode)
				if err != nil {
					zap.L().Error(fmt.Sprintf("chrome extensions err - %s", err.Error()), zap.String("module", moduleName))
				} else {
					extensionValues = append(extensionValues, parseChromeExtensionsValues...)
				}
			}
		}

	}

	// End Parsing

	// Write to output

	// chrome profile output
	profileOrionWriter, err := datawriter.NewOrionWriter(moduleName+"-profiles", mw.GetOrionRuntime(), mw.GetOutputType(), filepath.Dir(mw.GetOutfilePath()))
	if err != nil {
		zap.L().Error(err.Error(), zap.String("module", moduleName))
	} else {
		err := profileOrionWriter.WriteOutput(profileHeader, profileValues)
		if err != nil {
			zap.L().Error(fmt.Sprintf("while writing profile output - %s", err.Error()), zap.String("module", moduleName))
		}
	}
	// chrome download output
	downloadOrionWriter, err := datawriter.NewOrionWriter(moduleName+"-downloads", mw.GetOrionRuntime(), mw.GetOutputType(), filepath.Dir(mw.GetOutfilePath()))
	if err != nil {
		zap.L().Error(err.Error(), zap.String("module", moduleName))
	} else {
		err := downloadOrionWriter.WriteOutput(downloadHeader, downloadValues)
		if err != nil {
			zap.L().Error(fmt.Sprintf("while writing download output - %s", err.Error()), zap.String("module", moduleName))
		}
	}
	// chrome history output
	historyOrionWriter, err := datawriter.NewOrionWriter(moduleName+"-history", mw.GetOrionRuntime(), mw.GetOutputType(), filepath.Dir(mw.GetOutfilePath()))
	if err != nil {
		zap.L().Error(err.Error(), zap.String("module", moduleName))
	} else {
		err := historyOrionWriter.WriteOutput(urlHeader, historyValues)
		if err != nil {
			zap.L().Error(fmt.Sprintf("while writing history output - %s", err.Error()), zap.String("module", moduleName))
		}
	}
	// chrome extensions output
	extensionsOrionWriter, err := datawriter.NewOrionWriter(moduleName+"-extensions", mw.GetOrionRuntime(), mw.GetOutputType(), filepath.Dir(mw.GetOutfilePath()))
	if err != nil {
		zap.L().Error(err.Error(), zap.String("module", moduleName))
	} else {
		err := extensionsOrionWriter.WriteOutput(extensionHeader, extensionValues)
		if err != nil {
			zap.L().Error(fmt.Sprintf("while writing extension output - %s", err.Error()), zap.String("module", moduleName))
		}
	}

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

func (m MacChromeModule) parseChromeExtensionsValues(extensions []string, user string, profile string, forensicMode bool) ([][]string, error) {
	values := [][]string{}
	count := 0
	for _, extension := range extensions {
		chromeExtensionContents, err := ioutil.ReadFile(extension)
		if err != nil {
			zap.L().Error(fmt.Sprintf("chrome extension file error - %s", err.Error()), zap.String("module", moduleName))
		} else {
			var valmap = make(map[string]string)
			util.InitializeMapToEmptyString(valmap, extensionHeader)

			// TODO: format struct instead of interface unmarshalling
			var chromeExtensionData interface{}
			err := json.Unmarshal(chromeExtensionContents, &chromeExtensionData)
			if err != nil {
				zap.L().Error(fmt.Sprintf("chrome extension file error unmarshalling json- %s", err.Error()), zap.String("module", moduleName))
			} else {
				valmap["user"] = user
				valmap["profile"] = profile
				if val, err := util.JSONGetValueFromKey(chromeExtensionData, "name"); err != nil {
					valmap["name"] = "ERROR"
					zap.L().Error(err.Error(), zap.String("module", moduleName))
				} else {
					valmap["name"], err = util.InterfaceToString(val)
					if err != nil {
						zap.L().Error("could not read value 'name' "+err.Error(), zap.String("module", moduleName))
					}
				}
				if val, err := util.JSONGetValueFromKey(chromeExtensionData, "author"); err != nil {
					valmap["author"] = "ERROR"
					zap.L().Error(err.Error(), zap.String("module", moduleName))
				} else {
					valmap["author"], err = util.InterfaceToString(val)
					if err != nil {
						zap.L().Error("could not read value 'author' "+err.Error(), zap.String("module", moduleName))
					}
				}
				if val, err := util.JSONGetValueFromKey(chromeExtensionData, "permissions"); err != nil {
					valmap["permissions"] = "ERROR"
					zap.L().Error(err.Error(), zap.String("module", moduleName))
				} else {
					valmap["permissions"], err = util.InterfaceToString(val)
					if err != nil {
						zap.L().Error("could not read value 'permissions' "+err.Error(), zap.String("module", moduleName))
					}
				}
				if val, err := util.JSONGetValueFromKey(chromeExtensionData, "description"); err != nil {
					valmap["description"] = "ERROR"
					zap.L().Error(err.Error(), zap.String("module", moduleName))
				} else {
					valmap["description"], err = util.InterfaceToString(val)
					if err != nil {
						zap.L().Error("could not read value 'description' "+err.Error(), zap.String("module", moduleName))
					}
				}
				if val, err := util.JSONGetValueFromKey(chromeExtensionData, "scripts"); err != nil {
					valmap["scripts"] = "ERROR"
					zap.L().Error(err.Error(), zap.String("module", moduleName))
				} else {
					valmap["scripts"], err = util.InterfaceToString(val)
					if err != nil {
						zap.L().Error("could not read value 'scripts' "+err.Error(), zap.String("module", moduleName))
					}
				}
				if val, err := util.JSONGetValueFromKey(chromeExtensionData, "persistent"); err != nil {
					valmap["persistent"] = "ERROR"
					zap.L().Error(err.Error(), zap.String("module", moduleName))
				} else {
					valmap["persistent"], err = util.InterfaceToString(val)
					if err != nil {
						zap.L().Error("could not read value 'persistent' "+err.Error(), zap.String("module", moduleName))
					}
				}
				if val, err := util.JSONGetValueFromKey(chromeExtensionData, "version"); err != nil {
					valmap["version"] = "ERROR"
					zap.L().Error(err.Error(), zap.String("module", moduleName))
				} else {
					valmap["version"], err = util.InterfaceToString(val)
					if err != nil {
						zap.L().Error("could not read value 'version' "+err.Error(), zap.String("module", moduleName))
					}
				}

				// Convert valmap to entry and append to values
				entry, err := util.EntryFromMap(valmap, extensionHeader)
				if err != nil {
					zap.L().Debug("Error formatting extension valmap as entry: "+err.Error(), zap.String("module", moduleName))
					continue
				}
				values = append(values, entry)
				count++
			}
		}
	}
	zap.L().Debug(fmt.Sprintf("Parsed [%d] chrome extensions entries", count), zap.String("module", moduleName))
	return values, nil
}

func (m MacChromeModule) parseChromeVisitHistoryValues(user string, profile string, dbfilepath string, forensicMode bool) ([][]string, error) {
	// Copy to Temp
	// Open original
	var chromeHistoryDBPathTemp string
	chromeHistoryDBPathTemp = filepathTemporary + "/" + "Chrome-History-tmp"
	err := util.CopyFile(dbfilepath, chromeHistoryDBPathTemp, os.FileMode(int(0777)))
	if err != nil {
		zap.L().Error("Failed to copy Chrome History to temp: " + err.Error())
		return [][]string{}, err
	}

	values := [][]string{}
	query := `
	SELECT visit_time, urls.url, title, visit_duration, visit_count, typed_count, urls.last_visit_time, COALESCE(term, '') as term
	FROM visits  left join urls on visits.url = urls.id
                     left join keyword_search_terms on keyword_search_terms.url_id = urls.id
	`
	queryHeaders := []string{
		"visit_time",
		"url",
		"title",
		"visit_duration",
		"visit_count",
		"typed_count",
		"last_visit_time",
		"term",
	}
	// run query
	entries, err := util.QueryDB(chromeHistoryDBPathTemp, query, queryHeaders, forensicMode)
	if err != nil {
		return [][]string{}, err
	}

	// parse entries
	count := 0
	for _, e := range entries {
		var valmap = make(map[string]string)
		util.InitializeMapToEmptyString(valmap, urlHeader)

		valmap["user"] = user
		valmap["profile"] = profile
		valmap["visit_time"], err = util.ChromeTime(e[0])
		if err != nil {
			zap.L().Error(fmt.Sprintf("error parsing visit_time time - %s", err.Error()), zap.String("module", moduleName))
		}
		valmap["url"] = e[1]
		valmap["title"] = e[2]
		visitDurationInt, err := strconv.ParseInt(e[3], 10, 64)
		if err != nil {
			zap.L().Error(fmt.Sprintf("error parsing visit_duration time - %s", err.Error()), zap.String("module", moduleName))
		}
		valmap["visit_duration"] = time.Duration(visitDurationInt * 1000).String()
		valmap["visit_count"] = e[4]
		valmap["typed_count"] = e[5]
		valmap["last_visit_time"], err = util.ChromeTime(e[6])
		if err != nil {
			zap.L().Error(fmt.Sprintf("error parsing visit_time time - %s", err.Error()), zap.String("module", moduleName))
		}
		valmap["search_term"] = e[7]

		// Convert valmap to entry and append to values
		entry, err := util.EntryFromMap(valmap, urlHeader)
		if err != nil {
			zap.L().Debug("Error formatting url valmap as entry: "+err.Error(), zap.String("module", moduleName))
			continue
		}
		values = append(values, entry)
		count++
	}
	zap.L().Debug(fmt.Sprintf("Parsed [%d] entries from '%s'", count, dbfilepath), zap.String("module", moduleName))
	// Delete copy from Temp
	err = os.Remove(chromeHistoryDBPathTemp)
	if err != nil {
		zap.L().Error(err.Error(), zap.String("module", moduleName))
	}

	return values, nil
}

func (m MacChromeModule) parseChromeDownloadValues(user string, profile string, dbfilepath string, forensicMode bool) ([][]string, error) {
	// Copy to Temp
	// Open original
	var chromeHistoryDBPathTemp string
	chromeHistoryDBPathTemp = filepathTemporary + "/" + "Chrome-History-tmp"
	err := util.CopyFile(dbfilepath, chromeHistoryDBPathTemp, os.FileMode(int(0777)))
	if err != nil {
		zap.L().Error("Failed to copy Chrome History to temp: " + err.Error())
		return [][]string{}, err
	}

	values := [][]string{}

	// connect to database

	// query and queryHeaders
	query := `
	SELECT
	current_path, target_path, start_time, end_time, danger_type, opened, last_modified, referrer, tab_url, tab_referrer_url, site_url, url
	FROM downloads left join downloads_url_chains on downloads_url_chains.id = downloads.id
	`
	queryHeaders := []string{
		"current_path",
		"target_path",
		"start_time",
		"end_time",
		"danger_type",
		"opened",
		"last_modified",
		"referrer",
		"tab_url",
		"tab_referrer_url",
		"site_url",
		"url",
	}

	// run query
	entries, err := util.QueryDB(chromeHistoryDBPathTemp, query, queryHeaders, forensicMode)
	if err != nil {
		return [][]string{}, err
	}

	// parse entries
	count := 0
	for _, e := range entries {
		var valmap = make(map[string]string)
		util.InitializeMapToEmptyString(valmap, downloadHeader)

		valmap["user"] = user
		valmap["profile"] = profile
		valmap["current_path"] = e[0]
		valmap["download_path"] = e[1]
		valmap["download_started"], err = util.ChromeTime(e[2])
		if err != nil {
			zap.L().Error(fmt.Sprintf("error parsing download_started time - %s", err.Error()), zap.String("module", moduleName))
		}
		valmap["download_finished"], err = util.ChromeTime(e[3])
		if err != nil {
			zap.L().Error(fmt.Sprintf("error parsing download_finished time - %s", err.Error()), zap.String("module", moduleName))
		}
		valmap["danger_type"] = e[4]
		valmap["opened"] = e[5]

		if e[6] != "" {
			val, err := time.Parse(time.RFC1123, e[6])
			if err != nil {
				zap.L().Error(fmt.Sprintf("Error converting %s to string - %s", e[6], err.Error()), zap.String("module", moduleName))
			} else {
				valmap["last_modified"] = val.UTC().Format(time.RFC3339)
			}
		}

		valmap["referrer"] = e[7]
		valmap["tab_url"] = e[8]
		valmap["tab_referrer_url"] = e[9]
		valmap["download_url"] = e[10]
		valmap["url"] = e[11]

		// Convert valmap to entry and append to values
		entry, err := util.EntryFromMap(valmap, downloadHeader)
		if err != nil {
			zap.L().Debug("Error formatting download valmap as entry: "+err.Error(), zap.String("module", moduleName))
			continue
		}
		values = append(values, entry)
		count++
	}
	zap.L().Debug(fmt.Sprintf("Parsed [%d] entries from '%s'", count, dbfilepath), zap.String("module", moduleName))
	// Delete copy from Temp
	err = os.Remove(chromeHistoryDBPathTemp)
	if err != nil {
		zap.L().Error(err.Error(), zap.String("module", moduleName))
	}

	return values, nil
}

func keyexists(m map[string]interface{}, key string) bool {
	if _, ok := m[key]; ok {
		return true
	}
	return false
}

func itemexists(slice []string, key string) bool {
	for _, s := range slice {
		if s == key {
			return true
		}
	}
	return false
}
