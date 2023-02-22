package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
  "strconv"
	"time"
)

type ProxyData struct {
	ASN     string `json:"as"`
	Country string `json:"country"`
	IP      string `json:"query"`
}
var totalIPs, successCount, failureCount int

const (
    yellow = "\033[33m"
    blue   = "\033[34m"
    green  = "\033[32m"
    red    = "\033[31m"
    reset  = "\033[0m"
)


func checkProxy(ip, port string, timeout time.Duration, file *os.File, limit chan struct{}) bool {
	defer func() {
		<-limit
	}()
  success := false

	proxyUrl, err := url.Parse("http://" + port)
	if err != nil {
		fmt.Println("Error parsing proxy URL:", err)
		return false
	}

	httpClient := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)},
		Timeout:   timeout,
	}

	proxyUrl.Host = ip + ":" + port
	response, err := httpClient.Get("http://ip-api.com/json/")
	if err != nil {
		return false
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return false
	}

	var data ProxyData
	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		return false
	}

	if data.IP != "" {
        if _, err := file.WriteString(fmt.Sprintf("%s:%s\n", ip, port)); err != nil {
            return false
        }
        fmt.Printf("%s:%s (ASN: %s, Country: %s, Actual IP: %s)\n", ip, port, data.ASN, data.Country, data.IP)
        success = true
    }
    return success
}

func handlePanic() {
    if r := recover(); r != nil {
        fmt.Printf("%sUsage:%s zmap -p port -q | %s./scanner%s output-file max-threads timeout update-timeout port\n", yellow, reset, green, reset)
    fmt.Printf("%sOR%s\n", blue, reset)
    fmt.Printf("%sUsage:%s cat input-file | %s./scanner%s output-file max-threads timeout update-timeout\n", yellow, reset, green, reset)
        return
    }
}

func main() {
    defer handlePanic()
    output := os.Args[1]
    maxThreads, _ := strconv.Atoi(os.Args[2])
    timeoutSeconds, _ := strconv.Atoi(os.Args[3])
    timeout := time.Duration(timeoutSeconds) * time.Second
    updateSeconds, _ := strconv.Atoi(os.Args[4])
    currentThreads := 0
    limit := make(chan struct{}, maxThreads)
    ticker := time.NewTicker(time.Second * time.Duration(updateSeconds))
    ip := ""
    port := ""
    
    file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil {
        fmt.Println("Error opening file:", err)
        os.Exit(1)
    }
    defer file.Close()

    processed := make(map[string]bool)
    
    go func() {
        for range ticker.C {
            fmt.Printf("%sCurrent threads:%s %s%d%s, %sIPs processed:%s %s%d%s, %sSuccesses:%s %s%d%s, %sFailures:%s %s%d%s\n",
                yellow, reset, yellow, currentThreads, reset, blue, reset, blue, totalIPs, reset,
                green, reset, green, successCount, reset,
                red, reset, red, failureCount, reset,
            )
        }
    }()
    
    scanner := bufio.NewScanner(os.Stdin)
    if err := scanner.Err(); err != nil {
        fmt.Fprintln(os.Stderr, "reading standard input:", err)
    }
    for scanner.Scan() {
        totalIPs++
        line := strings.TrimSpace(scanner.Text())
        
        if strings.Contains(line, ":") {
            proxy := strings.Split(line, ":")
            ip = proxy[0]
            port = proxy[1]
        } else {
            ip = line
            port = os.Args[5]
        }
        
        if processed[ip+port] {
            continue
        }
        processed[ip+port] = true
        
        limit <- struct{}{}
        currentThreads++
        go func() {
            if checkProxy(ip, port, timeout, file, limit) {
                successCount++
            } else {
                failureCount++
            }
            currentThreads--
        }()
    }
}
