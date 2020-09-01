package windowsdirlist

import (
	"crypto/md5"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/anthonybm/Orion/datawriter"
	"github.com/anthonybm/Orion/instance"
	"github.com/anthonybm/Orion/util"
	"github.com/anthonybm/Orion/util/windowshelpers"
	"github.com/karrick/godirwalk"
	"go.uber.org/zap"
)

type WindowsDirlistModule struct{}

var (
	moduleName         = "WindowsDirlistModule"
	mode               = "windows"
	version            = "1.0"
	description        = "Walks the filesystem and collects data from each item encountered as specified in the config file"
	author             = "Anthony Martinez, martinez.anthonyb@gmail.com"
	hashSizeLimitBytes int
	verbose            bool
	doHashMD5          bool
	doHashSHA256       bool
	walkRootDir        string
	scratchBuffSize    = godirwalk.MinimumScratchBufferSize
)

// Start executes the module with instance instructions
func (m WindowsDirlistModule) Start(inst instance.Instance) error {
	err := m.dirlist(inst)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Error running %s: %s", moduleName, err.Error()), zap.String("module", moduleName))
	}
	return err
}

func (m WindowsDirlistModule) dirlist(inst instance.Instance) error {
	doHashMD5, _ = inst.GetOrionConfig().GetDirlistDoHashMD5()
	doHashSHA256, _ = inst.GetOrionConfig().GetDirlistDohashSHA256()
	hashSizeLimitBytes, _ = inst.GetOrionConfig().GetDirlistHashSizeLimitBytes()
	verbose, _ = inst.GetOrionConfig().IsVerbose()

	// Create OrionWriter
	mw, err := datawriter.NewOrionWriter(moduleName, inst.GetOrionRuntime(), inst.GetOrionOutputFormat(), inst.GetOrionOutputFilepath())
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
		return err
	}

	header := []string{
		"path",
		"name",
		"mode",
		"size",
		"mtime",
		"atime",
		"ctime",
		"btime",
		"sha256",
		"md5",
	}
	values := [][]string{}

	count := 0
	benchmarkStart := time.Now()
	dircount := 0
	filecount := 0
	// symcount := 0
	// devicecount := 0

	// Get a list of logical drives to use as root target paths
	walkRootDirs, err := windowsWalkRootDir(walkRootDir)
	if err != nil {
		return errors.New("Could not determine dirlist target root path! " + err.Error())
	}
	// Remove logical drives from target paths list based on excluded drives from Config
	excludedDrives, _ := inst.GetOrionConfig().GetDirlistExcludedDrives()
	for i, r := range walkRootDirs {
		for _, d := range excludedDrives {
			if r == d {
				// remove d from walkRootDirs
				walkRootDirs[i] = walkRootDirs[len(walkRootDirs)-1]
				walkRootDirs[len(walkRootDirs)-1] = ""
				walkRootDirs = walkRootDirs[:len(walkRootDirs)-1]
			}
		}
	}

	// Exclude directories via Glob using the list of drive root target paths
	excludedDirsNames, _ := inst.GetOrionConfig().GetDirlistExcludedDirs()
	excludedDirs := util.MultiMultiGlob(excludedDirsNames, walkRootDirs)
	zap.L().Debug(fmt.Sprintf("Want to exclude: %s", excludedDirsNames), zap.String("module", moduleName))
	zap.L().Debug(fmt.Sprintf("Actually excluding: %s", excludedDirs), zap.String("module", moduleName))

	// excludedDirsMap := make(map[string]bool) // TODO figure out efficient way to check if substrings of path are excluded?
	// for _, excDir := range excludedDirs {
	// 	excludedDirsMap[excDir] = true
	// }

	excludedExts, _ := inst.GetOrionConfig().GetDirlistExcludedExts()
	excludedExtsMap := make(map[string]bool) // Map for fast access to search excluded
	for _, excExt := range excludedExts {
		excludedExtsMap[excExt] = true
	}

	// loop through target drive root paths with gowalkdir
	for _, root := range walkRootDirs {
		err = godirwalk.Walk(root, &godirwalk.Options{
			Callback: func(osPathname string, de *godirwalk.Dirent) error {
				if de.IsDir() {
					if substringListContains(excludedDirs, osPathname) {
						// zap.L().Warn("SKIPPING DIR: " + osPathname)
						return filepath.SkipDir
					}
					dircount++
					parseDir(osPathname, de)
				} else if de.IsRegular() {
					if excludedExtsMap[util.FileExtension(osPathname)] || excludedExtsMap["."+util.FileExtension(osPathname)] {
						// zap.L().Warn("SKIPPING FILE: " + osPathname)
						return nil
					}
					filecount++
					values = append(values, parseRegular(osPathname, de))
				}
				// } else if de.IsSymlink() {
				// 	symcount++
				// } else if de.IsDevice() {
				// 	devicecount++
				// }
				// } else {
				// 	fmt.Fprintf(os.Stdout, "OTHER: %s\n", osPathname)
				// }
				count++
				return nil
			},
			ErrorCallback: func(osPathname string, err error) godirwalk.ErrorAction {
				if verbose {
					zap.L().Error(fmt.Sprintf("%s", err.Error()), zap.String("module", moduleName))
				}
				// For the purposes of this example, a simple SkipNode will suffice,
				// although in reality perhaps additional logic might be called for.
				return godirwalk.SkipNode
			},
			FollowSymbolicLinks: false, // TESTING
			ScratchBuffer:       make([]byte, scratchBuffSize),
			Unsorted:            true, // set true for faster yet non-deterministic enumeration (see godoc)
		})
		if err != nil {
			zap.L().Error(fmt.Sprintf("%s", err.Error()), zap.String("module", moduleName))
		}
	}
	benchmark := time.Now().Sub(benchmarkStart)
	zap.L().Debug("Walked ["+strconv.Itoa(count)+"] items in "+benchmark.String()+" seconds", zap.String("module", moduleName))
	zap.L().Debug("Dir: ["+strconv.Itoa(dircount)+"]", zap.String("module", moduleName))
	zap.L().Debug("Files: ["+strconv.Itoa(filecount)+"]", zap.String("module", moduleName))
	// zap.L().Debug("SymLinks: "+strconv.Itoa(symcount), zap.String("module", moduleName))
	// zap.L().Debug("Device: "+strconv.Itoa(devicecount), zap.String("module", moduleName))

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

