// +build darwin

package macusers

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"reflect"
	"strings"

	"github.com/tonythetiger06/Orion/datawriter"
	"github.com/tonythetiger06/Orion/instance"
	"github.com/tonythetiger06/Orion/util"
	"github.com/tonythetiger06/Orion/util/machelpers"
	"go.uber.org/zap"
)

type MacUsersModule struct{}

var (
	moduleName  = "MacUsersModule"
	mode        = "mac"
	version     = "1.0"
	description = `
	enumerate both deleted and current user profiles on
	the system. This module will also determine the last logged in user,
	and identify administrative users
	`
	author = "Anthony Martinez, martinez.anthonyb@gmail.com"
)

var (
	header = []string{
		"mtime",
		"atime",
		"ctime",
		"btime",
		"date_deleted",
		"unique_id",
		"user",
		"real_name",
		"admin",
		"last_logged_in_user",
	}
	filepathsDeletedUsersPlist = []string{
		"Library/Preferences/com.apple.preferences.accounts.plist",
	}
	filepathAdminUsersPlist = "private/var/db/dslocal/nodes/Default/groups/admin.plist"

	filepathLoginWindowPlist = "Library/Preferences/com.apple.loginwindow.plist"

	filepathsLiveUsers = []string{
		"Users/*",
	}
	filepathsPrivateUsers = []string{
		"private/var/*",
	}
)

func (m MacUsersModule) Start(inst instance.Instance) error {
	err := m.users(inst)
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
	}
	return err
}

