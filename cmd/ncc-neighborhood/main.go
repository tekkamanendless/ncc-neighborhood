package main

import (
	"crypto/tls"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	var rootCommand = &cobra.Command{
		Use:   "ncc-neighborhood [<subdivision> [...]]",
		Short: "New Castle County neighborhood parcel search",
		Long: `
This tool downloads the parcel data for the given addresses and prints a CSV of the data.

Examples:
   List all of the parcels in Green Valley.
      ncc-neighborhood 'GREEN VALLEY 1' 'GREEN VALLEY 2A' 'GREEN VALLEY 2B' 'GREEN VALLEY 3' 'GREEN VALLEY V'

   List all of the parcels in Green Valley.
      ncc-neighborhood 'COTSWOLD HILLS'
`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			logLevelString, _ := cmd.Flags().GetString("log-level")
			logLevel, err := logrus.ParseLevel(logLevelString)
			if err != nil {
				logrus.Errorf("Could not parse log level string %q: %v", logLevelString, err)
				os.Exit(1)
			}
			logrus.SetLevel(logLevel)

			outputFilename, _ := cmd.Flags().GetString("output-filename")
			if outputFilename == "" {
				logrus.Errorf("Output filename cannot be empty (use '-' for stdout)")
				os.Exit(1)
			}

			http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

			subdivisions := args
			if len(subdivisions) == 0 {
				logrus.Errorf("No subdivisions given")
				os.Exit(1)
			}

			houses := []map[string]string{}
			for _, subdivision := range subdivisions {
				logrus.Infof("Working on subdivision: %s", subdivision)
				results, err := querySubdivision(subdivision)
				if err != nil {
					logrus.Warnf("Could not query for subdivision %q: %s", subdivision, err.Error())
					continue
				}
				logrus.Infof("Found %d homes (%s).", len(results), subdivision)
				houses = append(houses, results...)
			}
			logrus.Infof("Found %d homes (total).", len(houses))

			sort.SliceStable(houses, func(i int, j int) bool {
				diff := strings.Compare(houses[i]["STNAME"], houses[j]["STNAME"])
				if diff != 0 {
					return diff < 0
				}

				leftInt, err := strconv.ParseInt(houses[i]["STNO"], 10, 32)
				if err != nil {
					return true
				}
				rightInt, err := strconv.ParseInt(houses[j]["STNO"], 10, 32)
				if err != nil {
					return true
				}
				return leftInt < rightInt
			})

			csvData := [][]string{}
			csvHeader := []string{}
			{
				csvHeaderFields := map[string]bool{}
				for _, house := range houses {
					for key := range house {
						csvHeaderFields[key] = true
					}
				}

				for key := range csvHeaderFields {
					csvHeader = append(csvHeader, key)
				}
				sort.Strings(csvHeader)
			}
			csvData = append(csvData, csvHeader)
			for _, house := range houses {
				houseLine := make([]string, len(csvHeader))
				for index, key := range csvHeader {
					houseLine[index] = house[key]
				}

				csvData = append(csvData, houseLine)
			}

			var writer io.Writer
			if outputFilename == "-" {
				logrus.Infof("Writing to stdout.")

				writer = os.Stdout
			} else {
				logrus.Infof("Writing to %q.", outputFilename)

				fileHandle, err := os.OpenFile(outputFilename, os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					logrus.Errorf("Could not open file %q: %v", outputFilename, err)
					os.Exit(1)
				}
				defer fileHandle.Close()

				writer = fileHandle
			}

			csvWriter := csv.NewWriter(writer)
			_ = csvWriter.WriteAll(csvData)
		},
	}
	rootCommand.PersistentFlags().String("log-level", "info", "The log level; this may be one of:\n* error\n* warn\n* info\n* debug\n* trace\n")
	rootCommand.PersistentFlags().String("output-filename", "-", "Where to write the results; use '-' for stdout")

	err := rootCommand.Execute()
	if err != nil {
		logrus.Errorf("Error: %v", err)
		os.Exit(1)
	}
}

type MapFindResponse struct {
	Results []struct {
		Attributes map[string]string `json:"attributes"`
	} `json:"results"`
	Error struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Details interface{} `json:"details"`
	}
}

func querySubdivision(subdivision string) ([]map[string]string, error) {
	baseUrl := "https://gis.nccde.org/agsserver/rest/services/BaseMaps/Base_Layers/MapServer/find"
	parameters := map[string]string{}
	parameters["searchText"] = subdivision
	parameters["contains"] = "false"
	parameters["searchFields"] = "SUBDIV"

	parameters["sr"] = ""
	parameters["layers"] = "0"
	parameters["layerDefs"] = ""
	parameters["returnGeometry"] = "false"
	parameters["maxAllowableOffset"] = ""
	parameters["geometryPrecision"] = ""
	parameters["dynamicLayers"] = ""
	parameters["returnZ"] = "false"
	parameters["returnM"] = "false"
	parameters["gdbVersion"] = ""
	parameters["f"] = "pjson"

	parts := []string{}
	for key, value := range parameters {
		part := url.QueryEscape(key) + "=" + url.QueryEscape(value)
		parts = append(parts, part)
	}

	fullUrl := baseUrl
	if len(parts) > 0 {
		fullUrl += "?" + strings.Join(parts, "&")
	}

	logrus.Debugf("GET %s", fullUrl)
	response, err := http.Get(fullUrl)
	if err != nil {
		return nil, fmt.Errorf("Could not perform request: %v", err)
	}

	logrus.Debugf("Status code: %d", response.StatusCode)
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Got back status code %d", response.StatusCode)
	}

	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("Could not read response body: %v", err)
	}

	var mapFindResponse MapFindResponse
	err = json.Unmarshal(contents, &mapFindResponse)
	if err != nil {
		return nil, fmt.Errorf("Could not parse response: %v", err)
	}

	if mapFindResponse.Error.Code > 299 {
		return nil, fmt.Errorf("Got back status code %d: %s", mapFindResponse.Error.Code, mapFindResponse.Error.Message)
	}

	results := []map[string]string{}
	for _, mapFindResponseResult := range mapFindResponse.Results {
		attributes := map[string]string{}
		for key, value := range mapFindResponseResult.Attributes {
			attributes[key] = fixValue(value)
		}
		results = append(results, attributes)
	}

	return results, nil
}

func fixValue(value string) string {
	// Eliminate redundant whitespace.
	value = strings.Join(strings.Fields(value), " ")
	value = strings.Trim(value, " ")
	return value
}
