package request

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func DownloadFile(filepath string, url string) (err error) {

	// Create the file
	out, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil  {
	  return err
	}
	defer out.Close()
  
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
	  return err
	}
	defer resp.Body.Close()
  
	// Check server response
	if resp.StatusCode != http.StatusOK {
	  return fmt.Errorf("bad status: %s", resp.Status)
	}
  
	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil  {
	  return err
	}
  
	return nil
  }