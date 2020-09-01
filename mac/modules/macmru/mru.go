// +build darwin

package macmru

//#cgo CFLAGS: -x objective-c
//#cgo LDFLAGS: -framework Foundation
//#include "foundation.h"
import "C"
import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unsafe"

	"github.com/anthonybm/Orion/datawriter"
	"github.com/anthonybm/Orion/instance"
	"github.com/anthonybm/Orion/util"
	"github.com/anthonybm/Orion/util/machelpers"
	"go.uber.org/zap"
)

type MacMRUModule struct {
}

var (
	moduleName  = "MacMRUModule"
	mode        = "mac"
	version     = "1.0"
	description = `
	Reads and parses the SFL, SFL2, and other various MRU plist files.
	Inspiration taken from Sarah Edwards, AutoMactc by CrowdStrike, and others.
	`
	author = "Anthony Martinez, martinez.anthonyb@gmail.com"
)

var (
	header = []string{
		"source_file",
		"user",
		"source_name",
		"item_index",
		"order",
		"name",
		"url",
		"source_key",
		"extras",
	}
	filepathsSidebarPlists = []string{
		"Users/*/Library/Preferences/com.apple.sidebarlists.plist",
		"private/var/*/Library/Preferences/com.apple.sidebarlists.plist",
	}
	filepathsFinderPlists = []string{
		"Users/*/Library/Preferences/com.apple.finder.plist",
		"private/var/*/Library/Preferences/com.apple.finder.plist",
	}
	filepathsSecureBookmarks = []string{
		"Users/*/Library/Containers/*/Data/Library/Preferences/*.securebookmarks.plist",
		"private/var/*/Library/Containers/*/Data/Library/Preferences/*.securebookmarks.plist",
	}
	filepathsSFLs = []string{
		"Users/*/Library/Application Support/com.apple.sharedfilelist/*.sfl",
		"Users/*/Library/Application Support/com.apple.sharedfilelist/*/*.sfl",
		"private/var/*/Library/Application Support/com.apple.sharedfilelist/*/*.sfl",
	}
	filepathsSFL2s = []string{
		"Users/*/Library/Application Support/com.apple.sharedfilelist/*.sfl2",
		"Users/*/Library/Application Support/com.apple.sharedfilelist/*/*.sfl2",
		"private/var/*/Library/Application Support/com.apple.sharedfilelist/*/*.sfl2",
	}
)

func (m MacMRUModule) Start(inst instance.Instance) error {
	err := m.mru(inst)
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
	}
	return err
}

