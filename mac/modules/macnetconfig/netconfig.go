// +build darwin

package macnetconfig

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/anthonybm/Orion/datawriter"
	"github.com/anthonybm/Orion/instance"
	"github.com/anthonybm/Orion/util"
	"github.com/anthonybm/Orion/util/machelpers"
	"go.uber.org/zap"
)

type MacNetconfigModule struct {
}

var (
	moduleName  = "MacNetconfigModule"
	mode        = "mac"
	version     = "1.0"
	description = `
	Reads and parses the network config plist
	`
	author = "Anthony Martinez, martinez.anthonyb@gmail.com"
)

var (
	filepathAirportPreferencesPlist = "Library/Preferences/SystemConfiguration/com.apple.airport.preferences.plist"
	filepathNetworkInterfacesPlist  = "Library/Preferences/SystemConfiguration/NetworkInterfaces.plist"
)

func (m MacNetconfigModule) Start(inst instance.Instance) error {
	err := m.netconfig(inst)
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
	}
	return err
}

func (m MacNetconfigModule) netconfig(inst instance.Instance) error {
	header := []string{
		"type",
		"AddedAt",
		"Captive",
		"CaptiveBypass",
		"Disabled",
		"HiddenNetwork",
		"LastAutoJoinAt",
		"LastManualJoinAt",
		"NetworkWasCaptive",
		"Passpoint",
		"PersonalHotspot",
		"PossiblyHiddenNetwork",
		"RoamingProfileType",
		"SPRoaming",
		"SSID",
		"SSIDString",
		"SecurityType",
		"ShareMode",
		"SystemMode",
		"TemporarilyDisabled",
		"UserRole",
	}
	values := [][]string{}

	mw, err := datawriter.NewOrionWriter(moduleName, inst.GetOrionRuntime(), inst.GetOrionOutputFormat(), inst.GetOrionOutputFilepath())
	if err != nil {
		zap.L().Error("Error running " + moduleName + ": " + err.Error())
		return err
	}

	// Read and parse Airport Interfaces
	airportvalues, err := m.airport(inst)
	if err != nil {
		zap.L().Error("Failed to parse '" + filepathAirportPreferencesPlist + "': " + err.Error())
	}
	values = util.AppendToDoubleSlice(values, airportvalues)

	// Read and parse Network Interfaces
	networkinterfacevalues, err := m.networkinterface(inst)
	if err != nil {
		zap.L().Error("Failed to parse '" + filepathNetworkInterfacesPlist + "' " + err.Error())
	}
	values = util.AppendToDoubleSlice(values, networkinterfacevalues)

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

func (m MacNetconfigModule) networkinterface(inst instance.Instance) ([][]string, error) {
	// Read and parse NetworkInterface data
	networkInterfaceData, err := machelpers.DecodePlist(filepathNetworkInterfacesPlist, inst.GetTargetPath())
	if err != nil {
		return [][]string{}, errors.New("failed to decode '" + filepathNetworkInterfacesPlist + "': " + err.Error())
	}

	interfaces, err := machelpers.GetSingleValueInterfaceFromPlist(networkInterfaceData, "Interfaces") // Should be []map[string]interface{}
	if err != nil {
		return [][]string{}, errors.New("failed to decode 'Interfaces' key for '" + filepathNetworkInterfacesPlist + "': " + err.Error())
	}

	// Parse entries from Interfaces
	networkinterfacecount := 0
	values := [][]string{}

	for _, entry := range interfaces.([]interface{}) {
		// fmt.Println("Type: " + reflect.ValueOf(entry).String())
		// machelpers.PrintPlistAsJSON(entry)
		// panic("EXIT TEST")

		// machelpers.PrintPlistAsJSON(entry) // For debuging values of Plist
		// fmt.Println("Type: " + reflect.ValueOf(entry.(map[string]interface{})["AddedAt"]).String())
		// fmt.Println(fmt.Sprint(entry.(map[string]interface{})["AddedAt"]))

		var valmap = make(map[string]string)

		valmap["AddedAt"] = "N/A"
		valmap["Captive"] = "N/A"
		valmap["CaptiveBypass"] = "N/A"
		valmap["Disabled"] = "N/A"
		valmap["HiddenNetwork"] = "N/A"
		valmap["LastAutoJoinAt"] = "N/A"
		valmap["LastManualJoinAt"] = "N/A"
		valmap["NetworkWasCaptive"] = "N/A"
		valmap["Passpoint"] = "N/A"
		valmap["PersonalHotspot"] = "N/A"
		valmap["PossiblyHiddenNetwork"] = "N/A"
		valmap["RoamingProfileType"] = "N/A"
		valmap["SPRoaming"] = "N/A"
		valmap["SSID"] = "N/A"
		valmap["SSIDString"] = "N/A"
		valmap["SecurityType"] = "N/A"
		valmap["ShareMode"] = "N/A"
		valmap["SystemMode"] = "N/A"
		valmap["TemporarilyDisabled"] = "N/A"
		valmap["UserRole"] = "N/A"

		// Use 'ok' syntax to check if map contains key or not
		if addedAtVal, ok := entry.(map[string]interface{})["AddedAt"]; ok {
			valmap["AddedAt"] = addedAtVal.(time.Time).UTC().Format(time.RFC3339)
		}
		if captiveVal, ok := entry.(map[string]interface{})["Captive"]; ok {
			valmap["Captive"] = strconv.FormatBool(captiveVal.(bool))
		}
		if captiveBypassVal, ok := entry.(map[string]interface{})["CaptiveBypass"]; ok {
			valmap["CaptiveBypass"] = strconv.FormatBool(captiveBypassVal.(bool))
		}
		if disabledVal, ok := entry.(map[string]interface{})["Disabled"]; ok {
			valmap["Disabled"] = strconv.FormatBool(disabledVal.(bool))
		}
		if hiddenNetworkVal, ok := entry.(map[string]interface{})["HiddenNetwork"]; ok {
			valmap["HiddenNetwork"] = strconv.FormatBool(hiddenNetworkVal.(bool))
		}
		if lastAutoJoinAtVal, ok := entry.(map[string]interface{})["LastAutoJoinAt"]; ok {
			valmap["LastAutoJoinAt"] = lastAutoJoinAtVal.(time.Time).UTC().Format(time.RFC3339)
		}
		if lastManualJoinAtVal, ok := entry.(map[string]interface{})["LastManualJoinAt"]; ok {
			valmap["LastManualJoinAt"] = lastManualJoinAtVal.(time.Time).UTC().Format(time.RFC3339)
		}
		if networkWasCaptiveVal, ok := entry.(map[string]interface{})["NetworkWasCaptive"]; ok {
			valmap["NetworkWasCaptive"] = strconv.FormatBool(networkWasCaptiveVal.(bool))
		}
		if passpointVal, ok := entry.(map[string]interface{})["Passpoint"]; ok {
			valmap["Passpoint"] = strconv.FormatBool(passpointVal.(bool))
		}
		if personalHotspotVal, ok := entry.(map[string]interface{})["PersonalHotspot"]; ok {
			valmap["PersonalHotspot"] = strconv.FormatBool(personalHotspotVal.(bool))
		}
		if possiblyHiddenNetworkVal, ok := entry.(map[string]interface{})["PossiblyHiddenNetwork"]; ok {
			valmap["PossiblyHiddenNetwork"] = strconv.FormatBool(possiblyHiddenNetworkVal.(bool))
		}
		if roamingProfileTypeVal, ok := entry.(map[string]interface{})["RoamingProfileType"]; ok {
			valmap["RoamingProfileType"] = roamingProfileTypeVal.(string)
		}
		if spRoamingVal, ok := entry.(map[string]interface{})["SPRoaming"]; ok {
			valmap["SPRoaming"] = strconv.FormatBool(spRoamingVal.(bool))
		}
		if ssidVal, ok := entry.(map[string]interface{})["SSID"]; ok {
			valmap["SSID"] = fmt.Sprint(ssidVal)
		}
		if ssidStringVal, ok := entry.(map[string]interface{})["SSIDString"]; ok {
			valmap["SSIDString"] = fmt.Sprint(ssidStringVal)
		}
		if securityTypeVal, ok := entry.(map[string]interface{})["SecurityType"]; ok {
			valmap["SecurityType"] = fmt.Sprint(securityTypeVal)
		}
		if shareModeVal, ok := entry.(map[string]interface{})["ShareMode"]; ok {
			valmap["ShareMode"] = strconv.FormatUint(shareModeVal.(uint64), 10)
		}
		if systemModeVal, ok := entry.(map[string]interface{})["SystemMode"]; ok {
			valmap["SystemMode"] = strconv.FormatBool(systemModeVal.(bool))
		}
		if temporarilyDisabledVal, ok := entry.(map[string]interface{})["TemporarilyDisabled"]; ok {
			valmap["TemporarilyDisabled"] = strconv.FormatBool(temporarilyDisabledVal.(bool))
		}
		if userRoleVal, ok := entry.(map[string]interface{})["UserRole"]; ok {
			valmap["UserRole"] = strconv.FormatUint(userRoleVal.(uint64), 10)
		}

		if bsdName, ok := entry.(map[string]interface{})["BSD Name"]; ok {
			valmap["BSDName"] = fmt.Sprint(bsdName)
		}

		// TODO Fix names
		/*
			"Active": true,
			"BSD Name": "en0",
			"IOBuiltin": true,
			"IOInterfaceNamePrefix": "en",
			"IOInterfaceType": 6,
			"IOInterfaceUnit": 0,
			"IOMACAddress": "pIPnHx6z",
			"IOPathMatch": "IOService:/AppleACPIPlatformExpert/PCI0@0/AppleACPIPCI/RP01@1C/IOPP/ARPT@0/AppleBCMWLANBusInterfacePCIe/AppleBCMWLANCore/en0",
			"SCNetworkInterfaceInfo": {
				"UserDefinedName": "Wi-Fi"
			},
			"SCNetworkInterfaceType": "IEEE80211"
		*/

		networkInterfacesEntry := []string{
			valmap["BSDName"],
			valmap["AddedAt"],
			valmap["Captive"],
			valmap["CaptiveBypass"],
			valmap["Disabled"],
			valmap["HiddenNetwork"],
			valmap["LastAutoJoinAt"],
			valmap["LastManualJoinAt"],
			valmap["NetworkWasCaptive"],
			valmap["Passpoint"],
			valmap["PersonalHotspot"],
			valmap["PossiblyHiddenNetwork"],
			valmap["RoamingProfileType"],
			valmap["SPRoaming"],
			valmap["SSID"],
			valmap["SSIDString"],
			valmap["SecurityType"],
			valmap["ShareMode"],
			valmap["SystemMode"],
			valmap["TemporarilyDisabled"],
			valmap["UserRole"],
		}
		values = append(values, networkInterfacesEntry)
		networkinterfacecount++
	}

	//
	zap.L().Debug(fmt.Sprintf("Parsed [%d] networkinterface entries", networkinterfacecount), zap.String("module", moduleName))
	return values, nil
}

func (m MacNetconfigModule) airport(inst instance.Instance) ([][]string, error) {
	// Read and parse Airport data
	airportData, err := machelpers.DecodePlist(filepathAirportPreferencesPlist, inst.GetTargetPath())
	if err != nil {
		return [][]string{}, errors.New("failed to decode '" + filepathAirportPreferencesPlist + "': " + err.Error())
	}

	knownNetworks, err := machelpers.GetSingleValueInterfaceFromPlist(airportData, "KnownNetworks") // KnownNetworks is map[string]interface{}
	if err != nil {
		return [][]string{}, errors.New("failed to decode 'KnownNetworks' key for '" + filepathAirportPreferencesPlist + "': " + err.Error())
	}

	// Parse entries from KnownNetworks
	airportcount := 0
	values := [][]string{}
	for _, entry := range knownNetworks.(map[string]interface{}) { // knownNetworks should be map[string]interface{}
		// machelpers.PrintPlistAsJSON(entry) // For debuging values of Plist
		// fmt.Println("Type: " + reflect.ValueOf(entry.(map[string]interface{})["AddedAt"]).String())
		// fmt.Println(fmt.Sprint(entry.(map[string]interface{})["AddedAt"]))

		var valmap = make(map[string]string) // valmap stores values that will be placed in output entry
		valmap["AddedAt"] = ""
		valmap["Captive"] = ""
		valmap["CaptiveBypass"] = ""
		valmap["Disabled"] = ""
		valmap["HiddenNetwork"] = ""
		valmap["LastAutoJoinAt"] = ""
		valmap["LastManualJoinAt"] = ""
		valmap["NetworkWasCaptive"] = ""
		valmap["Passpoint"] = ""
		valmap["PersonalHotspot"] = ""
		valmap["PossiblyHiddenNetwork"] = ""
		valmap["RoamingProfileType"] = ""
		valmap["SPRoaming"] = ""
		valmap["SSID"] = ""
		valmap["SSIDString"] = ""
		valmap["SecurityType"] = ""
		valmap["ShareMode"] = ""
		valmap["SystemMode"] = ""
		valmap["TemporarilyDisabled"] = ""
		valmap["UserRole"] = ""

		// Use 'ok' syntax to check if map contains key or not
		if addedAtVal, ok := entry.(map[string]interface{})["AddedAt"]; ok {
			valmap["AddedAt"] = addedAtVal.(time.Time).UTC().Format(time.RFC3339)
		}
		if captiveVal, ok := entry.(map[string]interface{})["Captive"]; ok {
			valmap["Captive"] = strconv.FormatBool(captiveVal.(bool))
		}
		if captiveBypassVal, ok := entry.(map[string]interface{})["CaptiveBypass"]; ok {
			valmap["CaptiveBypass"] = strconv.FormatBool(captiveBypassVal.(bool))
		}
		if disabledVal, ok := entry.(map[string]interface{})["Disabled"]; ok {
			valmap["Disabled"] = strconv.FormatBool(disabledVal.(bool))
		}
		if hiddenNetworkVal, ok := entry.(map[string]interface{})["HiddenNetwork"]; ok {
			valmap["HiddenNetwork"] = strconv.FormatBool(hiddenNetworkVal.(bool))
		}
		if lastAutoJoinAtVal, ok := entry.(map[string]interface{})["LastAutoJoinAt"]; ok {
			valmap["LastAutoJoinAt"] = lastAutoJoinAtVal.(time.Time).UTC().Format(time.RFC3339)
		}
		if lastManualJoinAtVal, ok := entry.(map[string]interface{})["LastManualJoinAt"]; ok {
			valmap["LastManualJoinAt"] = lastManualJoinAtVal.(time.Time).UTC().Format(time.RFC3339)
		}
		if networkWasCaptiveVal, ok := entry.(map[string]interface{})["NetworkWasCaptive"]; ok {
			valmap["NetworkWasCaptive"] = strconv.FormatBool(networkWasCaptiveVal.(bool))
		}
		if passpointVal, ok := entry.(map[string]interface{})["Passpoint"]; ok {
			valmap["Passpoint"] = strconv.FormatBool(passpointVal.(bool))
		}
		if personalHotspotVal, ok := entry.(map[string]interface{})["PersonalHotspot"]; ok {
			valmap["PersonalHotspot"] = strconv.FormatBool(personalHotspotVal.(bool))
		}
		if possiblyHiddenNetworkVal, ok := entry.(map[string]interface{})["PossiblyHiddenNetwork"]; ok {
			valmap["PossiblyHiddenNetwork"] = strconv.FormatBool(possiblyHiddenNetworkVal.(bool))
		}
		if roamingProfileTypeVal, ok := entry.(map[string]interface{})["RoamingProfileType"]; ok {
			valmap["RoamingProfileType"] = roamingProfileTypeVal.(string)
		}
		if spRoamingVal, ok := entry.(map[string]interface{})["SPRoaming"]; ok {
			valmap["SPRoaming"] = strconv.FormatBool(spRoamingVal.(bool))
		}
		if ssidVal, ok := entry.(map[string]interface{})["SSID"]; ok {
			valmap["SSID"] = fmt.Sprint(ssidVal)
		}
		if ssidStringVal, ok := entry.(map[string]interface{})["SSIDString"]; ok {
			valmap["SSIDString"] = fmt.Sprint(ssidStringVal)
		}
		if securityTypeVal, ok := entry.(map[string]interface{})["SecurityType"]; ok {
			valmap["SecurityType"] = fmt.Sprint(securityTypeVal)
		}
		if shareModeVal, ok := entry.(map[string]interface{})["ShareMode"]; ok {
			valmap["ShareMode"] = strconv.FormatUint(shareModeVal.(uint64), 10)
		}
		if systemModeVal, ok := entry.(map[string]interface{})["SystemMode"]; ok {
			valmap["SystemMode"] = strconv.FormatBool(systemModeVal.(bool))
		}
		if temporarilyDisabledVal, ok := entry.(map[string]interface{})["TemporarilyDisabled"]; ok {
			valmap["TemporarilyDisabled"] = strconv.FormatBool(temporarilyDisabledVal.(bool))
		}
		if userRoleVal, ok := entry.(map[string]interface{})["UserRole"]; ok {
			valmap["UserRole"] = strconv.FormatUint(userRoleVal.(uint64), 10)
		}

		airportEntry := []string{
			"Airport",
			valmap["AddedAt"],
			valmap["Captive"],
			valmap["CaptiveBypass"],
			valmap["Disabled"],
			valmap["HiddenNetwork"],
			valmap["LastAutoJoinAt"],
			valmap["LastManualJoinAt"],
			valmap["NetworkWasCaptive"],
			valmap["Passpoint"],
			valmap["PersonalHotspot"],
			valmap["PossiblyHiddenNetwork"],
			valmap["RoamingProfileType"],
			valmap["SPRoaming"],
			valmap["SSID"],
			valmap["SSIDString"],
			valmap["SecurityType"],
			valmap["ShareMode"],
			valmap["SystemMode"],
			valmap["TemporarilyDisabled"],
			valmap["UserRole"],
		}
		values = append(values, airportEntry)
		airportcount++
	}
	zap.L().Debug(fmt.Sprintf("Parsed [%d] airport entries", airportcount), zap.String("module", moduleName))
	return values, nil
}
