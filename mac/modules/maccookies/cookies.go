package maccookies

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthonybm/Orion/datawriter"
	"github.com/anthonybm/Orion/instance"
	"github.com/anthonybm/Orion/util"
	"go.uber.org/zap"
)

type MacCookiesModule struct {
}

var (
	moduleName  = "MacCookiesModule"
	mode        = "mac"
	version     = "1.0"
	description = `
	Reads and parses the cookies database for each user and browser
	`
	author = "Anthony Martinez, martinez.anthonyb@gmail.com"
)

var (
	filepathChromeCookiesGlob  = []string{"Users/*/Library/Application Support/Google/Chrome"}
	filepathFirefoxCookiesGlob = []string{"Users/*/Library/Application Support/Firefox/Profiles/*.*"}
	filepathTemporary          = "CookiesTemp"
)

func (m MacCookiesModule) Start(inst instance.Instance) error {

	err := m.cookies(inst)
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
	}
	return err
}

func (m MacCookiesModule) cookies(inst instance.Instance) error {
	header := []string{
		"browser",
		"user",
		"profile",
		"host_key",
		"name",
		"value",
		"path",
		"creation_utc",
		"expires_utc",
		"last_access_utc",
		"is_secure",
		"ishttponly",
		"same_site",
		"extra", //JSON Key Val {"key":"val","":""....}
	}
	values := [][]string{}

	mw, err := datawriter.NewOrionWriter(moduleName, inst.GetOrionRuntime(), inst.GetOrionOutputFormat(), inst.GetOrionOutputFilepath())
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
		return err
	}

	// Create temporary directory for files to copy to
	// Ensure directory path exists and if not create it
	if _, err := os.Stat(filepathTemporary); os.IsNotExist(err) {
		errDir := os.MkdirAll(filepathTemporary, 0777)
		if errDir != nil {
			return fmt.Errorf("Failed to create temp directory for files '%s': %s", filepathTemporary, err.Error())
		}
	}

	// Glob chrome cookies
	chromeCookiesFileLocations := util.Multiglob(filepathChromeCookiesGlob, inst.GetTargetPath())

	// Glob firefox cookies
	firefoxCookiesFileLocations := util.Multiglob(filepathFirefoxCookiesGlob, inst.GetTargetPath())

	if len(chromeCookiesFileLocations) == 0 {
		zap.L().Warn("No Chrome cookies files were found!", zap.String("module", moduleName))
	}
	if len(firefoxCookiesFileLocations) == 0 {
		zap.L().Warn("No Firefox cookies files were found!", zap.String("module", moduleName))
	}

	chromeCookiesValues, err := m.chromeCookies(chromeCookiesFileLocations, header)
	if err != nil {
		zap.L().Error("Failed to parse chrome cookies: "+err.Error(), zap.String("module", moduleName))
	}
	values = util.AppendToDoubleSlice(values, chromeCookiesValues)

	firefoxCookiesValues, err := m.firefoxCookies(firefoxCookiesFileLocations, header)
	if err != nil {
		zap.L().Error("Failed to parse chrome cookies: "+err.Error(), zap.String("module", moduleName))
	}
	values = util.AppendToDoubleSlice(values, firefoxCookiesValues)

	// Delete folder temp
	err = os.Remove(filepathTemporary)
	if err != nil {
		zap.L().Error(err.Error(), zap.String("module", moduleName))
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

func (m MacCookiesModule) firefoxCookies(fileLocations []string, header []string) ([][]string, error) {
	// Parse entries from ...

	values := [][]string{}

	for _, fl := range fileLocations {
		username := util.GetUsernameFromPath(fl)
		zap.L().Debug(fmt.Sprintf("Parsing Firefox cookies for %s user", username), zap.String("module", moduleName))

		firefoxCookiesData, err := m.pullFirefoxCookiesDataFromDB(filepath.Join(fl, "cookies.sqlite"), username, fl, header)
		if err != nil {
			zap.L().Debug(fmt.Sprintf("Failed to get Firefox Cookies data for '%s': %s", filepath.Join(fl, "cookies.sqlite"), err.Error()), zap.String("module", moduleName))
			continue
		}
		values = util.AppendToDoubleSlice(values, firefoxCookiesData)
	}

	return values, nil
}

func (m MacCookiesModule) chromeCookies(fileLocations []string, header []string) ([][]string, error) {
	// Generate list of all Chrome profiles under all chrome directories
	locs := []string{
		"Default",
		"Profile *",
		"Guest Profile",
	}
	chromeProfileLocations := []string{}
	for _, fl := range fileLocations {
		for _, loc := range locs {
			globbed, err := filepath.Glob(filepath.Join(fl, loc))
			if err != nil {
				zap.L().Error(fmt.Sprintf("Error globbing for chrome profiles under '%s': %s", filepath.Join(fl, loc), err.Error()), zap.String("module", moduleName))
				continue
			}
			if len(globbed) == 0 {
				zap.L().Debug(fmt.Sprintf("Files not found in: %s", filepath.Join(fl, loc)), zap.String("module", moduleName))
				continue
			}
			chromeProfileLocations = append(chromeProfileLocations, globbed...)
		}
	}
	zap.L().Debug(fmt.Sprintf("Will try to parse Chrome cookies from %d locations", len(chromeProfileLocations)), zap.String("module", moduleName))

	// Read and parse ...
	values := [][]string{}
	for _, profile := range chromeProfileLocations {
		username := util.GetUsernameFromPath(profile)
		zap.L().Debug(fmt.Sprintf("Parsing Chrome cookies for %s user", username), zap.String("module", moduleName))

		// Check if we can access
		// chromeVersion, err := m.getChromeVersion(filepath.Join(profile, "Cookies"))
		// if err != nil {
		// 	zap.L().Debug(fmt.Sprintf("Failed to get Chrome version from '%s': %s", filepath.Join(profile, "Cookies"), err.Error()), zap.String("module", moduleName))
		// 	chromeVersion = "ERROR"
		// 	continue
		// }
		chromeCookiesData, err := m.pullChromeCookiesDataFromDB(filepath.Join(profile, "Cookies"), username, profile, header)
		if err != nil {
			zap.L().Debug(fmt.Sprintf("Failed to get Chrome Cookies data for '%s': %s", filepath.Join(profile, "Cookies"), err.Error()), zap.String("module", moduleName))
			continue
		}
		values = util.AppendToDoubleSlice(values, chromeCookiesData)

	}

	return values, nil
}

func (m MacCookiesModule) pullFirefoxCookiesDataFromDB(firefoxCookiesDBPath string, username string, profile string, header []string) ([][]string, error) {
	values := [][]string{}

	// Copy to Temp
	// Open original
	var firefoxCookiesDBPathTemp string
	firefoxCookiesDBPathTemp = filepathTemporary + "/" + "Firefox-Cookies-tmp"
	err := util.CopyFile(firefoxCookiesDBPath, firefoxCookiesDBPathTemp, os.FileMode(int(0777)))
	if err != nil {
		if !strings.Contains(err.Error(), "no such file or directory") {
			zap.L().Error("Failed to copy firefox cookies to temp: " + err.Error())
		}
		return [][]string{}, err
	}

	// Query for DB
	var q = `
	SELECT 
	host, name, value, path, creationTime, expiry, lastAccessed, isSecure, isHttpOnly, inBrowserElement, sameSite
	FROM moz_cookies`

	// Firefox Cookies DB Headers
	dbheaders := []string{
		"host",
		"name",
		"value",
		"path",
		"creationTime",
		"expiry",
		"lastAccessed",
		"isSecure",
		"isHttpOnly",
		"inBrowserElement",
		"samesite",
	}

	// Query the DB
	parsedEntriesCount := 0
	entries, err := util.QueryDB(firefoxCookiesDBPathTemp, q, dbheaders, false)
	if err != nil {
		return [][]string{}, err
	}
	for _, e := range entries {
		// Create Value Mapping for header/entry writing
		var valmap = make(map[string]string)
		for _, h := range header {
			valmap[h] = ""
		}
		valmap["browser"] = "Firefox"
		valmap["user"] = username
		valmap["profile"] = profile
		valmap["host_key"] = e[0]
		valmap["name"] = e[1]
		valmap["value"] = e[2]
		valmap["path"] = e[3]
		valmap["creation_utc"] = e[4]
		valmap["expires_utc"] = e[5]
		valmap["last_access_utc"] = e[6]
		valmap["is_secure"] = e[7]
		valmap["ishttponly"] = e[8]
		valmap["same_site"] = e[9]
		valmap["extra"] = ""

		// Convert valmap to entry and append to values
		entry, err := util.EntryFromMap(valmap, header)
		if err != nil {
			zap.L().Debug("Error formatting valmap as entry: "+err.Error(), zap.String("module", moduleName))
			continue
		}
		values = append(values, entry)
		parsedEntriesCount++
	}
	zap.L().Debug(fmt.Sprintf("Parsed [%d] entries from '%s'", parsedEntriesCount, firefoxCookiesDBPath), zap.String("module", moduleName))

	// Delete copy from Temp
	err = os.Remove(firefoxCookiesDBPathTemp)
	if err != nil {
		zap.L().Error(err.Error(), zap.String("module", moduleName))
	}

	return values, nil
}

func (m MacCookiesModule) pullChromeCookiesDataFromDB(chromeCookiesDBPath string, username string, profile string, header []string) ([][]string, error) {
	values := [][]string{}

	// Copy to Temp
	// Open original
	var chromeCookiesDBPathTemp string
	chromeCookiesDBPathTemp = filepathTemporary + "/" + "Chrome-Cookies-tmp"
	err := util.CopyFile(chromeCookiesDBPath, chromeCookiesDBPathTemp, os.FileMode(int(0777)))
	if err != nil {
		if !strings.Contains(err.Error(), "no such file or directory") {
			zap.L().Error("Failed to copy chrome cookies to temp: " + err.Error())
		}
		return [][]string{}, err
	}

	// Query for DB
	var q = `
	SELECT
	host_key, name, value, path, creation_utc, expires_utc, last_access_utc, is_secure,
	is_httponly, has_expires, is_persistent, priority, encrypted_value, samesite, source_scheme
	FROM cookies`
	// Chrome Cookies DB Headers
	dbheaders := []string{
		"creation_utc",    // INTEGER
		"host_key",        // TEXT
		"name",            // TEXT
		"value",           // TEXT
		"path",            // TEXT
		"expires_utc",     // INTEGER
		"is_secure",       // INTEGER
		"is_httponly",     // INTEGER
		"last_access_utc", // INTEGER
		"has_expires",     // INTEGER // EXTRA
		"is_persistent",   // INTEGER // EXTRA
		"priority",        // INTEGER // EXTRA
		"encrypted_value", // BLOB // EXTRA
		"samesite",        // INTEGER
		"source_scheme",   // INTEGER // EXTRA
	}

	// Query the DB
	parsedEntriesCount := 0
	entries, err := util.QueryDB(chromeCookiesDBPathTemp, q, dbheaders, false)
	if err != nil {
		return [][]string{}, err
	}
	for _, e := range entries {
		// Create Value Mapping for header/entry writing
		var valmap = make(map[string]string)
		for _, h := range header {
			valmap[h] = ""
		}
		valmap["browser"] = "Chrome"
		valmap["user"] = username
		valmap["profile"] = profile
		valmap["host_key"] = e[1]
		valmap["name"] = e[2]
		valmap["value"] = e[3]
		valmap["path"] = e[4]
		valmap["creation_utc"] = e[0]
		valmap["expires_utc"] = e[5]
		valmap["last_access_utc"] = e[8]
		valmap["is_secure"] = e[6]
		valmap["ishttponly"] = e[7]
		valmap["same_site"] = e[13]
		valmap["extra"] = fmt.Sprintf(`{%s:%s, %s:%s, %s:%s, %s:%s, %s:%s}`,
			"has_expires", e[9],
			"is_persistent", e[10],
			"priority", e[11],
			"encrypted_value", e[12],
			"source_scheme", e[14])

		// Convert valmap to entry and append to values
		entry, err := util.EntryFromMap(valmap, header)
		if err != nil {
			zap.L().Debug("Error formatting valmap as entry: "+err.Error(), zap.String("module", moduleName))
			continue
		}
		values = append(values, entry)
		parsedEntriesCount++
	}
	zap.L().Debug(fmt.Sprintf("Parsed [%d] entries from '%s'", parsedEntriesCount, chromeCookiesDBPath), zap.String("module", moduleName))

	// Delete copy from Temp
	err = os.Remove(chromeCookiesDBPathTemp)
	if err != nil {
		zap.L().Error(err.Error(), zap.String("module", moduleName))
	}

	return values, nil
}
