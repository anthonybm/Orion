package engine

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/anthonybm/Orion/instance"
	"github.com/anthonybm/Orion/windows/modules/windowsdirlist"
	"go.uber.org/zap"
)

var typeRegistry = make(map[string]reflect.Type)

func registerType(elem interface{}) {
	t := reflect.TypeOf(elem).Elem()
	typeRegistry[t.Name()] = t
}

// init contains the registrations for module struct types, must be created before Orion runs
func init() {
	registerType((*windowsdirlist.WindowsDirlistModule)(nil))
	// ... add future modules here
}

// Execute runs Orion modules based on Config file, should only be called once
// Exposes instance functions
func Execute(i instance.Instance) error {
	modules, err := i.GetOrionModules()
	if err != nil {
		return err
	}
	err = executeModules(modules, i)

	return err
}

// makeInstance returns a new instance of the given named type as an interface and a bool indicating it is valid
func makeInstance(name string) (interface{}, bool) {
	elem, ok := typeRegistry[name]
	if !ok {
		return nil, false
	}
	return reflect.New(elem).Elem().Interface(), true
}

// executeModules executes the modules based on the strings in the input slice
func executeModules(modules []string, i instance.Instance) error {
	zap.L().Debug("[" + strconv.Itoa(len(modules)) + "]" + " modules will execute")
	execCount := 0
	benchmarkStart := time.Now()
	if i.NoMultithreading() == false {
		var wg sync.WaitGroup
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt)

		go func() {
			select {
			case sig := <-c:
				println()
				zap.L().Warn(fmt.Sprintf("Got %s signal. Attempting safe abort...\n", sig))
				var abortWg sync.WaitGroup
				abortWg.Add(1)
				go safeAbort(i, &abortWg)
				abortWg.Wait()
				os.Exit(1)
			}
		}()
		for _, module := range modules {
			wg.Add(1)
			execCount++
			zap.L().Debug(module + " sent to goroutine")
			go executeModule(module, &wg, i /*fields here*/)
		}
		zap.L().Debug("[" + strconv.Itoa(execCount) + "/" + strconv.Itoa(len(modules)) + "]" + " modules have been sent to goroutines")
		zap.L().Debug("Waiting for module goroutines to finish")
		wg.Wait()
		zap.L().Debug("module goroutines completed")
	} else {
		for _, module := range modules {
			executeModule(module, nil, i /*fields here*/)
		}
	}
	benchmark := time.Now().Sub(benchmarkStart)
	zap.L().Info("Finished all " + strconv.Itoa(len(modules)) + " modules in " + benchmark.String())
	archive(i)
	return nil
}

func archive(i instance.Instance) {
	files, err := filepath.Glob(i.GetOrionOutputFilepath() + "/*")
	if err != nil {
		zap.L().Error("Failed to glob for output files to archive")
		return
	}
	output := i.GetOrionRuntime() + ".zip"

	zf, err := os.Create(output)
	if err != nil {
		zap.L().Error("Failed to create output archive")
		return
	}
	defer zf.Close()

	zw := zip.NewWriter(zf)
	defer zw.Close()

	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			zap.L().Error("Failed to open " + file + " for archiving")
			continue
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			zap.L().Error("Failed to get stats for " + file + " for archiving")
			continue
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			zap.L().Error("Failed to get header info for " + file + " for archiving")
			continue
		}

		header.Name = file
		header.Method = zip.Deflate

		wr, err := zw.CreateHeader(header)
		if err != nil {
			zap.L().Error("Failed to create archive header for " + file)
			continue
		}

		_, err = io.Copy(wr, f)
		if err != nil {
			zap.L().Error("Failed to write " + file + " to archive")
			continue
		}
		zap.L().Debug("Wrote " + file + " to archive")
	}

	zap.L().Info("Orion packing complete")
	zap.L().Sync()
	zap.L().Core().Sync()
	err = i.CloseLogger()
	if err != nil {
		fmt.Println(fmt.Sprintf("Failed to close Orion Logger"))
	}
	err = os.RemoveAll(i.GetOrionOutputFilepath())
	if err != nil {
		fmt.Println(fmt.Sprintf("Failed to remove Orion output folder '%s': %s", i.GetOrionOutputFilepath(), err.Error()))
		if strings.Contains(err.Error(), "used by another") {
			fmt.Println("Waiting 5 seconds before attempting again")
			time.Sleep(5 * time.Second)
			err = os.RemoveAll(i.GetOrionOutputFilepath())
			if err != nil {
				fmt.Println(fmt.Sprintf("Still failed to remove Orion output folder: %s", err.Error()))
			}
		}
	}
}

