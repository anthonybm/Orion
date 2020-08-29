package windowshelpers

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/djherbis/times.v1"
)

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

	// var UID int
	// var GID int
	// if stat, ok := stat.Sys().(*syscall.Stat_t); ok {
	// 	UID = int(stat.Uid)
	// 	GID = int(stat.Gid)
	// } else {
	// 	zap.L().Debug("getting file metadata for non-linux file: '" + fp + "'")
	// 	UID = os.Getuid()
	// 	GID = os.Getgid()
	// }
	// m["uid"] = strconv.Itoa(UID)
	// m["gid"] = strconv.Itoa(GID)
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
