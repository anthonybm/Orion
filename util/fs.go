package util

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// Exists returns whether the given file or directory exists
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// DirPermissions are the default permission bits we apply to directories.
const DirPermissions = os.ModeDir | 0775

// PathExists returns true if the given path exists, as a file or a directory.
func PathExists(filename string) bool {
	_, err := os.Lstat(filename)
	return err == nil
}

// FileExists returns true if the given path exists and is a file.
func FileExists(filename string) bool {
	info, err := os.Lstat(filename)
	return err == nil && !info.IsDir()
}

// IsSymlink returns true if the given path exists and is a symlink.
func IsSymlink(filename string) bool {
	info, err := os.Lstat(filename)
	return err == nil && (info.Mode()&os.ModeSymlink) != 0
}

// IsDirectory checks if a given path is a directory
func IsDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// CopyFiles globs a list of filenames from fileglobnames and copies those files to the destfolder. Returns a list of the paths of copied files and an error (nil if no error)
// fileglobnames is case sensitive and must be a slice of strings formatted as glob ex. Users\\*\\OneDrive MINUS the rootTargetPath
// destfolder must be relative to your current Orion execution folder
// rootTargetPath specifies the prefix of the fileglobnames
func CopyFiles(fileglobnames []string, destfolder string, rootTargetPath string) ([]string, error) {
	// Glob fileglobnames for list of files we want to copy
	// Prepends rootTargetPath to each item in fileglobnames
	filesToCopy := Multiglob(fileglobnames, rootTargetPath)

	// Return if list is empty
	if len(filesToCopy) == 0 {
		zap.L().Debug(fmt.Sprintf("[fs CopyFiles] - No files were copied from %s to %s", fileglobnames, destfolder))
		return []string{}, nil
	}

	// Ensure directory path exists and if not create it
	if _, err := os.Stat(destfolder); os.IsNotExist(err) {
		err = os.MkdirAll(destfolder, 0700)
		if err != nil {
			return []string{}, err
		}
	}

	// Create mapping of globbed/source-filenames to copied/dest-filenames
	filenameSuffix := "_Orion_Copy"
	var sourceTodest = make(map[string]string)
	for _, fileToCopy := range filesToCopy {
		sourceTodest[fileToCopy] = filepath.Join(destfolder, strings.TrimSuffix(filepath.Base(fileToCopy), filepath.Ext(filepath.Base(fileToCopy)))+filenameSuffix+filepath.Ext(fileToCopy))
		// fmt.Println(filepath.Join(destfolder, strings.TrimSuffix(filepath.Base(fileToCopy), filepath.Ext(filepath.Base(fileToCopy)))+filenameSuffix))
	}

	// CopyFile source to dest
	filesCopied := []string{}
	for source, dest := range sourceTodest {
		err := CopyFile(source, dest, os.FileMode(int(0777)))
		if err != nil {
			zap.L().Error(fmt.Sprintf("[fs CopyFiles] - failed to copy source %s to dest %s: %s", source, dest, err.Error()))
			continue
		}
		zap.L().Debug(fmt.Sprintf("[fs CopyFiles] - Copied source %s to dest %s", source, dest))
		filesCopied = append(filesCopied, dest)
	}
	if len(filesCopied) == 0 {
		zap.L().Debug(fmt.Sprintf("[fs CopyFiles] - No files were copied from %s to %s", fileglobnames, destfolder))
		return []string{}, nil
	}
	return filesCopied, nil
}

// CopyFile copies a file from 'from' to 'to', with an attempt to perform a copy & rename
// to avoid chaos if anything goes wrong partway.
func CopyFile(from string, to string, mode os.FileMode) error {
	fromFile, err := os.Open(from)
	if err != nil {
		return err
	}
	defer fromFile.Close()
	return WriteFile(fromFile, to, mode)
}

// WriteFile writes data from a reader to the file named 'to', with an attempt to perform
// a copy & rename to avoid chaos if anything goes wrong partway.
func WriteFile(fromFile io.Reader, to string, mode os.FileMode) error {
	if err := os.RemoveAll(to); err != nil {
		return err
	}
	dir, file := split(to)
	if err := os.MkdirAll(dir, DirPermissions); err != nil {
		fmt.Println("Failed to make ", dir, " from ", to)
		return err
	}
	tempFile, err := ioutil.TempFile(dir, file)
	if err != nil {
		return err
	}
	if _, err := io.Copy(tempFile, fromFile); err != nil {
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	// OK, now file is written; adjust permissions appropriately.
	if mode == 0 {
		mode = 0664
	}
	if err := os.Chmod(tempFile.Name(), mode); err != nil {
		return err
	}
	// And move it to its final destination.
	return os.Rename(tempFile.Name(), to)
}

func split(p string) (dir, file string) {
	dir, file = path.Split(p)
	if dir == "" && strings.Contains(p, "\\") {
		return filepath.Split(p)
	}
	return dir, file
}

// Multiglob returns a list of globbed strings based on input list of patterns
func Multiglob(sliceGlob []string, targetPath string) []string {
	res := []string{}
	for _, s := range sliceGlob {
		f, err := filepath.Glob(filepath.Join(targetPath, s))
		if err != nil {
			zap.L().Error(err.Error())
		}
		if len(f) == 0 {
			zap.L().Debug("files not found in '" + filepath.Join(targetPath, s) + "'")
			continue
		}
		for _, i := range f {
			res = append(res, i)
		}
	}
	return res
}

// MultiMultiGlob returns a list of globbed strings based on input list of patterns for an input list of target paths
func MultiMultiGlob(sliceGlob []string, targetPaths []string) []string {
	res := []string{}
	for _, t := range targetPaths {
		res = append(res, Multiglob(sliceGlob, t)...)
	}
	return res
}

func Glob(glob string, targetPath string) []string {
	f, _ := filepath.Glob(filepath.Join(targetPath, glob))
	return f
}
