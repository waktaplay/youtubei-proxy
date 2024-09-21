package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"net/http"
	"net/url"

	"encoding/json"
	"github.com/common-nighthawk/go-figure"
	"golang.org/x/net/proxy"
)

const (
	PORT       = 8123
	CHUNK_SIZE = 10 * 1024 * 1024 // 10MB
)

func copyHeader(headerName string, to http.Header, from http.Header) {
	hdrVal := from.Get(headerName)
	if hdrVal != "" {
		to.Set(headerName, hdrVal)
	}
}

func handler(w http.ResponseWriter, req *http.Request) {
	method := req.Method
	headers := req.Header
	urlStr := req.URL.String()

	// CORS preflight 처리
	if method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", headers.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
		w.WriteHeader(http.StatusOK)
		return
	}

	urlObj, err := url.Parse(urlStr)
	if err != nil {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	if !urlObj.Query().Has("__host") {
		http.Error(w, "Request is formatted incorrectly. Please include __host in the query string.", http.StatusBadRequest)
		return
	}

	urlObj.Scheme = "https"
	urlObj.Host = urlObj.Query().Get("__host")
	query := urlObj.Query()
	query.Del("__host")
	urlObj.RawQuery = query.Encode()

	// 헤더 복사
	requestHeaders := make(http.Header)
	if headersStr := urlObj.Query().Get("__headers"); headersStr != "" {
		json.Unmarshal([]byte(headersStr), &requestHeaders)
	}
	copyHeader("User-Agent", requestHeaders, headers)
	query.Del("__headers")
	urlObj.RawQuery = query.Encode()

	proxyURL := os.Getenv("PROXY")
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	if proxyURL != "" {
		proxyURLParsed, err := url.Parse(proxyURL)
		if err == nil {
			dialer, err := proxy.FromURL(proxyURLParsed, proxy.Direct)
			if err == nil {
				transport := &http.Transport{Dial: dialer.Dial}
				client.Transport = transport
			}
		}
	}

	// 청크 단위로 요청
	chunkStart := 0
	for {
		chunkEnd := chunkStart + CHUNK_SIZE - 1
		if chunkEnd < 0 {
			break
		}

		proxyReq, err := http.NewRequest(method, urlObj.String(), nil)
		if err != nil {
			log.Fatal("Failed to create request", err)
			http.Error(w, "Failed to create request", http.StatusInternalServerError)
			return
		}

		// Range 헤더 설정
		proxyReq.Header = requestHeaders
		proxyReq.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", chunkStart, chunkEnd))

		fetchRes, err := client.Do(proxyReq)
		if err != nil {
			log.Fatal("Failed to fetch response", err)
			http.Error(w, "Failed to fetch response", http.StatusInternalServerError)
			return
		}
		defer fetchRes.Body.Close()

		// 응답 처리
		if fetchRes.StatusCode != http.StatusPartialContent {
			break
		}

		// 응답 스트림을 클라이언트에 전달
		if _, err := io.Copy(w, fetchRes.Body); err != nil {
			log.Printf("Error writing response: %v", err)
			return
		}

		chunkStart += CHUNK_SIZE // 다음 청크로 이동
	}
}

func main() {
	fig := figure.NewFigure("Innertube Proxy", "doom", true)
	fmt.Println(fig.String())
	fmt.Println("----------------------------------------------------------------")

	http.HandleFunc("/", handler)
	log.Printf("[INFO] Server is running on port %d.\n", PORT)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", PORT), nil))
}
