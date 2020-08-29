// +build darwin

package maceventtaps

//#cgo CFLAGS: -x objective-c
//#cgo LDFLAGS: -framework CoreGraphics -framework ApplicationServices -framework Foundation
//#include "eventtaps.h"
import "C"
import (
	"fmt"
	"strconv"
	"time"
	"unsafe"

	"github.com/tonythetiger06/Orion/datawriter"
	"github.com/tonythetiger06/Orion/instance"
	"github.com/tonythetiger06/Orion/util"
	"go.uber.org/zap"
)

var (
	moduleName  = "MacEventTapsModule"
	mode        = "mac"
	version     = "1.0"
	description = `
	eventtaps
	`
	author = "Anthony Martinez, martinez.anthonyb@gmail.com"
)

var (
	header = []string{"eventTapID", "tapPoint", "options", "eventsOfInterest", "tappingProcess", "processBeingTapped", "enabled", "minUsecLatency", "avgUsecLatency", "maxUsecLatency"}
)

type MacEventTapsModule struct{}
type EventTap struct {
	// typedef struct CGEventTapInformation
	// {
	//     uint32_t		eventTapID;
	//     CGEventTapLocation	tapPoint;		/* HID, session, annotated session */
	//     CGEventTapOptions	options;		/* Listener, Filter */
	//     CGEventMask		eventsOfInterest;	/* Mask of events being tapped */
	//     pid_t		tappingProcess;		/* Process that is tapping events */
	//     pid_t		processBeingTapped;	/* Zero if not a per-process tap */
	//     bool		enabled;		/* True if tap is enabled */
	//     float		minUsecLatency;		/* Minimum latency in microseconds */
	//     float		avgUsecLatency;		/* Average latency in microseconds */
	//     float		maxUsecLatency;		/* Maximum latency in microseconds */
	// } CGEventTapInformation;
	eventTapID         int64
	tapPoint           interface{}
	options            interface{}
	eventsOfInterest   interface{}
	tappingProcess     string
	processBeingTapped string
	enabled            bool
	minUsecLatency     float64
	avgUsecLatency     float64
	maxUsecLatency     float64
}

func (m MacEventTapsModule) Start(inst instance.Instance) error {
	err := m.eventtaps(inst)
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
	}
	return err
}

func (m MacEventTapsModule) eventtaps(inst instance.Instance) error {
	mw, err := datawriter.NewOrionWriter(moduleName, inst.GetOrionRuntime(), inst.GetOrionOutputFormat(), inst.GetOrionOutputFilepath())
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
		return err
	}
	values := [][]string{}
	count := 0
	taps := NSArrayEventTapToGoEventTapSlice(C.GetEventTapList())
	// for _, item := range NSArrayEventTapToGoEventTapSlice(C.GetEventTapList()) {
	// 	fmt.Println(item)
	// }
	for _, tap := range taps {
		var valmap = make(map[string]string)
		util.InitializeMapToEmptyString(valmap, header)

		// fmt.Println(tap)
		// header = []string{"eventTapID", "tapPoint", "options", "eventsOfInterest", "tappingProcess", "processBeingTapped", "enabled", "minUsecLatency", "avgUsecLatency", "maxUsecLatency"}
		valmap["eventTapID"] = strconv.FormatInt(tap.eventTapID, 10)
		valmap["tapPoint"] = fmt.Sprint(tap.tapPoint)
		valmap["options"] = fmt.Sprint(tap.options)
		valmap["eventsOfInterest"] = fmt.Sprint(tap.eventsOfInterest)
		valmap["tappingProcess"] = tap.tappingProcess
		valmap["processBeingTapped"] = tap.processBeingTapped
		valmap["enabled"] = strconv.FormatBool(tap.enabled)
		valmap["minUsecLatency"] = time.Duration(tap.minUsecLatency * 1000).String()
		valmap["avgUsecLatency"] = time.Duration(tap.avgUsecLatency * 1000).String()
		valmap["maxUsecLatency"] = time.Duration(tap.maxUsecLatency * 1000).String()

		// Convert valmap to entry and append to values
		entry, err := util.EntryFromMap(valmap, header)
		if err != nil {
			zap.L().Debug("Error formatting valmap as entry: "+err.Error(), zap.String("module", moduleName))
			continue
		}
		values = append(values, entry)
		count++
	}
	zap.L().Debug(fmt.Sprintf("Parsed [%d] event tap entries", count), zap.String("module", moduleName))

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

