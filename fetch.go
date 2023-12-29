package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

/*
YAML file being parsed:

	name (string, required) - A free-text name to describe the HTTP endpoint.

	url (string, required) - The URL of the HTTP endpoint.
	You may assume that the URL is always a valid HTTP or HTTPS address.

	method (string, optional) - The HTTP method of the endpoint.
	If this field is present, you may assume it's a valid HTTP method (e.g. GET, POST, etc.).
	If this field is omitted, the default is GET.

	headers (dictionary, optional) - The HTTP headers to include in the request.
	If this field is present, you may assume that the keys and values of this dictionary
	are strings that are valid HTTP header names and values.
	If this field is omitted, no headers need to be added to or modified in the HTTP
	request.

	body (string, optional) - The HTTP body to include in the request.
	If this field is present, you should assume it's a valid JSON-encoded string. You
	do not need to account for non-JSON request bodies.
	If this field is omitted, no body is sent in the request.
*/

// YAML config file parsed data
type HealthCheck struct {
	Body     string            `yaml:"body,omitempty"`
	Headers  map[string]string `yaml:"headers,omitempty"`
	Method   string            `yaml:"method,omitempty"`
	Name     string            `yaml:"name"`
	URL      string            `yaml:"url"`
	hostname string            `yaml:"-"`
}

// Data structure for tracking percent of uptime for domains of a URL
// Hostname with two additional k/v: "count":int, "success":int
//   - count:   number of connection attempts
//   - success: number of succcessful attempts
// ["fetch.com"][string]int{ count:0, success:0 }
var HealthCheckStatus = make(map[string]map[string]int)

// HTTP Request timeout set in milliseconds
var responseTimeout int = 500

// Output timeout set in seconds
var outputTimeout int = 15

func main() {

	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <configFile.yaml>\n", os.Args[0])
		os.Exit(-1)
	}

	yamlConfigFile := os.Args[1]
	yamlFile, err := ioutil.ReadFile(yamlConfigFile)
	if err != nil {
		fmt.Printf("Error: Unable to open yaml config file: %s ", err)
		os.Exit(-1)
	}

	var healthcheck []HealthCheck
	err = yaml.Unmarshal(yamlFile, &healthcheck)
	if err != nil {
		fmt.Printf("Error: Unable to unmarshal/parse yaml config: %s", err)
		os.Exit(-1)
	}

	// Loop through the yaml file
	for i, hc := range healthcheck {

		// Sanity checks
		if hc.Name == "" {
			fmt.Printf("Error: Required name not found\n")
			os.Exit(-1)
		}
		if hc.URL == "" {
			fmt.Printf("Error: Required URL not found\n")
			os.Exit(-1)
		}

		// Get the subdomain.domain.whatever
		// e.g.: http://www.foo.com -> www.foo.com
		address, err := url.Parse(hc.URL)
		if err != nil {
			fmt.Printf("Error: Cant parse URL: %s", hc.URL)
			os.Exit(-1)
		}
		hostname := address.Hostname()
		healthcheck[i].hostname = hostname
		// Here we setup a little data structure to track the status of the URL domains
		HealthCheckStatus[hostname] = map[string]int{"count": 0, "success": 0}

	}

	for {
		for _, hc := range healthcheck {
			success := check(hc)
			HealthCheckStatus[hc.hostname]["count"]++
			if success {
				HealthCheckStatus[hc.hostname]["success"]++
			}
		}

		// Output percentage of uptime for the domains of each URL
		// Formula: 100 * (number of HTTP requests that had an outcome of UP / number of HTTP requests)
		for host, _ := range HealthCheckStatus {
			success := HealthCheckStatus[host]["success"]
			count := HealthCheckStatus[host]["count"]
			uptime := int(math.Round(100 * (float64(success) / float64(count))))
			//log.Printf("%s has %d%% availablity percentage (count: %d) (success: %d)\n", host, uptime, count, success)
			fmt.Printf("%s has %d%% availablity percentage\n", host, uptime)
		}

		// Delay polling
		time.Sleep(time.Duration(outputTimeout) * time.Second)

	}
}

func check(site HealthCheck) bool {

	method := "GET"
	if site.Method != "" {
		method = site.Method
	}

	// HTTP Client with timeout defined above as global var
	// Set timeout defined above as global variable
	client := http.Client{
		Timeout: time.Duration(responseTimeout) * time.Millisecond,
	}

	req, err := http.NewRequest(method, site.URL, bytes.NewBufferString(site.Body))

	if site.Headers != nil {
		for k, v := range site.Headers {
			req.Header.Add(k, v)
		}
	}

	if err != nil {
		return false
	}

	//fmt.Printf("Attempting to connect to: %s\n", site.URL)
	resp, err := client.Do(req)

	if err != nil {
		return false
	}

	defer resp.Body.Close()
	// Response code must be between 200 and 299 otherwise it is considered down
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		return true
	}

	return false
}
