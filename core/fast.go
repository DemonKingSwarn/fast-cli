package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func isTerminal() bool {
	return os.Getenv("TERM") != ""
}

type Options struct {
	JSON   bool
	Simple bool
}

type Result struct {
	DownloadSpeed float64 `json:"downloadSpeed"`
	UploadSpeed   float64 `json:"uploadSpeed"`
	DownloadUnit  string  `json:"downloadUnit"`
	UploadUnit    string  `json:"uploadUnit"`
	Downloaded    int64   `json:"downloaded"`
	Uploaded      int64   `json:"uploaded"`
	Latency       int     `json:"latency"`
	BufferBloat   int     `json:"bufferBloat"`
	UserLocation  string  `json:"userLocation"`
	UserIP        string  `json:"userIp"`
	ServerURL     string  `json:"serverUrl"`
}

func (r Result) String(verbose bool) string {
	output := fmt.Sprintf("download: %.1f Mbps", r.DownloadSpeed)
	if r.UploadSpeed > 0 {
		output += fmt.Sprintf("\nupload: %.1f Mbps", r.UploadSpeed)
	}
	if verbose {
		output += fmt.Sprintf("\n\nLatency: %d ms (unloaded) / %d ms (loaded)", r.Latency, r.BufferBloat)
	}
	return output
}

func (r Result) JSON() string {
	data, _ := json.MarshalIndent(r, "", "  ")
	return string(data)
}

var (
	spinnerRunning bool
	spinnerMutex   sync.Mutex
	spinnerDone    chan struct{}
)

func startSpinner() {
	if !isTerminal() {
		fmt.Fprint(os.Stderr, ".")
		return
	}

	spinnerMutex.Lock()
	spinnerRunning = true
	spinnerDone = make(chan struct{})
	spinnerMutex.Unlock()

	go func() {
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			select {
			case <-spinnerDone:
				return
			default:
				fmt.Fprintf(os.Stderr, "\033[G%s", frames[i%len(frames)])
				time.Sleep(80 * time.Millisecond)
				i++
			}
		}
	}()
}

func stopSpinner() {
	spinnerMutex.Lock()
	if spinnerRunning {
		spinnerRunning = false
		close(spinnerDone)
	}
	spinnerMutex.Unlock()

	if !isTerminal() {
		return
	}
	time.Sleep(100 * time.Millisecond)
	fmt.Fprintf(os.Stderr, "\r          \r")
}

func RunTest(opts Options) (Result, error) {
	token, err := getFastToken()
	if err != nil {
		return Result{}, fmt.Errorf("failed to get token: %w", err)
	}

	urls, err := getDownloadURLs(token)
	if err != nil {
		return Result{}, fmt.Errorf("failed to get download URLs: %w", err)
	}

	if len(urls) == 0 {
		return Result{}, fmt.Errorf("no download URLs returned")
	}

	result := Result{
		DownloadUnit: "Mbps",
		UploadUnit:   "Mbps",
		ServerURL:    urls[0],
	}

	latency, err := measureLatency(urls[0])
	if err == nil {
		result.Latency = latency
		result.BufferBloat = latency
	}

	if !opts.Simple {
		startSpinner()
	}
	downloadSpeed, downloaded, err := measureDownload(urls[0])
	stopSpinner()
	if err != nil {
		return Result{}, fmt.Errorf("download test failed: %w", err)
	}

	result.DownloadSpeed = downloadSpeed
	result.Downloaded = downloaded

	if !opts.Simple {
		startSpinner()
	}
	uploadSpeed, uploaded, err := measureUpload(urls[0], token)
	stopSpinner()
	if err != nil {
		return Result{}, fmt.Errorf("upload test failed: %w", err)
	}
	result.UploadSpeed = uploadSpeed
	result.Uploaded = uploaded

	return result, nil
}

func getFastToken() (string, error) {
	resp, err := http.Get("https://fast.com")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	re := regexp.MustCompile(`app-[^/]+\.js`)
	matches := re.FindString(html)
	if matches == "" {
		return "", fmt.Errorf("could not find app script")
	}

	scriptURL := fmt.Sprintf("https://fast.com/%s", matches)
	resp, err = http.Get(scriptURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	scriptBody, _ := io.ReadAll(resp.Body)
	script := string(scriptBody)

	re = regexp.MustCompile(`token:"([a-zA-Z]+)"`)
	tokenMatch := re.FindStringSubmatch(script)
	if len(tokenMatch) < 2 {
		return "", fmt.Errorf("could not find token in script")
	}

	return tokenMatch[1], nil
}

func getDownloadURLs(token string) ([]string, error) {
	url := fmt.Sprintf("https://api.fast.com/netflix/speedtest?https=true&token=%s&urlCount=6", token)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var data []struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	urls := make([]string, len(data))
	for i, d := range data {
		urls[i] = d.URL
	}
	return urls, nil
}

func measureDownload(url string) (float64, int64, error) {
	start := time.Now()

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	var totalBytes int64
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			totalBytes += int64(n)
		}
		if err != nil {
			break
		}
	}

	duration := time.Since(start).Seconds()
	speedMbps := float64(totalBytes*8) / (duration * 1_000_000)

	return speedMbps, totalBytes, nil
}

func measureUpload(serverURL, token string) (float64, int64, error) {
	uploadURL := fmt.Sprintf("%s?token=%s", serverURL, token)

	data := make([]byte, 2*1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	start := time.Now()

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	req, _ := http.NewRequest("POST", uploadURL, bytes.NewReader(data))
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	io.Copy(io.Discard, resp.Body)

	duration := time.Since(start).Seconds()
	uploaded := int64(len(data))
	speedMbps := float64(uploaded*8) / (duration * 1_000_000)

	return speedMbps, uploaded, nil
}

func measureLatency(url string) (int, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	latencyURL := strings.Replace(url, "/speedtest?", "/speedtest/range/0-0?", 1)

	var latencies []int64

	for i := 0; i < 5; i++ {
		start := time.Now()
		req, _ := http.NewRequest("GET", latencyURL, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")

		resp, err := client.Do(req)
		if err != nil {
			return 0, err
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		latency := time.Since(start).Milliseconds() / 2
		latencies = append(latencies, latency)
		time.Sleep(200 * time.Millisecond)
	}

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	return int(latencies[len(latencies)/2]), nil
}

func getContentLength(url string) (int64, error) {
	resp, err := http.Head(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	cl := resp.Header.Get("Content-Length")
	if cl == "" {
		return 0, fmt.Errorf("no content length")
	}

	return strconv.ParseInt(cl, 10, 64)
}