func safeAbort(i instance.Instance, wg *sync.WaitGroup) {
	defer wg.Done()
	zap.L().Warn("DO NOT INTERRUPT - Packaging files before terminating...")
	zap.L().Warn("Does not save output progress made by modules not yet complete!!!") // TODO Channels?
	files, err := filepath.Glob(i.GetOrionOutputFilepath() + "/*")
	if err != nil {
		zap.L().Error("Failed to glob for output files to archive")
		return
	}
	output := i.GetOrionRuntime() + "_ABORT.zip"

	zf, err := os.Create(output)
	if err != nil {
		zap.L().Error("Failed to create output archive for terminated instance")
		return
	}
	defer zf.Close()

	zw := zip.NewWriter(zf)
	defer zw.Close()

	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			zap.L().Error("Failed to open " + file + " for archiving")
			continue
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			zap.L().Error("Failed to get stats for " + file + " for archiving")
			continue
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			zap.L().Error("Failed to get header info for " + file + " for archiving")
			continue
		}

		header.Name = file
		header.Method = zip.Deflate

		wr, err := zw.CreateHeader(header)
		if err != nil {
			zap.L().Error("Failed to create archive header for " + file)
			continue
		}

		_, err = io.Copy(wr, f)
		if err != nil {
			zap.L().Error("Failed to write " + file + " to archive")
			continue
		}
		zap.L().Warn("Wrote " + file + " to abort archive")
	}

	zap.L().Warn("Orion termination complete")
}

// executeModule takes in the name of a method we want to retreive, module, and executes it
// Uses instances of receiver type *Module
func executeModule(module string, wg *sync.WaitGroup, inst instance.Instance /*fields here*/) error {
	// Handle multithreading, defer synchronous waitgroup
	if wg != nil {
		defer wg.Done()
	}

	zap.L().Debug("Starting [" + module + "] module.")
	startTime := time.Now()

	// Take instance of *Module and turn into reflect.Value via reflect.ValueOf()
	// Call MethodByName() with the name of the method we want to retreive
	out, err := invoke(module, "Start", inst)
	if err != nil {
		finishTime := time.Now()
		zap.L().Error("Exiting ["+module+"] module with errors. Total time: "+finishTime.Sub(startTime).String(), zap.Error(err))
		return err
	}
	// out[0] will always be an error
	if out[0].Interface() != nil {
		finishTime := time.Now()
		zap.L().Error("Exiting ["+module+"] module with errors. Total time: "+finishTime.Sub(startTime).String(), zap.Error(out[0].Interface().(error)))
		return err
	}

	finishTime := time.Now()
	zap.L().Info("Finished [" + module + "] module. Total time: " + finishTime.Sub(startTime).String())
	return nil
}

// invoke takes any struct type, a method name of that struct, and arguments and executes the method
// creates an instance of the struct type
func invoke(mod string, name string, args ...interface{}) (out []reflect.Value, err error) {
	s, ok := makeInstance(mod)
	if !ok {
		return make([]reflect.Value, 0), fmt.Errorf("failed to create instance of '%s'", mod)
	}
	if strings.Split(reflect.TypeOf(s).String(), ".")[1] != mod {
		return make([]reflect.Value, 0), fmt.Errorf("typeRegistry returned wrong instance '%s' for '%s", strings.Split(reflect.TypeOf(s).String(), ".")[1], mod)
	}

	modValue := reflect.ValueOf(s)
	m := modValue.MethodByName(name)

	if !m.IsValid() {
		return make([]reflect.Value, 0), fmt.Errorf("method not found \"%s\"", name)
	}
	inputs := make([]reflect.Value, len(args))
	for i, arg := range args {
		inputs[i] = reflect.ValueOf(arg)
	}

	out = m.Call(inputs)
	return
}
