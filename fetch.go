/*
 fetch HTTP HealthCheck

 Author:
   kyle.lafkoff@gmail.com

 Usage:
   go run fetch.go fetch.yaml

   go build fetch.go
   ./fetch fetch.yaml

 About:
   The fetch HTTP HealthCheck program will attempt to connect to the sites
   defined in a yaml file every 15 seconds and report back if UP or DOWN,
   with a percentage of uptime.

 Criteria for UP:
   1. 2xx HTTP Response code
   2. Response returns within the 500ms threshold

 See README.md for information on installing dependencies
*/

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"sync"
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

// Result is the data structure to store the history of attempts
type Result struct {
	Attempt float64
	Success float64
}

// Calculate successful percentage of uptime for the domains of each URL
func (r Result) Uptime() int {
	if r.Attempt == 0 {
		return 0
	}
	return int(math.Round(100 * (r.Success / r.Attempt)))
}

// Thread-safe structure for tracking percent uptime of domains
type Results struct {
	lock  sync.Locker
	Sites map[string]*Result
}

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

	status := &Results{
		lock:  new(sync.Mutex),
		Sites: make(map[string]*Result),
	}

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
		healthcheck[i].hostname = address.Hostname()
		status.Sites[healthcheck[i].hostname] = new(Result)
	}

	for {
		wg := new(sync.WaitGroup)
		wg.Add(len(healthcheck))

		for _, hc := range healthcheck {
			go func(hc HealthCheck) {
				success := check(hc)
				status.lock.Lock()
				status.Sites[hc.hostname].Attempt++
				if success {
					status.Sites[hc.hostname].Success++
				}
				status.lock.Unlock()
				wg.Done()
			}(hc)
		}
		wg.Wait()

		// Output percentage of uptime for the domains of each URL
		for host, res := range status.Sites {
			fmt.Printf("%s has %d%% availablity percentage\n", host, res.Uptime())
		}

		// Delay polling
		time.Sleep(time.Duration(outputTimeout) * time.Second)
	}
}

// Simple HTTP request function
func check(site HealthCheck) bool {

	// HTTP Client with timeout defined above as global variable responseTimeout
	client := http.Client{
		Timeout: time.Duration(responseTimeout) * time.Millisecond,
	}

	method := "GET"
	if site.Method != "" {
		method = site.Method
	}

	req, err := http.NewRequest(method, site.URL, bytes.NewBufferString(site.Body))
	if err != nil {
		return false
	}

	// Add The headers
	if site.Headers != nil {
		for k, v := range site.Headers {
			req.Header.Add(k, v)
		}
	}

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
