package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gocolly/colly"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

var debug bool
var JobQueue chan *Job
var client http.Client

type Job struct {
	Target    *Target
	Path      string
	Method    string
	Data      string
	Collector colly.Collector
}

func NewJob(target Target, badPath string, requestMethod string, requestBody string) Job {
	collector := colly.NewCollector(
		colly.MaxDepth(1),
	)
	collector.AllowURLRevisit = true
	collector.WithTransport(&http.Transport{ // Add this line
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	})
	collector.ParseHTTPErrorResponse = true
	collector.OnHTML("title", func(element *colly.HTMLElement) {
		if len(element.Text) > 49 {
			target.Title = element.Text[:49]
		} else {
			target.Title = element.Text
		}
	})
	collector.OnResponse(func(response *colly.Response) {
		target.Status = response.StatusCode
		target.Body = string(response.Body)
	})
	return Job{&target, badPath, requestMethod, requestBody, *collector}
}

type Target struct {
	Domain          string
	Ip              string
	Port            string
	FingerprintName string
	Title           string
	Status          int
	Body            string
}

type Worker struct {
	JobChannel   chan Job
	Fingerprints []Fingerprint
	WorkGroup    *sync.WaitGroup
	Colly        colly.Collector
}

func NewWorker(jobs chan Job, fingerprints []Fingerprint, wg *sync.WaitGroup) Worker {
	return Worker{
		JobChannel:   jobs,
		Fingerprints: fingerprints,
		WorkGroup:    wg,
	}
}

func (w Worker) Start2() {
	w.WorkGroup.Add(1)
	go func(wgi *sync.WaitGroup) {
		for {
			select {
			case job := <-w.JobChannel:
				if job == nil {
					wgi.Done()
					return
				} else {
					job.Fetch()
					for _, fingerprint := range w.Fingerprints {
						if len(job.Target.FingerprintName) == 0 {
							for _, search := range fingerprint.Fingerprints {
								if matched, _ := regexp.MatchString(search, job.Target.Body); matched {
									job.Target.FingerprintName = fingerprint.Name
									fmt.Printf("https://%s:%s\n", job.Target.Domain, fingerprint.Name)
									break
								}
							}
						} else {
							break
						}
					}

					if len(job.Target.FingerprintName) == 0 && debug {
						fmt.Println("No Match")
					}
				}
			}
		}
	}(w.WorkGroup)
}

func NewTarget(targetParts string) Target {
	targetPieces := strings.Split(targetParts, ",")
	return Target{strings.ReplaceAll(targetPieces[0], "\"", ""),
		strings.ReplaceAll(targetPieces[1], "\"", ""),
		strings.ReplaceAll(targetPieces[2], "\"", ""), "", "", 0, ""}
}

type Fingerprint struct {
	//identifier of the fingerprint e.g. JIRA,Tomcat,AEM,etc
	Name string `json:"name"`
	//the actual string used to fingerprint a service or application
	Fingerprints []string `json:"fingerprint"`
}

func genBadPath() string {
	seededRand := rand.New(
		rand.NewSource(time.Now().UnixNano()))
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 8)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return "/" + string(b)
}

func main() {
	wg := sync.WaitGroup{}
	var fingerprints []Fingerprint
	var badPath string
	badpathPtr := flag.String("badpath", "", "The intentional 404 path to hit each target with to get a response.")
	fingerprintFile := flag.String("fingerprints", "", "JSON file containing fingerprints to search for.")
	timeoutPtr := flag.Int("timeout", 10, "timeout for connecting to servers")
	workers := flag.Int("workers", 20, "Number of workers to process urls")
	queuePtr := flag.Int("queue", 100000, "Queue size to pool jobs")
	targetPtr := flag.String("target", "", "Target file with hosts to fingerprint")
	methodPtr := flag.String("method", "GET", "which HTTP request to make the request with")
	bodyPtr := flag.String("body", "", "Data to send in the request body")
	debugPtr := flag.Bool("debug", false, "Enable to see any errors with fetching targets")
	flag.Parse()
	badPath = *badpathPtr
	if len(badPath) == 0 {
		badPath = genBadPath()
	}
	JobQueue = make(chan *Job, *queuePtr)
	debug = *debugPtr

	dialer := net.Dialer{
		Timeout:   time.Duration(*timeoutPtr) * time.Second,
		KeepAlive: time.Duration(*timeoutPtr) * time.Second,
	}

	defaultTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			CipherSuites:       nil,
			MaxVersion:         tls.VersionTLS13,
		},
		DialContext:           dialer.DialContext,
		MaxIdleConns:          100000,
		MaxIdleConnsPerHost:   2,
		IdleConnTimeout:       time.Duration(*timeoutPtr) * time.Second,
		ResponseHeaderTimeout: time.Duration(*timeoutPtr) * time.Second,
	}
	client = http.Client{
		Transport: defaultTransport,
		Timeout:   time.Duration(*timeoutPtr) * time.Second,
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
		worker := NewWorker(JobQueue, fingerprints, &wg)
		worker.Start2()
	}
	fmt.Printf("Started %d workers!\n", *workers)
	inputFile, err := os.Open(*targetPtr)
	if err != nil {
		fmt.Println(err)
	}
	defer inputFile.Close()
	scanner := bufio.NewScanner(inputFile)
	for scanner.Scan() {

		// let's create a job with the payload
		//Push the work onto the queue. Blocking when we've filled up the queue

		scheduleJob(NewJob(NewTarget(scanner.Text()), badPath, *methodPtr, *bodyPtr))
		//Push the work onto the queue. Blocking when we've filled up the queue
		scheduleJob(NewJob(NewTarget(scanner.Text()), "/", *methodPtr, *bodyPtr))

	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	close(JobQueue)
	fmt.Println("Done ingesting jobs")
	wg.Wait()
	fmt.Println("Done")
}

func scheduleJob(work Job) {
	done := false
	for {
		if !done {
			select {
			case JobQueue <- &work:
				done = true
			default:
				if debug {
				}
			}
		} else {
			break
		}
	}
}
