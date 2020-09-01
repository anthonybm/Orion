package macdirlist

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/anthonybm/Orion/datawriter"
	"github.com/anthonybm/Orion/instance"
	"github.com/anthonybm/Orion/util"
	"github.com/anthonybm/Orion/util/machelpers"
	"github.com/karrick/godirwalk"
	"go.uber.org/zap"
)

type MacDirlistModule struct {
}

var (
	moduleName         = "MacDirlistModule"
	mode               = "mac"
	version            = "1.0"
	description        = "Walks the filesystem and collects data from each item encountered as specified in the config file"
	author             = "Anthony Martinez, martinez.anthonyb@gmail.com"
	hashSizeLimitBytes int
	doHashMD5          bool
	doHashSHA256       bool
	walkRootDir        string
	verbose            bool
	scratchBuffSize    = godirwalk.MinimumScratchBufferSize
)

// Start executes the module with Config instructions and writes to OrionWriter
func (m MacDirlistModule) Start(inst instance.Instance) error {
	err := m.dirlist(inst)
	if err != nil {
		zap.L().Error("Error running "+moduleName+": "+err.Error(), zap.String("module", moduleName))
	}
	return err
}

func (m MacDirlistModule) dirlist(inst instance.Instance) error {
	doHashMD5, _ = inst.GetOrionConfig().GetDirlistDoHashMD5()
	doHashSHA256, _ = inst.GetOrionConfig().GetDirlistDohashSHA256()
	walkRootDir = inst.GetTargetPath()
	hashSizeLimitBytes, _ = inst.GetOrionConfig().GetDirlistHashSizeLimitBytes()
	verbose, _ = inst.GetOrionConfig().IsVerbose()

	mw, err := datawriter.NewOrionWriter(moduleName, inst.GetOrionRuntime(), inst.GetOrionOutputFormat(), inst.GetOrionOutputFilepath())
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
		return err
	}

	header := []string{
		"mode",
		"size",
		"owner",
		"uid",
		"gid",
		"mtime",
		"atime",
		"ctime",
		"btime",
		"path",
		"name",
		"sha256",
		"md5",
		"quarantine",
		"wherefrom_1",
		"wherefrom_2",
		// "code_signatures",
	}
	values := [][]string{}

	count := 0
	benchmarkStart := time.Now()
	dircount := 0
	filecount := 0
	// symcount := 0
	// devicecount := 0

	excludedDirs, _ := inst.GetOrionConfig().GetDirlistExcludedDirs()
	// also for non-forensic mode, exclude /Volumes/* to prevent recusion of mounted volumes
	if !inst.ForensicMode() {
		excludeVols, _ := filepath.Glob(filepath.Join(inst.GetTargetPath(), "Volumes/*"))
		excludedDirs = append(excludedDirs, excludeVols...)
	}
	// excludedDirsMap := make(map[string]bool) // TODO figure out efficient way to check if substrings of path are excluded?
	// for _, excDir := range excludedDirs {
	// 	excludedDirsMap[excDir] = true
	// }

	excludedExts, _ := inst.GetOrionConfig().GetDirlistExcludedExts()
	excludedExtsMap := make(map[string]bool) // Map for fast access to search excluded
	for _, excExt := range excludedExts {
		excludedExtsMap[excExt] = true
	}

	// var wg sync.WaitGroup // TODO multithreading
	err = godirwalk.Walk(walkRootDir, &godirwalk.Options{
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			if de.IsDir() {
				if substringListContains(excludedDirs, osPathname) {
					// zap.L().Warn("SKIPPING DIR: " + osPathname)
					return filepath.SkipDir
				}
				// wg.Add(1)
				dircount++
				parseDir(osPathname, de /*, &wg*/)
			} else if de.IsRegular() {
				if excludedExtsMap[util.FileExtension(osPathname)] || excludedExtsMap["."+util.FileExtension(osPathname)] {
					// zap.L().Warn("SKIPPING FILE: " + osPathname)
					return nil
				}
				// wg.Add(1)
				filecount++
				values = append(values, parseRegular(osPathname, de /*, &wg*/))
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
				zap.L().Error(err.Error(), zap.String("module", moduleName))
			}
			// For the purposes of this example, a simple SkipNode will suffice,
			// although in reality perhaps additional logic might be called for.
			return godirwalk.SkipNode
		},
		FollowSymbolicLinks: false, // TESTING
		ScratchBuffer:       make([]byte, scratchBuffSize),
		Unsorted:            true, // set true for faster yet non-deterministic enumeration (see godoc)
	})
	// wg.Wait()
	if err != nil {
		zap.L().Error(err.Error(), zap.String("module", moduleName))
	}
	benchmark := time.Now().Sub(benchmarkStart)
	zap.L().Debug("Walked ["+strconv.Itoa(count)+"] files in "+benchmark.String()+" seconds", zap.String("module", moduleName))
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
	// zap.L().Debug("Regular: "+osPathname, zap.String("module", moduleName))
	metadata, _ := machelpers.FileMetadata(osPathname, moduleName)
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

	// get quarantine extended attribute if available
	quarantineXattr := "N/E"
	quarantineXattrBytes, err := machelpers.ReadXAttr(osPathname, "com.apple.quarantine")
	if err != nil {
		quarantineXattr = "ERROR"
	} else {
		quarantineXattr = hex.EncodeToString(quarantineXattrBytes)
	}

	// get wherefrom extended attribute for each file, if available
	wherefromXattr1 := ""
	wherefromXattr2 := ""
	// wherefromXattrBytes, err := machelpers.ReadXAttr(osPathname, "com.apple.metadata:kMDItemWhereFroms")
	// if err != nil {
	// 	wherefromXattr1 = "ERROR"
	// 	wherefromXattr2 = "ERROR"
	// } else if strings.HasPrefix(string(wherefromXattrBytes), "bplist") {
	// 	parsedWF, err := machelpers.DecodePlistBytes(wherefromXattrBytes)
	// 	if err == nil {
	// 		fmt.Print(parsedWF)
	// 	}
	// }
	entry := []string{
		metadata["mode"],  // "mode",
		metadata["size"],  // "size",
		"N/P",             // "owner",
		metadata["uid"],   // "uid",
		metadata["gid"],   // "gid",
		metadata["mtime"], // "mtime",
		metadata["atime"], // "atime",
		metadata["ctime"], // "ctime",
		metadata["btime"], // "btime",
		metadata["path"],  // "path",
		metadata["name"],  // "name",
		hashSHA256,        // "sha256",
		hashMD5,           // "md5",
		quarantineXattr,   // "quarantine",
		wherefromXattr1,   // "wherefrom_1",
		wherefromXattr2,   // "wherefrom_2",
		// "code_signatures",
	}

	// values = append(values, entry)
	return entry
}

func fileSHA256(fp string) (string, error) {
	f, err := os.Open(fp)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func fileMD5(fp string) (string, error) {
	f, err := os.Open(fp)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