func (m MacUsersModule) users(inst instance.Instance) error {
	mw, err := datawriter.NewOrionWriter(moduleName, inst.GetOrionRuntime(), inst.GetOrionOutputFormat(), inst.GetOrionOutputFilepath())
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
		return err
	}

	forensicMode, err := inst.GetOrionConfig().IsForensicMode()
	if err != nil {
		return err
	}

	values := [][]string{}

	// Start Parsing
	// Parse the com.apple.preferences.accounts.plist to identify deleted accounts
	vals, err := m.deletedUsers(inst)
	if err != nil {
		if strings.HasSuffix(err.Error(), "were found") {
			zap.L().Debug("When parsing deleted users - "+err.Error(), zap.String("module", moduleName))
		} else {
			zap.L().Error("Error parsing deleted users: "+err.Error(), zap.String("module", moduleName))
		}
	} else {
		values = util.AppendToDoubleSlice(values, vals)
	}

	// Try to determine admin users on the system
	admins := []string{}
	// try via plist first
	adminData, err := machelpers.DecodePlist(filepathAdminUsersPlist, inst.GetTargetPath())
	if err != nil {
		zap.L().Debug(fmt.Sprintf("Could not retreive admin users via plist parsing: %s", err.Error()), zap.String("module", moduleName))

		// use DSCL to obtain admin users if we are NOT in forensic mode

		if !forensicMode {
			zap.L().Debug(fmt.Sprintf("Resorting to DSCL to obtain admin users from live system"), zap.String("module", moduleName))
			adminsDataFromDSCL, err := m.adminsFromDSCL()
			if err != nil {
				zap.L().Error(fmt.Sprintf("Could not retrieve admin users via DSCL: %s", err.Error()), zap.String("module", moduleName))
			} else {
				admins = append(admins, adminsDataFromDSCL...)
				zap.L().Debug(fmt.Sprintf("found the following admins: %s", admins), zap.String("module", moduleName))
			}
		} else {
			zap.L().Error(fmt.Sprintf("Could not retrieve admin users from mounted volume"), zap.String("module", moduleName))
		}
	} else {
		// Parse admin users from plist data
		for _, item := range adminData {
			machelpers.PrintPlistAsJSON(item)
			// panic("EXIT admins from plist")
			zap.L().Error(fmt.Sprintf("Unimplemented parsing of '%s': '%s'", filepathAdminUsersPlist, item), zap.String("module", moduleName))
		}
	}

	// Enumerate users still active on disk in /Users and /private/var off live/dead disks
	notUsers := []string{".localized", "Shared", "agentx", "at", "audit", "backups", "db", "empty",
		"folders", "install", "jabberd", "lib", "log", "mail", "msgs", "netboot",
		"networkd", "rpc", "run", "rwho", "spool", "tmp", "vm", "yp", "ma"}
	liveUsers := util.Multiglob(filepathsLiveUsers, inst.GetTargetPath())
	if len(liveUsers) == 0 {
		zap.L().Debug("No live users were found", zap.String("module", moduleName))
	}
	privateUsers := util.Multiglob(filepathsPrivateUsers, inst.GetTargetPath())
	if len(privateUsers) == 0 {
		zap.L().Debug("No private users were found", zap.String("module", moduleName))
	}
	liveUsers = append(liveUsers, privateUsers...)
	possibleUsers := []string{}
	for _, usr := range liveUsers {
		if !util.SliceContainsString(notUsers, util.GetUsernameFromPath(usr)) {
			possibleUsers = append(possibleUsers, usr)
		}
	}
	zap.L().Debug(fmt.Sprintf("Found the following users: %s", possibleUsers), zap.String("module", moduleName))

	// Enumerate all user plists in from either /private/var/db/dslocal/nodes or via dscl command
	// u1 : kval : vval
	usersMap := make(map[string]map[string]string)
	userPlists := util.Multiglob([]string{"/private/var/db/dslocal/nodes/Default/users/*"}, inst.GetTargetPath())
	if len(userPlists) == 0 {
		zap.L().Debug("No user plists were found", zap.String("module", moduleName))
	} else {
		for _, userPlist := range userPlists {
			userPlistData, err := machelpers.DecodePlist(userPlist, inst.GetTargetPath())
			if err != nil {
				zap.L().Error(err.Error(), zap.String("module", moduleName))
			}

			machelpers.PrintPlistAsJSON(userPlistData)
			fmt.Println(reflect.TypeOf(userPlistData))

			zap.L().Error(fmt.Sprintf("Unimplemented parsing of '%s': '%s'", userPlist, userPlistData), zap.String("module", moduleName))
		}
	}

	// For live systems Mojave and above, use dscl to get the same dict
	var onlyUserDirectories bool
	onlyUserDirectories = false
	if !forensicMode && len(usersMap) == 0 {
		usersMap, err = m.usersFromDSCL()
		if err != nil {
			zap.L().Error(fmt.Sprintf("Users from dscl live - %s", err.Error(), zap.String("module", moduleName)))
		}
	} else if forensicMode && len(usersMap) == 0 {
		// If running in forensic mode and there was still an error accessing dslocal, operate only with the paths for each user
		onlyUserDirectories = true
		fmt.Println("onlyUserDirectories", onlyUserDirectories)
	}

	// Get last logged in user on system
	var lastUser string
	loginWindowPlistData, err := machelpers.DecodePlist(filepathLoginWindowPlist, inst.GetTargetPath())
	if err != nil {
		zap.L().Debug(fmt.Sprintf("Could not determine last user - login window plist error: %s", err.Error()), zap.String("module", moduleName))
	} else {
		if _, ok := loginWindowPlistData[0]["lastUserName"]; ok {
			lastUser = fmt.Sprint(loginWindowPlistData[0]["lastUserName"])
			zap.L().Debug(fmt.Sprintf("Got last user '%s' from '%s'", lastUser, filepathLoginWindowPlist), zap.String("module", moduleName))
		} else {
			lastUser = ""
			zap.L().Debug(fmt.Sprintf("Could not determine last user from '%s'", filepathLoginWindowPlist), zap.String("module", moduleName))
		}
	}

	// Iterate through all users identified with folders on disk and output their records
	for _, user := range possibleUsers {
		var username string
		if _, ok := usersMap[util.GetUsernameFromPath(user)]; ok {
			username = util.GetUsernameFromPath(user)
		} else {
			continue
		}

		valmap := make(map[string]string, len(header))
		util.InitializeMapToEmptyString(valmap, header)
		valmap["user"] = strings.TrimSpace(username)

		// get timestamps
		for k, v := range machelpers.FileTimestamps(user, moduleName) {
			valmap[k] = v
		}

		if val, ok := usersMap[username]["uid"]; ok {
			valmap["unique_id"] = strings.TrimSpace(val)
		}
		if val, ok := usersMap[username]["real_name"]; ok {
			valmap["real_name"] = strings.TrimSpace(val)
		}
		if util.SliceContainsString(admins, username) {
			valmap["admin"] = "true"
		}
		if username == lastUser {
			valmap["last_logged_in_user"] = "true"
		} else {
			valmap["last_logged_in_user"] = "false"
		}
		delete(usersMap, username)

		// write entry
		vals, err := util.EntryFromMap(valmap, header)
		if err != nil {
			zap.L().Error(err.Error(), zap.String("module", moduleName))
		} else {
			values = append(values, vals)
		}
	}

	if onlyUserDirectories {
		zap.L().Debug("Iterating only through /User directories on disk for metadata due to dslocal errors in forensic mode", zap.String("module", moduleName))
		for _, user := range possibleUsers {
			var username string
			if _, ok := usersMap[util.GetUsernameFromPath(user)]; ok {
				username = util.GetUsernameFromPath(user)
			} else {
				continue
			}

			valmap := make(map[string]string, len(header))
			util.InitializeMapToEmptyString(valmap, header)
			valmap["user"] = strings.TrimSpace(username)

			// get timestamps
			for k, v := range machelpers.FileTimestamps(user, moduleName) {
				valmap[k] = v
			}
			delete(usersMap, username)

			// write entry
			vals, err := util.EntryFromMap(valmap, header)
			if err != nil {
				zap.L().Error(err.Error(), zap.String("module", moduleName))
			} else {
				values = append(values, vals)
			}
		}
	}

	// Iterate through any remaining users identified with DSCL or the plist files that did not have directories found on disk
	zap.L().Debug("Iterating through users with no directories on disk", zap.String("module", moduleName))
	for user, userData := range usersMap {
		var username string
		if _, ok := usersMap[util.GetUsernameFromPath(user)]; ok {
			username = util.GetUsernameFromPath(user)
		} else {
			continue
		}

		valmap := make(map[string]string, len(header))
		util.InitializeMapToEmptyString(valmap, header)
		valmap["user"] = strings.TrimSpace(username)
		valmap["unique_id"] = strings.TrimSpace(userData["uid"])
		valmap["real_name"] = strings.TrimSpace(userData["real_name"])

		// write entry
		vals, err := util.EntryFromMap(valmap, header)
		if err != nil {
			zap.L().Error(err.Error(), zap.String("module", moduleName))
		} else {
			values = append(values, vals)
		}
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

func (m MacUsersModule) usersFromDSCL() (map[string]map[string]string, error) {
	// UniqueID
	userIDsCmd := exec.Command("dscl", ".", "-list", "Users", "UniqueID")
	userIDsOut, outerr := userIDsCmd.StdoutPipe()
	userIDsErr, errerr := userIDsCmd.StderrPipe()
	if outerr != nil {
		return map[string]map[string]string{}, errors.New("userIDsOut error - could not parse userIDs via dscl: " + outerr.Error())
	}
	if errerr != nil {
		return map[string]map[string]string{}, errors.New("userIDsErr error - could not parse userIDs via dscl: " + errerr.Error())
	}
	userIDsCmd.Start()
	userIDsOutBytes, outbyteserr := ioutil.ReadAll(userIDsOut)
	userIDsErrorBytes, errbyteserr := ioutil.ReadAll(userIDsErr)
	if outbyteserr != nil {
		return map[string]map[string]string{}, errors.New("userIDsOutBytes error - could not parse userIDs via dscl: " + outbyteserr.Error())
	}
	if errbyteserr != nil {
		return map[string]map[string]string{}, errors.New("userIDsErrorBytes error - could not parse userIDs via dscl: " + errbyteserr.Error())
	}
	waiterr := userIDsCmd.Wait()
	if waiterr != nil {
		return map[string]map[string]string{}, errors.New("userIDsCmd wait error - could not parse userIDs via dscl: " + waiterr.Error())
	}

	userIDsOutString := string(userIDsOutBytes)
	userIDsErrorString := string(userIDsErrorBytes)

	if len(userIDsErrorString) > 0 {
		if !strings.Contains(userIDsErrorString, "NOTE:") {
			return map[string]map[string]string{}, errors.New(fmt.Sprintf("userIDsErrorString not empty - could not parse: %s", userIDsErrorString))
		}
	}

	usersMap := make(map[string]map[string]string)
	for _, userID := range strings.Split(userIDsOutString, "\n") {
		userIDData := strings.Split(userID, " ")
		usersMap[userIDData[0]] = map[string]string{
			"uid":       userIDData[len(userIDData)-1],
			"real_name": "",
		}
	}

	// RealName
	userNamesCmd := exec.Command("dscl", ".", "-list", "Users", "UniqueID")
	userNamesOut, outerr := userNamesCmd.StdoutPipe()
	userNamesErr, errerr := userNamesCmd.StderrPipe()
	if outerr != nil {
		return map[string]map[string]string{}, errors.New("userNamesOut error - could not parse userNames via dscl: " + outerr.Error())
	}
	if errerr != nil {
		return map[string]map[string]string{}, errors.New("userNamesErr error - could not parse userNames via dscl: " + errerr.Error())
	}
	userNamesCmd.Start()
	userNamesOutBytes, outbyteserr := ioutil.ReadAll(userNamesOut)
	userNamesErrorBytes, errbyteserr := ioutil.ReadAll(userNamesErr)
	if outbyteserr != nil {
		return map[string]map[string]string{}, errors.New("userNamesOutBytes error - could not parse userNames via dscl: " + outbyteserr.Error())
	}
	if errbyteserr != nil {
		return map[string]map[string]string{}, errors.New("userNamesErrorBytes error - could not parse userNames via dscl: " + errbyteserr.Error())
	}
	waiterr = userNamesCmd.Wait()
	if waiterr != nil {
		return map[string]map[string]string{}, errors.New("userNamesCmd wait error - could not parse userNames via dscl: " + waiterr.Error())
	}

	userNamesOutString := string(userNamesOutBytes)
	userNamesErrorString := string(userNamesErrorBytes)

	if len(userNamesErrorString) > 0 {
		if !strings.Contains(userNamesErrorString, "NOTE:") {
			return map[string]map[string]string{}, errors.New(fmt.Sprintf("userNamesErrorString not empty - could not parse: %s", userNamesErrorString))
		}
	}

	for _, userID := range strings.Split(userNamesOutString, "\n") {
		userNameData := strings.Split(userID, " ")
		usersMap[userNameData[0]]["real_name"] = strings.Join(userNameData[1:], " ")
	}

	return usersMap, nil
}

// adminsFromDSCL uses exce to run the DSCL command to retrieve a slice of admin users as strings
func (m MacUsersModule) adminsFromDSCL() ([]string, error) {
	adminsCmd := exec.Command("dscl", ".", "-read", "/Groups/admin", "GroupMembership")
	adminsOut, outerr := adminsCmd.StdoutPipe()
	adminsErr, errerr := adminsCmd.StderrPipe()
	if outerr != nil {
		return []string{}, errors.New("adminsOut error - could not parse admins via dscl: " + outerr.Error())
	}
	if errerr != nil {
		return []string{}, errors.New("adminsErr error - could not parse admins via dscl: " + errerr.Error())
	}
	adminsCmd.Start()
	adminsOutBytes, outbyteserr := ioutil.ReadAll(adminsOut)
	adminsErrorBytes, errbyteserr := ioutil.ReadAll(adminsErr)
	if outbyteserr != nil {
		return []string{}, errors.New("adminsOutBytes error - could not parse admins via dscl: " + outbyteserr.Error())
	}
	if errbyteserr != nil {
		return []string{}, errors.New("adminsErrorBytes error - could not parse admins via dscl: " + errbyteserr.Error())
	}
	waiterr := adminsCmd.Wait()
	if waiterr != nil {
		return []string{}, errors.New("adminsCmd wait error - could not parse admins via dscl: " + waiterr.Error())
	}

	adminsOutString := string(adminsOutBytes)
	adminsErrorString := string(adminsErrorBytes)

	if len(adminsErrorString) > 0 {
		if !strings.Contains(adminsErrorString, "NOTE:") {
			return []string{}, errors.New(fmt.Sprintf("adminsErrorString not empty - could not parse: %s", adminsErrorString))
		}
	}

	entries := strings.Split(strings.Split(strings.TrimSpace(adminsOutString), ":")[1], " ")
	return entries, nil
}

func (m MacUsersModule) deletedUsers(inst instance.Instance) ([][]string, error) {
	values := [][]string{}
	count := 0

	deletedUsersPaths := util.Multiglob(filepathsDeletedUsersPlist, inst.GetTargetPath())
	if len(deletedUsersPaths) == 0 {
		return [][]string{}, errors.New("no deleted users were found")
	}

	for _, path := range deletedUsersPaths {
		var valmap = make(map[string]string)
		valmap = util.InitializeMapToEmptyString(valmap, header)

		// Parse plist/bplist
		deletedUsersPlistData, err := machelpers.DecodePlist(path, inst.GetTargetPath())
		if err != nil {
			zap.L().Error("could not parse plist '"+path+"': "+err.Error(), zap.String("module", moduleName))
			continue
		}
		for _, item := range deletedUsersPlistData {
			machelpers.PrintPlistAsJSON(item)
			return [][]string{}, errors.New("deleted users - Unimplemented method - go build this feature :) ")
		}
	}
	zap.L().Debug(fmt.Sprintf("Parsed [%d] deleted users entries", count), zap.String("module", moduleName))

	return values, nil
}
