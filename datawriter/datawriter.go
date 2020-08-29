package datawriter

import (
	"bufio"
	"encoding/csv"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"
)

type OrionWriterInterface interface {
	Write() error
	WriteAll() error
}

type OrionWriter struct {
	csv         bool
	csvmw       *CSVOrionWriter
	xlsx        bool
	xlsxmw      *XLSXOrionWriter
	outfilepath string
}

type CSVOrionWriter struct {
	mutex      *sync.Mutex
	csvwriter  *csv.Writer
	file       *os.File
	filebuffer *bufio.Writer
	module     string
	runtime    string
}

type XLSXOrionWriter struct {
	mutex   *sync.Mutex
	file    *os.File
	module  string
	runtime string
}

func NewOrionWriter(module string, orionRuntime string, outputtype string, fp string) (OrionWriter, error) {
	switch outputtype {
	case "csv":
		// Ensure directory path exists and if not create it
		if _, err := os.Stat(fp); os.IsNotExist(err) {
			os.MkdirAll(fp, 0700)
		}

		fn := orionRuntime + "_" + module + "." + outputtype
		if !strings.HasSuffix(fp, "/") {
			fp = fp + "/"
		}
		fp, _ := filepath.Abs(fp + fn)
		// file, err := os.OpenFile(fp, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
		file, err := os.Create(fp)
		if err != nil {
			zap.L().Error("error creating file " + fp + ".")
			return OrionWriter{}, errors.New("error creating file " + fp + ".")
		}

		filebuffer := bufio.NewWriter(file) // buffered writer for performance (not making syscall per write)
		defer filebuffer.Flush()

		filewriter := csv.NewWriter(filebuffer)

		csvmw := CSVOrionWriter{
			mutex:      &sync.Mutex{},
			csvwriter:  filewriter,
			file:       file,
			filebuffer: filebuffer,
			module:     module,
			runtime:    orionRuntime,
		}

		zap.L().Debug("OrionWriter: Created file for module: " + module + ".")
		return OrionWriter{
			csv:         true,
			csvmw:       &csvmw,
			xlsx:        false,
			outfilepath: fp,
		}, nil
	}
	return OrionWriter{}, errors.New("cannot create OrionWriter for the given output type")
}

func (mw OrionWriter) SelfDestruct() error {
	zap.L().Debug("Removing OrionWriter: " + mw.outfilepath)
	return os.Remove(mw.outfilepath)
}

func (mw OrionWriter) GetOutfilePath() string {
	return mw.outfilepath
}

func (mw OrionWriter) GetOutputType() string {
	if mw.csv == true {
		return "csv"
	} else if mw.xlsx == true {
		return "xlsx"
	} else {
		return "ERROR"
	}
}

func (mw OrionWriter) GetOrionRuntime() string {
	if mw.csv == true {
		return mw.csvmw.runtime
	} else if mw.xlsx == true {
		return mw.xlsxmw.runtime
	} else {
		return "ERROR"
	}
}

func (mw OrionWriter) WriteHeader(header []string) error {
	return mw.Write(header)
}

// Write writes a single entry to output
func (mw OrionWriter) Write(entry []string) error {
	outputtype := mw.GetOutputType()
	if outputtype == "ERROR" || outputtype == "" {
		return errors.New("could not get OrionWriter output type, found: '" + outputtype + "'")
	}
	switch outputtype {
	case "csv":
		// zap.L().Debug("Sending entry to CSV Writer: " + strings.Join(entry, " "))
		err := mw.csvmw.Write(entry)
		if err != nil {
			return err
		}
		return mw.csvmw.Flush()
	}
	return errors.New("failed to write entry")
}

// WriteAll writes multiple entries to output
func (mw OrionWriter) WriteAll(entries [][]string) error {
	outputtype := mw.GetOutputType()
	if outputtype == "ERROR" || outputtype == "" {
		return errors.New("could not get OrionWriter output type, found: '" + outputtype + "'")
	}
	switch outputtype {
	case "csv":
		err := mw.csvmw.WriteAll(entries)
		if err != nil {
			return err
		}
		return mw.csvmw.Flush()
	}
	return errors.New("failed to write entries")
}

func (mw OrionWriter) WriteOutput(header []string, entries [][]string) error {
	outputtype := mw.GetOutputType()
	if outputtype == "ERROR" || outputtype == "" {
		return errors.New("could not get OrionWriter output type, found: '" + outputtype + "'")
	}
	switch outputtype {
	case "csv":
		err := mw.csvmw.WriteOutput(header, entries)
		if err != nil {
			return err
		}
		return mw.csvmw.Flush()
	}
	return errors.New("failed to write header and entries to output")
}

func (mw OrionWriter) Close() error {
	outputtype := mw.GetOutputType()
	if outputtype == "ERROR" || outputtype == "" {
		return errors.New("could not get OrionWriter output type, found: '" + outputtype + "'")
	}
	switch outputtype {
	case "csv":
		return mw.csvmw.Close()
	}
	return errors.New("failed to close file")
}

func (cmw *CSVOrionWriter) Write(row []string) error {
	cmw.mutex.Lock()
	defer cmw.mutex.Unlock()
	return cmw.csvwriter.Write(row)

}

func (cmw *CSVOrionWriter) WriteAll(rows [][]string) error {
	cmw.mutex.Lock()
	defer cmw.mutex.Unlock()
	return cmw.csvwriter.WriteAll(rows)
}

func (cmw *CSVOrionWriter) WriteOutput(header []string, values [][]string) error {
	err := cmw.Write(header)
	err = cmw.WriteAll(values)
	err = cmw.Close()
	if err != nil {
		return err
	}
	return nil
}

// Flush forces any pending writes
func (w *CSVOrionWriter) Flush() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.csvwriter.Flush()
	return w.csvwriter.Error()
}

// Close CSV file for writing (calls Flush() implicitly)
func (w *CSVOrionWriter) Close() error {
	err := w.Flush()
	if err != nil {
		return err
	}
	return w.file.Close()
}