func substringListContains(l []string, substr string) bool {
	for _, val := range l {
		if strings.HasSuffix(substr, val) {
			return true
		}
	}
	return false
}

func parseDir(osPathname string, de *godirwalk.Dirent) {
	// zap.L().Debug("Dir: "+osPathname, zap.String("module", moduleName))
}
func parseRegular(osPathname string, de *godirwalk.Dirent) []string {
	metadata, _ := windowshelpers.FileMetadata(osPathname, moduleName)
	hashSHA256 := "N/E"
	hashMD5 := "N/E"
	size, _ := strconv.Atoi(metadata["size"])

	if doHashSHA256 && (size < hashSizeLimitBytes) {
		h, err := fileSHA256(osPathname)
		if err != nil {
			hashSHA256 = "ERROR"
		}
		hashSHA256 = h
	}
	if doHashMD5 && (size < hashSizeLimitBytes) {
		h, err := fileMD5(osPathname)
		if err != nil {
			hashMD5 = "ERROR"
		}
		hashMD5 = h
	}

	entry := []string{
		metadata["path"],  // "path",
		metadata["name"],  // "name",
		metadata["mode"],  // "mode",
		metadata["size"],  // "size",
		metadata["mtime"], // "mtime",
		metadata["atime"], // "atime",
		metadata["ctime"], // "ctime",
		metadata["btime"], // "btime",
		hashSHA256,
		hashMD5,
	}
	// zap.L().Debug(fmt.Sprintf("Regular: %s SHA256: %s", osPathname, entry[0]), zap.String("module", moduleName))

	return entry
}

func fileSHA256(fp string) (string, error) {
	f, err := os.Open(fp)
	if err != nil {
		zap.L().Error(fmt.Sprintf("file open error %s for %s: ", err.Error(), fp), zap.String("module", moduleName))
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		zap.L().Error(fmt.Sprintf("sha256 error %s for %s: ", err.Error(), fp), zap.String("module", moduleName))
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func fileMD5(fp string) (string, error) {
	f, err := os.Open(fp)
	if err != nil {
		zap.L().Error(fmt.Sprintf("file open error %s for %s: ", err.Error(), fp), zap.String("module", moduleName))
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		zap.L().Error(fmt.Sprintf("md5 error %s for %s: ", err.Error(), fp), zap.String("module", moduleName))
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// Use win32 API to get list of logical drives for traversal. C:\, D:\, Z:\
func windowsWalkRootDir(root string) ([]string, error) {
	kernel32, _ := syscall.LoadLibrary("kernel32.dll")
	getLogicalDrivesHandle, _ := syscall.GetProcAddress(kernel32, "GetLogicalDrives")
	var drives []string

	if ret, _, err := syscall.Syscall(uintptr(getLogicalDrivesHandle), 0, 0, 0, 0); err != 0 {
		return []string{}, errors.New("Syscall Error grabbing logical drives: " + err.Error())
	} else {
		bmap := uint32(ret)
		availableDrives := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"}
		for i := range availableDrives {
			if bmap&1 == 1 {
				drives = append(drives, availableDrives[i])
			}
			bmap >>= 1
		}
	}
	for i, d := range drives {
		drives[i] = d + ":"
	}
	zap.L().Debug(fmt.Sprintf("Found these logical drives: %s", drives), zap.String("module", moduleName))
	return drives, nil
}
