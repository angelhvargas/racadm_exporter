package main

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"

	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const SensorTypeHeaderFormat = `Sensor Type :[[:space:]](\S+)`
const SensorHeaderFormat = `<([^>]+)>`
const SensorDataFormat = `([0-9A-Za-z\-\_][0-9A-Za-z \-\_%]+?)(?:\s{2,}|$|\s?\[[A-Z]\]$?)`
const PowerDataFormat = `([^>]+)_`
const Namespace = "racadm"

var SensorTypeHeaderFormatRegexp = regexp.MustCompile(SensorTypeHeaderFormat)
var SensorHeaderFormatRegexp = regexp.MustCompile(SensorHeaderFormat)
var SensorDataFormatRegexp = regexp.MustCompile(SensorDataFormat)
var PowerDataFormatRegexp = regexp.MustCompile(PowerDataFormat)
var replaceSpace = regexp.MustCompile(`\s`)

func main() {
	http.Handle("/metrics", promhttp.Handler())
}

func parseRacadmOutput(input string) map[string]int {
	scanner := bufio.NewScanner(strings.NewReader(input))
	var headerSensor = ""
	var metricsSensor = make(map[string]int)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 1 || line[0:1] == "[" {
			continue
		}

		if ok, match := getSensorsTypeHeaders(line); ok == true {
			fmt.Println(match)
		} else {
			continue
		}

		typeHeaders := SensorTypeHeaderFormatRegexp.FindAllStringSubmatch(line, -1)
		if len(typeHeaders) >= 1 {
			//fmt.Println(typeHeaders[0][1])
			headerSensor = typeHeaders[0][1]
			metricsSensor[headerSensor] = 1
			continue
		}

		if ok, match := getSensorHeaders(line); ok == true {
			fmt.Println(match)
		} else {
			continue
		}

		if ok, match := getSensorData(line); ok == true {
			fmt.Println(match)
		} else {
			continue
		}

	}
	return metricsSensor
}

func getSensorsTypeHeaders(input string) (ok bool, match string) {
	typeHeaders := SensorTypeHeaderFormatRegexp.FindAllStringSubmatch(input, -1)
	if len(typeHeaders) >= 1 {
		return true, typeHeaders[0][1]
	}
	return ok, match
}

func getSensorHeaders(input string) (ok bool, match []string) {
	header := SensorHeaderFormatRegexp.FindAllStringSubmatch(input, -1)
	if len(header) >= 1 {
		for _, value := range header {
			match = append(match, replaceSpace.ReplaceAllString(strings.ToLower(value[1]), `_`))
		}
		return true, match
	}
	return
}

func getSensorData(input string) (ok bool, match []string) {
	sensorData := SensorDataFormatRegexp.FindAllStringSubmatch(input, -1)
	if len(sensorData) >= 1 {
		for _, value := range sensorData {
			match = append(match, replaceSpace.ReplaceAllString(strings.ToLower(value[1]), `_`))
		}
		return true, match
	}
	return
}

func metricsPower(match []string, headers []string) (metric *prometheus.GaugeVec) {
	if len(match) != len(headers) {
		return
	}
	psuLabel := PowerDataFormatRegexp.FindAllStringSubmatch(match[0], -1)
	if len(psuLabel) < 1 {
		return
	}
	var isPresent float64 = 0
	var metricsLabels = make(prometheus.Labels)
	metricsLabels[headers[1]] = match[1]
	metricsLabels["PSU"] = psuLabel[0][1]
	metric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:      "power_" + headers[0],
		Namespace: Namespace,
	}, []string{headers[1], "PSU"})
	// Set psu status if !present = 1 , present is default so 0
	if psuLabel[0][1] != "present" {
		isPresent = 1
	}
	metric.WithLabelValues(metricsLabels[headers[1]], metricsLabels["PSU"]).Set(isPresent)
	return
}
