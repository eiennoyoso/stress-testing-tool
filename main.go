package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/url"
	"os"
	"sync"
	"time"
)

type ConcurrentCounter struct {
	count int
	mutex sync.RWMutex
}

func main() {
	var httpMethod = flag.String("httpMethod", "GET", "HTTP Method: GET or POST")
	var rawurl = flag.String("url", "", "URL")
	var postData = flag.String("postData", "", "POST data")
	var maxConcurrentRequestCount = flag.Int("concurrent", 10, "Concurent request count")

	flag.Parse()

	log.SetOutput(os.Stderr)

	u, err := url.Parse(*rawurl)

	if err != nil {
		log.Fatalln(err)
	}

	var currentConcurrentRequestCounter = ConcurrentCounter{count: 0}
	concurrentRequestCountChannel := make(chan int)
	go listenRequestSent(&currentConcurrentRequestCounter, concurrentRequestCountChannel)

	for {
		if currentConcurrentRequestCounter.count == *maxConcurrentRequestCount {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		currentConcurrentRequestCounter.mutex.Lock()
		currentConcurrentRequestCounter.count = currentConcurrentRequestCounter.count + 1
		currentConcurrentRequestCounter.mutex.Unlock()

		go fetch(
			*httpMethod,
			u,
			postData,
			concurrentRequestCountChannel,
		)
	}
}

func listenRequestSent(
	currentConcurrentRequestCounter *ConcurrentCounter,
	concurrentRequestCountChannel chan int,
) {
	for {
		<-concurrentRequestCountChannel

		currentConcurrentRequestCounter.mutex.Lock()
		currentConcurrentRequestCounter.count = currentConcurrentRequestCounter.count - 1
		currentConcurrentRequestCounter.mutex.Unlock()
	}
}

func fetch(
	httpMethod string,
	u *url.URL,
	postData *string,
	concurrentRequestCountChannel chan int,
) error {
	var port = u.Port()
	if port == "" {
		switch u.Scheme {
		case "http":
			port = "80"
		case "https":
			port = "443"
		default:
			return errors.New("Unsupported schema")
		}
	}

	var dialer = net.Dialer{Timeout: 1 * time.Second}
	conn, err := dialer.Dial("tcp", u.Hostname() + ":" + port)
	if err != nil {
		log.Println(err)
		return err
	}

	var requestUri = u.RequestURI()
	if u.RawQuery != "" {
		requestUri = fmt.Sprintf("%s&rand=%d", requestUri, rand.Intn(10))
	} else {
		requestUri = fmt.Sprintf("%s?rand=%d", requestUri, rand.Intn(10))
	}

	var headers = "Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8\n" //Accept-Language: en-us,en;q=0.5\nAccept-Encoding: gzip,deflate\nAccept-Charset: ISO-8859-1,utf-8;q=0.7,*;q=0.7\nKeep-Alive: 115\nConnection: keep-alive\n"
	headers = headers + "User-agent: Mozilla/5.0 (compatible; MSIE 9.0; Windows NT 6.0) Opera 12.14\n"
	if httpMethod == "POST" && postData != nil && *postData != "" {
		headers = fmt.Sprintf("%sContent-Length: %d\n", headers, len(*postData))
		headers = headers + "Content-type: application/x-www-form-urlencoded\n"
	}

	var payload = ""
	if httpMethod == "POST" && postData != nil && *postData != ""  {
		payload = "\n" + *postData + "\n"
	}

	request := fmt.Sprintf(
		"%s %s HTTP/1.0\nHost: %s\n%s%s\n",
		httpMethod,
		requestUri,
		u.Hostname(),
		headers,
		payload,
	)

	_, err = conn.Write([]byte(request))
	if err != nil {
		log.Fatalln("Error sending request", err)
	}

	concurrentRequestCountChannel <- 1

	var reader = bufio.NewReader(conn)
	response, err := reader.ReadString('\n')

	if err != nil {
		log.Println("Error reading response", err)
	} else {
		log.Println(string(response))
	}

	conn.Close()

	return nil
}