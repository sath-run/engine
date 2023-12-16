package request

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"syscall"
)

var Origin string = "http://unix"

func SendRequestToEngine(method string, path string, data map[string]interface{}) (map[string]interface{}, int) {
	url := Origin + path
	buffer := new(bytes.Buffer)
	json.NewEncoder(buffer).Encode(data)
	req, err := http.NewRequest(method, url, buffer)
	if err != nil {
		log.Fatal(err)
	}

	client := http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", "/var/run/sathcli.sock")
			},
		},
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if errors.Is(err, syscall.ECONNREFUSED) {
		fmt.Println("SATH engine is not running.")
		fmt.Println("If sath-engine is installed, run the following command to start:")
		fmt.Println("  sudo systemctl start sath")
		os.Exit(1)
	} else if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	//Create a variable of the same type as our model
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	var result map[string]interface{}

	if len(body) == 0 {
	} else if err := json.Unmarshal(body, &result); err != nil {
		result = map[string]interface{}{
			"message": string(body),
		}
	}

	return result, resp.StatusCode
}

func EngineGet(path string) map[string]interface{} {
	res, code := SendRequestToEngine(http.MethodGet, path, nil)
	if code < 200 || code >= 400 {
		log.Fatal(res, code)
	}
	return res
}

func EnginePost(path string, data map[string]interface{}) map[string]interface{} {
	res, code := SendRequestToEngine(http.MethodPost, path, data)
	if code < 200 || code >= 400 {
		log.Fatal(res, code)
	}
	return res
}

func EnginePut(path string, data map[string]interface{}) map[string]interface{} {
	res, code := SendRequestToEngine(http.MethodPut, path, data)
	if code < 200 || code >= 400 {
		log.Fatal(res, code)
	}
	return res
}

func EnginePatch(path string, data map[string]interface{}) map[string]interface{} {
	res, code := SendRequestToEngine(http.MethodPatch, path, data)
	if code < 200 || code >= 400 {
		log.Fatal(res, code)
	}
	return res
}

func EngineDelete(path string, data map[string]interface{}) map[string]interface{} {
	res, code := SendRequestToEngine(http.MethodDelete, path, data)
	if code < 200 || code >= 400 {
		log.Fatal(res, code)
	}
	return res
}
