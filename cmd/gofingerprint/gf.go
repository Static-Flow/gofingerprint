package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"sync"
	"time"
)

type Fingerprint struct {
	//identifier of the fingerprint e.g. JIRA,Tomcat,AEM,etc
	Name string `json:"name"`
	//the actual string used to fingerprint a service or application
	Fingerprint string `json:"fingerprint"`
	//the expected response code
	ExpectedResonse int `json:"code"`
}

func matcher(response string, fingerprints []Fingerprint) (Fingerprint, bool) {
	for _, fingerprint := range fingerprints {
		if strings.Contains(response, strings.ToLower(fingerprint.Fingerprint)) {
			return fingerprint, true
		}
	}
	return Fingerprint{}, false
}

func fetcher(host string, path string) (string, error) {
	//normalize host and path so we don't get host//path situations
	if !strings.HasPrefix(host, "https") {
		host = "https://" + host
	}
	if host[len(host)-1] == '/' {
		if path[0] == '/' {
			host = host + path[1:]
		} else {
			host = host + path
		}
	} else {
		if path[0] == '/' {
			host = host + path
		} else {
			host = host + "/" + path
		}
	}
	resp, err := http.Get(host)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	responseString, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return "", err
	}
	return strings.ToLower(string(responseString)), nil
}

func main() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	var wg sync.WaitGroup
	domainsToSearch := make(chan string)
	matchBuckets := make(map[string][]string)
	var fingerprints []Fingerprint
	var pathToFetch string
	badPath := flag.String("path", "", "The path to hit each target with to get a response.")
	fingerprintFile := flag.String("fingerprints", "", "JSON file containing fingerprints to search for.")
	workers := flag.Int("workers", 20, "Number of workers to process urls")
	outputDir := flag.String("output", "./", "Directory to output files")
	timeoutPtr := flag.Int("timeout", 10, "timeout for connecting to servers")
	debug := flag.Bool("debug", false, "Enable to see any errors with fetching targets")
	flag.Parse()
	http.DefaultClient.Timeout = time.Duration(*timeoutPtr) * time.Second
	if pathToFetch = *badPath; len(pathToFetch) == 0 {
		log.Fatalln("-badpath flag must be set.")
	}

	jsonFile, err := os.Open(*fingerprintFile)
	if err != nil {
		log.Fatalln(err)
	}
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)
	if err := json.Unmarshal(byteValue, &fingerprints); err != nil {
		log.Fatalf("Error parsing JSON. Check that it is compliant. \n %s \n", err)
	}

	for i := 0; i < *workers; i++ {
		wg.Add(1)
		/*
			This following goroutine is where the magic happens
			It pulls domains from the group, sends a GET request, then checks the headers and body for the fingerprints
			described by the supplied JSON and moves the domain in the matching bucket if a match is found
		*/
		go func(fingerprintContainers map[string][]string) {
			for domain := range domainsToSearch {
				responseString, err := fetcher(domain, pathToFetch)
				if err == nil {
					matchedFingerprint, matchFound := matcher(responseString, fingerprints)
					if matchFound {
						log.Println(matchedFingerprint.Name + " found at " + domain)
						fingerprintContainers[matchedFingerprint.Name] = append(matchBuckets[matchedFingerprint.Name], domain)
					}
				} else {
					if *debug {
						println(err.Error())
					}
				}
			}
			wg.Done()
		}(matchBuckets)
	}
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		domainsToSearch <- s.Text()
	}
	close(domainsToSearch)
	wg.Wait()
	fmt.Println("Writing results to fingerprint files")

	outputDirectory := *outputDir
	if !strings.HasSuffix(*outputDir, "/") {
		outputDirectory += "/"
	}

	if _, err := os.Stat(outputDirectory); os.IsNotExist(err) {
		errDir := os.MkdirAll(outputDirectory, 0755)
		if errDir != nil {
			log.Fatal(err)
		}

	}
	for fingerprint := range matchBuckets {
		f, err := os.Create(outputDirectory + fingerprint + ".txt")
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		for _, fingerprintedDomain := range matchBuckets[fingerprint] {
			_, err := f.WriteString(fingerprintedDomain + "\n")
			if err != nil {
				fmt.Println(err.Error())
				f.Close()
				return
			}
		}
		err = f.Close()
		if err != nil {
			fmt.Println(err.Error())
			return
		}
	}
}
