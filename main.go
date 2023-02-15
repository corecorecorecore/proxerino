package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
  "strings"
  "log"
  "io/ioutil"
  "strconv"
)

func checkProxy(ip, port string, wg *sync.WaitGroup, file *os.File, semaphore chan bool) {
	defer wg.Done()
  log.SetOutput(ioutil.Discard)
	proxyUrl, err := url.Parse("http://" + port)
	if err != nil {
		fmt.Println("Error parsing proxy URL:", err)
		return
	}
	httpClient := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)},
		Timeout:   time.Second * 2,
	}
	proxyUrl.Host = ip + ":" + port
	response, err := httpClient.Get("http://ip-api.com/json/")
	if err != nil {
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return
	}
	var data struct {
		ASN     string `json:"as"`
		Country string `json:"country"`
    IP string `json:"query"`   
	}
  if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		return
	}  
	if data.IP == "" {
		return
	}
  if _, err := file.WriteString(fmt.Sprintf("%s:%s\n", ip, port)); err != nil {
		return
	}
	fmt.Printf("%s:%s (ASN: %s, Country: %s, Actual IP: %s)\n", ip, port, data.ASN, data.Country, data.IP)
  <-semaphore
}

func main() {
  var port string
  var output string
  var maxThreads int
  if len(os.Args) < 2 {
    fmt.Print("\033c")
    fmt.Println("\n"+"zmap -p port -q | ./scanner outputfile port <threads>"+"\n")
    fmt.Println(`masscan -p port1,port2 0.0.0.0/0 --rate=99999999 --exclude 255.255.255.255 | awk '{print $6":"$4}' | sed 's/\/tcp//g' | ./scanner outputfile <threads>`+"\n")
    fmt.Println("cat ips.txt | ./scanner outputfile port <threads>"+"\n")
    fmt.Println(`cat proxies.txt | ./scanner outputfile <threads>`)
    return
  }
  output = os.Args[1]
  if len(os.Args) >= 3 {
    if value, err := strconv.Atoi(os.Args[2]); err == nil {
      maxThreads = value
    }
  } else {
    maxThreads = 100
  }
	file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
    	os.Exit(1)
  }
  defer file.Close()
  threads := make(chan bool, maxThreads)
  scanner := bufio.NewScanner(os.Stdin)
  var wg sync.WaitGroup
  semaphore := make(chan struct{}, maxThreads)
  for scanner.Scan() {
    scanned := scanner.Text()
    if strings.Contains(scanned, ":") {
      proxy := strings.Split(scanned, ":")
      semaphore <- struct{}{}
      wg.Add(1)
      go func() {
        checkProxy(proxy[0], proxy[1], &wg, file,threads)
        <-semaphore
      }()
    } else {
      if len(os.Args) < 3 {
        fmt.Print("\033c")
        fmt.Println("\n"+"zmap -p port -q | ./scanner outputfile port <threads>"+"\n")
        fmt.Println(`masscan -p port1,port2 0.0.0.0/0 --rate=99999999 --exclude 255.255.255.255 | awk '{print $6":"$4}' | sed 's/\/tcp//g' | ./scanner outputfile <threads>`+"\n")
        fmt.Println("cat ips.txt | ./scanner outputfile port <threads>"+"\n")
        fmt.Println(`cat proxies.txt | ./scanner outputfile <threads>`)
        return
      }
      port = os.Args[2]
      semaphore <- struct{}{}
      wg.Add(1)
      go func() {
        checkProxy(scanned, port, &wg, file, threads)
        <-semaphore
      }()
    }
  }
  wg.Wait()
}