func (m MacMRUModule) mru(inst instance.Instance) error {
	mw, err := datawriter.NewOrionWriter(moduleName, inst.GetOrionRuntime(), inst.GetOrionOutputFormat(), inst.GetOrionOutputFilepath())
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
		return err
	}

	values := [][]string{}

	// Start Parsing
	vals, err := m.sfl(inst)
	if err != nil {
		if strings.HasSuffix(err.Error(), " were found") {
			zap.L().Warn("Error parsing - "+err.Error(), zap.String("module", moduleName))
		} else {
			zap.L().Error("Error parsing SFLS: "+err.Error(), zap.String("module", moduleName))
		}
	} else {
		values = util.AppendToDoubleSlice(values, vals)
	}

	vals, err = m.sfl2(inst)
	if err != nil {
		if strings.HasSuffix(err.Error(), " were found") {
			zap.L().Warn("Error parsing - "+err.Error(), zap.String("module", moduleName))
		} else {
			zap.L().Error("Error parsing SFLS2: "+err.Error(), zap.String("module", moduleName))
		}
	} else {
		values = util.AppendToDoubleSlice(values, vals)
	}

	vals, err = m.secureBookmarks(inst)
	if err != nil {
		if strings.HasSuffix(err.Error(), " were found") {
			zap.L().Warn("Error parsing - "+err.Error(), zap.String("module", moduleName))
		} else {
			zap.L().Error("Error parsing secureBookmarks: "+err.Error(), zap.String("module", moduleName))
		}
	} else {
		values = util.AppendToDoubleSlice(values, vals)
	}

	vals, err = m.sidebarPlists(inst)
	if err != nil {
		if strings.HasSuffix(err.Error(), " were found") {
			zap.L().Warn("Error parsing - "+err.Error(), zap.String("module", moduleName))
		} else {
			zap.L().Error("Error parsing sidebarPlists: "+err.Error(), zap.String("module", moduleName))
		}
	} else {
		values = util.AppendToDoubleSlice(values, vals)
	}

	vals, err = m.finderPlists(inst)
	if err != nil {
		if strings.HasSuffix(err.Error(), "files were found") {
			zap.L().Warn("Error parsing - "+err.Error(), zap.String("module", moduleName))
		} else {
			zap.L().Error("Error parsing finderPlists: "+err.Error(), zap.String("module", moduleName))
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

func (m MacMRUModule) SFLs(inst instance.Instance) ([][]string, error) {
	values := [][]string{}
	count := 0

	SFLPaths := util.Multiglob(filepathsSFLs, inst.GetTargetPath())
	if len(SFLPaths) == 0 {
		return [][]string{}, errors.New("no SFL files were found")
	}

	for _, path := range SFLPaths {
		var valmap = make(map[string]string)
		valmap = util.InitializeMapToEmptyString(valmap, header)

		valmap["user"] = util.GetUsernameFromPath(path)

		// Parse plist/bplist
		data, err := machelpers.DecodePlist(path, inst.GetTargetPath())
		if err != nil {
			zap.L().Error("could not parse plist '"+path+"': "+err.Error(), zap.String("module", moduleName))
			continue
		}
		for _, item := range data {
			machelpers.PrintPlistAsJSON(item)
			return [][]string{}, errors.New("SFL - Unimplemented method - go build this feature :) ")
		}
	}
	zap.L().Debug(fmt.Sprintf("Parsed [%d] SFL entries", count), zap.String("module", moduleName))

	return values, nil
}

func (m MacMRUModule) SFL2s(inst instance.Instance) ([][]string, error) {
	values := [][]string{}
	count := 0

	SFL2Paths := util.Multiglob(filepathsSFL2s, inst.GetTargetPath())
	if len(SFL2Paths) == 0 {
		return [][]string{}, errors.New("no SFL2 files were found")
	}

	for _, path := range SFL2Paths {
		var valmap = make(map[string]string)
		valmap = util.InitializeMapToEmptyString(valmap, header)

		valmap["user"] = util.GetUsernameFromPath(path)

		// Parse plist/bplist
		data, err := machelpers.DecodePlist(path, inst.GetTargetPath())
		if err != nil {
			zap.L().Error("could not parse plist '"+path+"': "+err.Error(), zap.String("module", moduleName))
			continue
		}
		for _, item := range data {
			machelpers.PrintPlistAsJSON(item)
			return [][]string{}, errors.New("SFL2 - Unimplemented method - go build this feature :) ")
		}
	}
	zap.L().Debug(fmt.Sprintf("Parsed [%d] SFL2 entries", count), zap.String("module", moduleName))

	return values, nil
}

func (m MacMRUModule) sidebarPlists(inst instance.Instance) ([][]string, error) {
	values := [][]string{}
	count := 0

	sidebarPlistPaths := util.Multiglob(filepathsSidebarPlists, inst.GetTargetPath())
	if len(sidebarPlistPaths) == 0 {
		return [][]string{}, errors.New("no Sidebar Plists were found")
	}

	for _, path := range sidebarPlistPaths {
		var valmap = make(map[string]string)
		valmap = util.InitializeMapToEmptyString(valmap, header)

		valmap["user"] = util.GetUsernameFromPath(path)

		// Parse plist/bplist
		data, err := machelpers.DecodePlist(path, inst.GetTargetPath())
		if err != nil {
			zap.L().Error("could not parse plist '"+path+"': "+err.Error(), zap.String("module", moduleName))
			continue
		}
		for _, item := range data {
			machelpers.PrintPlistAsJSON(item)
			return [][]string{}, errors.New("Sidebar Plists - Unimplemented method - go build this feature :) ")
		}
	}
	zap.L().Debug(fmt.Sprintf("Parsed [%d] Sidebar Plist entries", count), zap.String("module", moduleName))

	return values, nil
}

func (m MacMRUModule) finderPlists(inst instance.Instance) ([][]string, error) {
	values := [][]string{}
	count := 0

	finderPlistPaths := util.Multiglob(filepathsFinderPlists, inst.GetTargetPath())
	if len(finderPlistPaths) == 0 {
		return [][]string{}, errors.New("no Finder Plists were found")
	}

	for _, path := range finderPlistPaths {
		var valmap = make(map[string]string)
		valmap = util.InitializeMapToEmptyString(valmap, header)

		valmap["user"] = util.GetUsernameFromPath(path)

		// Parse plist/bplist
		data, err := machelpers.DecodePlist(path, inst.GetTargetPath())
		if err != nil {
			zap.L().Error("could not parse plist '"+path+"': "+err.Error(), zap.String("module", moduleName))
			continue
		}
		for _, item := range data {
			if fxrecentfolders, ok := item["FXRecentFolders"]; ok {
				for _, fxitem := range fxrecentfolders.([]interface{}) {
					// Extract name key from fxitem
					if nameVal, ok := fxitem.(map[string]interface{})["name"]; ok {
						valmap["name"], err = util.InterfaceToString(nameVal)
						if err != nil {
							valmap["name"] = "ERROR"
							zap.L().Error(err.Error())
						}
					} else {
						valmap["name"] = "ERROR"
					}

					// Extract file-bookmark from fxitem using Objective-C Foundation api
					if _, ok := fxitem.(map[string]interface{})["file-bookmark"]; ok {
						for _, item := range NSArrayURLToGoSliceURL(C.FinderFXRecentFolders()) {
							valmap["url"] = item.String()

							valmap["user"] = util.GetUsernameFromPath(path)
							valmap["source_file"] = path
							valmap["source_name"] = "FinderPlist"
							valmap["source_key"] = "FXRecentFolders"
							valmap["extras"] = ""
							entry, err := util.UnsafeEntryFromMap(valmap, header)
							if err != nil {
								zap.L().Error("Failed to convert map to entry: "+err.Error(), zap.String("module", moduleName))
								continue
							}
							values = append(values, entry)
							count++
						}
					}
				}
			}
			if moveandcopy, ok := item["RecentMoveAndCopyDestinations"]; ok {
				for _, moveandcopyitem := range moveandcopy.([]interface{}) {
					// fmt.Println("MoveAndCopyItem: ", moveandcopyitem)
					valmap["url"] = fmt.Sprint(moveandcopyitem)

					valmap["user"] = util.GetUsernameFromPath(path)
					valmap["source_file"] = path
					valmap["source_name"] = "FinderPlist"
					valmap["source_key"] = "RecentMoveAndCopyDestinations"
					valmap["extras"] = ""
					count++
				}
			}
			if _, ok := item["FXDesktopVolumePositions"]; ok {
				zap.L().Warn("FinderPlist - Unimplemented parsing of 'FXDesktopVolumePositions' - go build this feature :) ", zap.String("module", moduleName))
			}
			if _, ok := item["FXConnectToLastURL"]; ok {
				zap.L().Warn("FinderPlist - Unimplemented parsing of 'FXConnectToLastURL' - go build this feature :) ", zap.String("module", moduleName))
			}
			if _, ok := item["NSNavLastRootDirectory"]; ok {
				zap.L().Warn("FinderPlist - Unimplemented parsing of 'NSNavLastRootDirectory' - go build this feature :) ", zap.String("module", moduleName))
			}
			if _, ok := item["NSNavLastCurrentDirectory"]; ok {
				zap.L().Warn("FinderPlist - Unimplemented parsing of 'NSNavLastCurrentDirectory' - go build this feature :) ", zap.String("module", moduleName))
			}
			if _, ok := item["GoToField"]; ok {
				zap.L().Warn("FinderPlist - Unimplemented parsing of 'GoToField' - go build this feature :) ", zap.String("module", moduleName))
			}
			if _, ok := item["BulkRename"]; ok {
				zap.L().Warn("FinderPlist - Unimplemented parsing of 'BulkRename' - go build this feature :) ", zap.String("module", moduleName))
			}
		}
	}
	zap.L().Debug(fmt.Sprintf("Parsed [%d] Finder Plist entries", count), zap.String("module", moduleName))

	return values, nil
}

func (m MacMRUModule) secureBookmarks(inst instance.Instance) ([][]string, error) {
	values := [][]string{}
	count := 0

	secureBookmarkPaths := util.Multiglob(filepathsSecureBookmarks, inst.GetTargetPath())
	if len(secureBookmarkPaths) == 0 {
		return [][]string{}, errors.New("no Secure Bookmarks were found")
	}

	for _, path := range secureBookmarkPaths {
		var valmap = make(map[string]string)
		valmap = util.InitializeMapToEmptyString(valmap, header)

		// Parse plist/bplist
		data, err := machelpers.DecodePlist(path, inst.GetTargetPath())
		if err != nil {
			zap.L().Error("could not parse plist '"+path+"': "+err.Error(), zap.String("module", moduleName))
			continue
		}
		for _, item := range data {
			for k, v := range item {
				valmap["user"] = util.GetUsernameFromPath(path)
				valmap["source_file"] = path
				valmap["source_name"] = "SecureBookmarks"
				valmap["name"] = util.GetUsernameFromPath(k)
				valmap["url"] = fmt.Sprint(k)
				valmap["extras"] = util.MapToJSONString(v.(map[string]interface{}))

				entry, err := util.UnsafeEntryFromMap(valmap, header)
				if err != nil {
					zap.L().Error("Failed to convert map to entry: "+err.Error(), zap.String("module", moduleName))
					continue
				}
				values = append(values, entry)
				count++
			}
		}
	}
	zap.L().Debug(fmt.Sprintf("Parsed [%d] Secure Bookmark entries", count), zap.String("module", moduleName))

	return values, nil
}

func (m MacMRUModule) sfl(inst instance.Instance) ([][]string, error) {
	values := [][]string{}
	count := 0

	sflPaths := util.Multiglob(filepathsSFLs, inst.GetTargetPath())
	if len(sflPaths) == 0 {
		return [][]string{}, errors.New("no SFL files were found")
	}

	for _, path := range sflPaths {
		var valmap = make(map[string]string)
		valmap = util.InitializeMapToEmptyString(valmap, header)

		valmap["user"] = util.GetUsernameFromPath(path)

		// Parse plist/bplist
		data, err := machelpers.DecodePlist(path, inst.GetTargetPath())
		if err != nil {
			zap.L().Error("could not parse plist '"+path+"': "+err.Error(), zap.String("module", moduleName))
			continue
		}
		for _, item := range data {
			machelpers.PrintPlistAsJSON(item)
			return [][]string{}, errors.New("Unimplemented method - go build this feature :) ")
		}
	}
	zap.L().Debug(fmt.Sprintf("Parsed [%d] SFL entries", count), zap.String("module", moduleName))

	return values, nil
}

func (m MacMRUModule) sfl2(inst instance.Instance) ([][]string, error) {
	values := [][]string{}
	count := 0

	sfl2Paths := util.Multiglob(filepathsSFL2s, inst.GetTargetPath())
	if len(sfl2Paths) == 0 {
		return [][]string{}, errors.New("no SFL2 files were found")
	}

	for _, path := range sfl2Paths {
		var valmap = make(map[string]string)
		valmap = util.InitializeMapToEmptyString(valmap, header)

		valmap["user"] = util.GetUsernameFromPath(path)

		// Parse plist/bplist
		data, err := machelpers.DecodePlist(path, inst.GetTargetPath())
		if err != nil {
			zap.L().Error("could not parse plist '"+path+"': "+err.Error(), zap.String("module", moduleName))
			continue
		}
		for _, item := range data {
			machelpers.PrintPlistAsJSON(item)
			return [][]string{}, errors.New("Unimplemented method - go build this feature :) ")
		}
	}
	zap.L().Debug(fmt.Sprintf("Parsed [%d] SFL2 entries", count), zap.String("module", moduleName))

	return values, nil
}

func NSArrayURLToGoSliceURL(arr *C.NSArray) []url.URL {
	var result []url.URL
	length := NSArrayLen(arr)

	for i := uint(0); i < length; i++ {
		nsurl := (*C.NSURL)(NSArrayItem(arr, i))
		u := NSURLToGoURL(nsurl)
		result = append(result, *u)
	}
	return result
}
func NSArrayLen(arr *C.NSArray) uint { return uint(C.NSArrayLen(arr)) }

func NSArrayItem(arr *C.NSArray, i uint) unsafe.Pointer {
	return C.NSArrayItem(arr, C.ulong(i))
}

func NSStringToCString(s *C.NSString) *C.char { return C.NSStringToCString(s) }

func NSStringToGoString(s *C.NSString) string { return C.GoString(NSStringToCString(s)) }

func NSNumberToGoInt(i *C.NSNumber) int { return int(C.NSNumberToGoInt(i)) }

func NSURLToGoURL(nsurlptr *C.NSURL) *url.URL {
	nsurl := *C.NSURLData(nsurlptr)
	userInfo := url.UserPassword(
		NSStringToGoString(nsurl.user),
		NSStringToGoString(nsurl.password),
	)
	host := NSStringToGoString(nsurl.host)

	if nsurl.port != nil {
		port := NSNumberToGoInt(nsurl.port)
		host = host + ":" + strconv.FormatInt(int64(port), 10)
	}

	return &url.URL{
		Scheme:   NSStringToGoString(nsurl.scheme),
		User:     userInfo, // username and password information
		Host:     host,     // host or host:port
		Path:     NSStringToGoString(nsurl.path),
		RawQuery: NSStringToGoString(nsurl.query),    // encoded query values, without '?'
		Fragment: NSStringToGoString(nsurl.fragment), // fragment for references, without '#'
	}
}
