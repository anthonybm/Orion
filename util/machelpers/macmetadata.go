// +build darwin

package machelpers

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/xattr"
	"go.uber.org/zap"
	"gopkg.in/djherbis/times.v1"
)

func FileTimestamps(fp string, modulename string) map[string]string {
	var m = make(map[string]string)
	m["mtime"] = "NO VALUE"
	m["atime"] = "NO VALUE"
	m["ctime"] = "NO VALUE"
	m["btime"] = "NO VALUE"

	timestat, err := times.Stat(fp)
	if err != nil {
		// zap.L().Error("Could not get metadata for '" + fp + "': " + err.Error(), zap.String("module", modulename))
		return m
	}

	m["mtime"] = timestat.ModTime().UTC().Format(time.RFC3339)
	m["atime"] = timestat.AccessTime().UTC().Format(time.RFC3339)
	if timestat.HasChangeTime() {
		m["ctime"] = timestat.ChangeTime().UTC().Format(time.RFC3339)
	}
	if timestat.HasBirthTime() {
		m["btime"] = timestat.BirthTime().UTC().Format(time.RFC3339)
	}

	return m
}

func FileMetadata(fp string, modulename string) (map[string]string, error) {
	var m = make(map[string]string)
	m["mode"] = "NO VALUE"
	m["size"] = "NO VALUE"
	m["uid"] = "NO VALUE"
	m["gid"] = "NO VALUE"
	m["mtime"] = "NO VALUE"
	m["atime"] = "NO VALUE"
	m["ctime"] = "NO VALUE"
	m["btime"] = "NO VALUE"
	m["path"] = "NO VALUE"
	m["name"] = "NO VALUE"

	stat, err := os.Lstat(fp)
	if err != nil {
		// zap.L().Error("Could not get metadata for '" + fp + "': " + err.Error(), zap.String("module", modulename))
		return m, errors.New("Could not get metadata for '" + fp + "': " + err.Error())
	}
	timestat, err := times.Stat(fp)
	if err != nil {
		// zap.L().Error("Could not get metadata for '" + fp + "': " + err.Error(), zap.String("module", modulename))
		return m, errors.New("Could not get time metadata for '" + fp + "': " + err.Error())
	}

	mode := stat.Mode()
	if mode.IsRegular() {
		m["mode"] = "Regular File"
	} else if mode.IsDir() {
		m["mode"] = "Directory"
	} else if mode&os.ModeSymlink != 0 {
		m["mode"] = "Symbolic Link"
	} else if mode&os.ModeNamedPipe != 0 {
		m["mode"] = "Named Pipe"
	}
	m["size"] = strconv.FormatInt(stat.Size(), 10)

	var UID int
	var GID int
	if stat, ok := stat.Sys().(*syscall.Stat_t); ok {
		UID = int(stat.Uid)
		GID = int(stat.Gid)
	} else {
		zap.L().Debug("getting file metadata for non-linux file: '" + fp + "'")
		UID = os.Getuid()
		GID = os.Getgid()
	}
	m["uid"] = strconv.Itoa(UID)
	m["gid"] = strconv.Itoa(GID)
	m["mtime"] = timestat.ModTime().UTC().String()
	m["atime"] = timestat.AccessTime().UTC().String()
	if timestat.HasChangeTime() {
		m["ctime"] = timestat.ChangeTime().UTC().String()
	}
	if timestat.HasBirthTime() {
		m["btime"] = timestat.BirthTime().UTC().String()
	}
	m["path"] = fp
	m["name"] = filepath.Base(fp)

	return m, nil
}

// ReadAttr reads an attribute from the given file, using xattrs
// It returns an empty slice if it can't be read.
func ReadXAttr(filename, xattrName string) ([]byte, error) {
	b, err := xattr.LGet(filename, xattrName)
	if err != nil {
		// if util.IsSymlink(filename) {
		// 	// Symlinks can't take xattrs on Linux. We stash it on the fallback hash file instead.
		// 	// What about MacOS?
		// 	return ReadAttrFile(filename)
		// } else if e2 := err.(*xattr.Error).Err; !os.IsNotExist(e2) && e2 != xattr.ENOATTR {
		// 	log.Warning("Failed to read hash for %s: %s", filename, err)
		// }
		return nil, err
	}
	return b, nil
}

// ListXAttr returns the list of Xattrs for given file
func ListXAttr(filename string) ([]string, error) {
	return xattr.List(filename)
}

// ReadAttrFile reads a hash for the given file. It's the fallback for ReadAttr and pairs with
// RecordAttrFile to read the same files it would write.
func ReadAttrFile(filename string) []byte {
	b, _ := ioutil.ReadFile(filename)
	return b
}

func GetCodesignatures(fp string) []string {
	_, err := os.Stat(fp)
	if os.IsNotExist(err) {
		return []string{"ERROR-FILE-DNE"}
	}

	signers, err := getSignatureChain(fp)
	if err != nil {
		signers, err = getCodeSignaturesFromSubProcess(fp)
		if err != nil {
			return []string{"ERROR-GETSIG-FAIL"}
		}
	}
	if len(signers) == 0 {
		return []string{"Unsigned"}
	}
	return signers
}

func getSignatureChain(fp string) ([]string, error) {
	return []string{}, errors.New("unimplemented method")
}

func getCodeSignaturesFromSubProcess(fp string) ([]string, error) {
	codesignCmd := exec.Command("codesign", "-dv", "--verbose=2", fp)
	codesignOut, outerr := codesignCmd.StdoutPipe()
	codesignErr, errerr := codesignCmd.StderrPipe()
	if outerr != nil {
		return []string{}, errors.New("codesignOut error - could not parse codesign: " + fp + ": " + outerr.Error())
	}
	if errerr != nil {
		return []string{}, errors.New("codesignErr error - could not parse codesign: " + fp + ": " + errerr.Error())
	}
	codesignCmd.Start()
	codesignOutBytes, outbyteserr := ioutil.ReadAll(codesignOut)
	codesignErrorBytes, errbyteserr := ioutil.ReadAll(codesignErr)
	if outbyteserr != nil {
		return []string{}, errors.New("codesignOutBytes error - could not parse codesign: " + fp + ": " + outbyteserr.Error())
	}
	if errbyteserr != nil {
		return []string{}, errors.New("codesignErrorBytes error - could not parse codesign: " + fp + ": " + errbyteserr.Error())
	}
	waiterr := codesignCmd.Wait()
	if waiterr != nil {
		return []string{}, errors.New("codesignCmd wait error - could not parse codesign: " + fp + ": " + waiterr.Error())
	}

	codesignOutString := string(codesignOutBytes)
	codesignErrorString := string(codesignErrorBytes)

	if len(codesignErrorString) > 0 {
		if !strings.Contains(codesignErrorString, "NOTE:") {
			return []string{}, fmt.Errorf("codesignErrorString not empty - could not parse '%s': %s", fp, codesignErrorString)
		}
	}

	codesignData := strings.Split(codesignOutString, "\n")
	signers := []string{}
	for _, line := range codesignData {
		if strings.HasPrefix(line, "Authority=") {
			nline := strings.Replace(line, "Authority=", "", 1)
			signers = append(signers, nline)
		}
	}
	if len(signers) == 0 {
		return []string{"Unsigned"}, nil
	}
	return signers, nil
}
