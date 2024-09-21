package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"net/http"
	"net/url"

	"encoding/json"

	"github.com/common-nighthawk/go-figure"
	"golang.org/x/net/proxy"
)

const PORT = 8123

func copyHeader(headerName string, to http.Header, from http.Header) {
	hdrVal := from.Get(headerName)
	if hdrVal != "" {
		to.Set(headerName, hdrVal)
	}
}

func copyResponseHeaders(from http.Header, to http.Header) {
	copyHeader("Accept-Ranges", to, from)
	copyHeader("Alt-Svc", to, from)
	copyHeader("Cache-Control", to, from)

	copyHeader("Content-Type", to, from)

	copyHeader("Date", to, from)
	copyHeader("Expires", to, from)
	copyHeader("Last-Modified", to, from)

	copyHeader("Server", to, from)
	copyHeader("Vary", to, from)
	copyHeader("X-Content-Type-Options", to, from)
}

func handler(w http.ResponseWriter, req *http.Request) {
	method := req.Method
	headers := req.Header
	urlStr := req.URL.String()

	// CORS preflight 처리
	if method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", headers.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization, x-goog-visitor-id, x-goog-api-key, x-origin, x-youtube-client-version, x-youtube-client-name, x-goog-api-format-version, x-user-agent, Accept-Language, Range, Referer")
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		w.WriteHeader(http.StatusNoContent)
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

	// Set the URL host to the __host parameter
	urlObj.Scheme = "https"
	urlObj.Host = urlObj.Query().Get("__host")
	query := urlObj.Query()
	query.Del("__host")
	urlObj.RawQuery = query.Encode()

	// Copy headers from the request to the new request
	requestHeaders := make(http.Header)
	if headersStr := urlObj.Query().Get("__headers"); headersStr != "" {
		json.Unmarshal([]byte(headersStr), &requestHeaders)
	}

	copyHeader("Range", requestHeaders, headers)
	if requestHeaders.Get("User-Agent") == "" {
		copyHeader("User-Agent", requestHeaders, headers)
	}

	query.Del("__headers")
	urlObj.RawQuery = query.Encode()

	// Make the request to the target server
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

	reqBody, _ := io.ReadAll(req.Body)

	// Construct the return headers
	responseHeaders := w.Header()

	// Add CORS headers
	responseHeaders.Set("Access-Control-Allow-Origin", headers.Get("Origin"))
	responseHeaders.Set("Access-Control-Allow-Headers", "*")
	responseHeaders.Set("Access-Control-Allow-Methods", "*")
	responseHeaders.Set("Access-Control-Allow-Credentials", "true")
	responseHeaders.Set("Cross-Origin-Resource-Policy", "cross-origin")

	// Check the contentLength is smaller than 10MB
	// If so, set Range header to fetch the whole content
	if strings.HasSuffix(urlObj.Host, "googlevideo.com") && urlObj.Path == "/videoplayback" {
		contentLength, _ := strconv.Atoi(urlObj.Query().Get("clen"))

		if contentLength < 10*1024*1024 {
			requestHeaders.Set("Range", "bytes=0-"+strconv.Itoa(contentLength))
		}
	}

	// Fetch the response and send it to the client as usual
	proxyReq, err := http.NewRequest(method, urlObj.String(), strings.NewReader(string(reqBody)))
	if err != nil {
		log.Fatal("Failed to create request", err)
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}
	proxyReq.Header = requestHeaders

	fetchRes, err := client.Do(proxyReq)
	if err != nil {
		log.Fatal("Failed to fetch response", err)
		http.Error(w, "Failed to fetch response", http.StatusInternalServerError)
		return
	}

	defer fetchRes.Body.Close()

	// Send the response headers to the client
	copyResponseHeaders(fetchRes.Header, responseHeaders)
	copyHeader("Content-Length", responseHeaders, fetchRes.Header)
	w.WriteHeader(fetchRes.StatusCode)

	// Send the response body
	if _, err := io.Copy(w, fetchRes.Body); err != nil {
		log.Printf("Error writing response: %v", err)
		return
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
