package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api2/route1/1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("API 2 received call")
		type response struct {
			Id        string
			Data      string
			Number    int
			CreatedAt string
		}

		time.Sleep(29 * time.Millisecond)

		data, err := json.Marshal(response{
			Id:        "123456789",
			Data:      "This is a lot of data!",
			Number:    29,
			CreatedAt: "2024-12-12T12:00",
		})
		if err != nil {
			http.Error(w, "Unable to json marshal", http.StatusInternalServerError)
			return
		}

		w.Header().Add("Content-Type", "application/json")
		w.Write(data)
	})

	err := http.ListenAndServe(":8092", mux)
	if err != nil {
		log.Fatal(err)
	}
}