func NSArrayEventTapToGoEventTapSlice(arr *C.NSArray) []EventTap {

	var result []EventTap
	length := nsArrayLen(arr)

	for i := uint(0); i < length; i++ {
		// tapptr := nsArrayItem(arr, i)
		// fmt.Println(reflect.TypeOf(tapptr).String())

		tapdict := (*C.NSDictionary)(nsArrayItem(arr, i))
		// fmt.Println(tapdict)
		// fmt.Println((*tapdict))

		//  /*
		//  * Structure used to report information on event taps
		//  */
		// 	tapsResults[i] = @{
		//         @"path": tappingProcess,
		//         @"target": @(tap.processBeingTapped),
		//         @"enabled": @(tap.enabled),
		//         @"tapPoint": @(tap.tapPoint),
		//         @"eventTapID": @(tap.eventTapID),
		//         @"options": @(tap.options),
		//         @"eventsOfInterest": @(tap.eventsOfInterest),
		//         @"minUsecLatency": @(tap.minUsecLatency),
		//         @"avgUsecLatency": @(tap.avgUsecLatency),
		//         @"maxUsecLatency": @(tap.maxUsecLatency)};
		// }
		// fmt.Println("eventTapID", nsStringToGoString(C.NSDictionaryValueForKey(tapdict, goStringToNSString("eventTapID"))))
		// fmt.Println("tappingProcess", nsStringToGoString(C.NSDictionaryValueForKey(tapdict, goStringToNSString("tappingProcess"))))
		// fmt.Println("processBeingTapped", nsStringToGoString(C.NSDictionaryValueForKey(tapdict, goStringToNSString("processBeingTapped"))))
		// fmt.Println("enabled", nsStringToGoString(C.NSDictionaryValueForKey(tapdict, goStringToNSString("enabled"))))
		// fmt.Println("tapPoint", nsStringToGoString(C.NSDictionaryValueForKey(tapdict, goStringToNSString("tapPoint"))))
		// fmt.Println("options", nsStringToGoString(C.NSDictionaryValueForKey(tapdict, goStringToNSString("options"))))
		// fmt.Println("eventsOfInterest", nsStringToGoString(C.NSDictionaryValueForKey(tapdict, goStringToNSString("eventsOfInterest"))))
		// fmt.Println("min", nsStringToGoString(C.NSDictionaryValueForKey(tapdict, goStringToNSString("minUsecLatency"))))
		// fmt.Println("avg", nsStringToGoString(C.NSDictionaryValueForKey(tapdict, goStringToNSString("avgUsecLatency"))))
		// fmt.Println("max", nsStringToGoString(C.NSDictionaryValueForKey(tapdict, goStringToNSString("maxUsecLatency"))))
		var et EventTap = EventTap{}
		et.eventTapID, _ = strconv.ParseInt(nsStringToGoString(C.NSDictionaryValueForKey(tapdict, goStringToNSString("eventTapID"))), 10, 64)
		et.tappingProcess = nsStringToGoString(C.NSDictionaryValueForKey(tapdict, goStringToNSString("tappingProcess")))
		et.processBeingTapped = nsStringToGoString(C.NSDictionaryValueForKey(tapdict, goStringToNSString("processBeingTapped")))
		et.enabled, _ = strconv.ParseBool(nsStringToGoString(C.NSDictionaryValueForKey(tapdict, goStringToNSString("enabled"))))
		et.tapPoint = nsStringToGoString(C.NSDictionaryValueForKey(tapdict, goStringToNSString("tapPoint")))
		et.options = nsStringToGoString(C.NSDictionaryValueForKey(tapdict, goStringToNSString("options")))
		et.eventsOfInterest = nsStringToGoString(C.NSDictionaryValueForKey(tapdict, goStringToNSString("eventsOfInterest")))
		et.minUsecLatency, _ = strconv.ParseFloat(nsStringToGoString(C.NSDictionaryValueForKey(tapdict, goStringToNSString("minUsecLatency"))), 64)
		et.avgUsecLatency, _ = strconv.ParseFloat(nsStringToGoString(C.NSDictionaryValueForKey(tapdict, goStringToNSString("avgUsecLatency"))), 64)
		et.maxUsecLatency, _ = strconv.ParseFloat(nsStringToGoString(C.NSDictionaryValueForKey(tapdict, goStringToNSString("maxUsecLatency"))), 64)
		result = append(result, et)
	}
	return result
}

func nsArrayLen(arr *C.NSArray) uint                    { return uint(C.NSArrayLen(arr)) }
func nsArrayItem(arr *C.NSArray, i uint) unsafe.Pointer { return C.NSArrayItem(arr, C.ulong(i)) }
func goStringToNSString(str string) *C.NSString         { return C.CStringToNSString(C.CString(str)) }
func nsStringToGoString(s *C.NSString) string           { return C.GoString(nsStringToCString(s)) }
func nsStringToCString(s *C.NSString) *C.char {
	if s == nil {
		return nil
	}
	return C.NSStringToCString(s)
}
