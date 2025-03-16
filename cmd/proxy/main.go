package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type responseType string

const (
	fromApiBaseUrl string = "http://localhost:8091/api1"
	toApiBaseUrl   string = "http://localhost:8092/api2"

	selfBaseUrl string = "/proxy-api"

	responseTypeFrom responseType = "FROM_RESPONSE"
	responseTypeTo   responseType = "TO_RESPONSE"
)

type response struct {
	rType        responseType
	httpResponse *http.Response
	time         time.Duration
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/proxy-api/", func(w http.ResponseWriter, r *http.Request) {
		// Currently only GET & OPTION result in a compare flow, since other methods have a high possibility to result in write operations.
		if r.Method != http.MethodGet && r.Method != http.MethodOptions {
			proxyFromRequest(nil, w, r, fromApiBaseUrl)
			return
		}

		responseChan := make(chan response, 2)
		go compare(responseChan)

		go doToRequest(responseChan, r, toApiBaseUrl)
		proxyFromRequest(responseChan, w, r, fromApiBaseUrl)
	})

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatal(err)
	}
}

type compareResult struct {
	success              bool
	fromResponseBody     []byte
	toResponseBody       []byte
	fromResponseStatus   int
	toResponseStatus     int
	fromResponseDuration time.Duration
	toResponseDuration   time.Duration
}

func compare(responseChan <-chan response) {
	var toResponse response
	var fromResponse response

	timeOut := time.NewTimer(1 * time.Minute)
	for toResponse == (response{}) || fromResponse == (response{}) {
		select {
		case r := <-responseChan:
			if r.rType == responseTypeTo {
				toResponse = r
			} else {
				fromResponse = r
			}
		case <-timeOut.C:
			fmt.Println("Timeout: not measuring request")
			return
		}
	}

	fromBody, err := io.ReadAll(fromResponse.httpResponse.Body)
	if err != nil {
		return
	}

	toBody, err := io.ReadAll(toResponse.httpResponse.Body)
	if err != nil {
		return
	}

	result := compareResult{
		success:              true,
		fromResponseBody:     fromBody,
		toResponseBody:       toBody,
		fromResponseStatus:   fromResponse.httpResponse.StatusCode,
		toResponseStatus:     toResponse.httpResponse.StatusCode,
		fromResponseDuration: fromResponse.time,
		toResponseDuration:   toResponse.time,
	}

	if string(result.fromResponseBody) != string(result.toResponseBody) {
		result.success = false
	}

	if result.fromResponseStatus != result.toResponseStatus {
		result.success = false
	}

	logResult(result)
}

func proxyFromRequest(responseChan chan<- response, w http.ResponseWriter, r *http.Request, target string) {
	path := strings.TrimPrefix(r.URL.Path, selfBaseUrl)

	req, err := http.NewRequest(r.Method, target+path, r.Body)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		if responseChan != nil {
			responseChan <- response{
				rType:        responseTypeFrom,
				httpResponse: nil,
				time:         0,
			}
		}
	}
	req.Header = r.Header.Clone()

	startTime := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "Failed to reach backend", http.StatusBadGateway)
		if responseChan != nil {
			responseChan <- response{
				rType:        responseTypeFrom,
				httpResponse: nil,
				time:         0,
			}
		}
	}
	endTime := time.Now()

	defer resp.Body.Close()

	for k, v := range resp.Header {
		w.Header()[k] = v
	}

	w.WriteHeader(resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Cannot read body")
		if responseChan != nil {
			responseChan <- response{
				rType:        responseTypeFrom,
				httpResponse: nil,
				time:         0,
			}
		}
	}
	defer resp.Body.Close()

	resp.Body = io.NopCloser(bytes.NewReader(body))

	w.Write(body)

	if responseChan == nil {
		return
	}

	responseChan <- response{
		rType:        responseTypeFrom,
		httpResponse: resp,
		time:         endTime.Sub(startTime),
	}
}

func doToRequest(responseChan chan<- response, r *http.Request, target string) {
	path := strings.TrimPrefix(r.URL.Path, selfBaseUrl)

	req, err := http.NewRequest(r.Method, target+path, r.Body)
	if err != nil {
		fmt.Println("error: " + err.Error())
		responseChan <- response{
			rType:        responseTypeTo,
			httpResponse: nil,
			time:         0,
		}
	}
	req.Header = r.Header.Clone()

	startTime := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("error: " + err.Error())
		responseChan <- response{
			rType:        responseTypeTo,
			httpResponse: nil,
			time:         0,
		}
	}
	endTime := time.Now()

	responseChan <- response{
		rType:        responseTypeTo,
		httpResponse: resp,
		time:         endTime.Sub(startTime),
	}
}

func logResult(result compareResult) {
	if result.success {
		fmt.Printf("SUCCESS: fromResponse took %s, toResponse took %s \n", result.fromResponseDuration, result.toResponseDuration)
		return
	}

	fmt.Printf("ERROR: reponses do not match: fromResponseStatus: %v, toResponseStatus %v, fromResponseBody: %s, toResponseBody: %s \n", result.fromResponseStatus, result.toResponseStatus, result.fromResponseBody, result.toResponseBody)
}
