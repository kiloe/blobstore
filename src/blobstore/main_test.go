package main

import (
	"blob"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"sync"
	"testing"
)

const endpoint = "http://localhost:7000/"

func init() {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		wg.Done()
		ListenAndServe(":7000")
	}()
	wg.Wait()
}

func multipartRequest(path string) (*http.Request, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}
	file.Close()

	body := new(bytes.Buffer)
	w := multipart.NewWriter(body)
	part, err := w.CreateFormFile("file", fi.Name())
	if err != nil {
		return nil, err
	}
	part.Write(fileContents)

	err = w.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req, nil
}

func TestUploadAndDownload(t *testing.T) {
	// Upload...
	req, err := multipartRequest("test.jpg")
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("200 expected got: %d", res.StatusCode)
	}
	// Response...
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	var blobs []*blob.Blob
	err = json.Unmarshal(b, &blobs)
	if err != nil {
		t.Fatal(err)
	}
	if len(blobs) != 1 {
		t.Fatal("expected one blob info response json")
	}
	blob := blobs[0]
	if !blob.Valid() {
		t.Fatal("expected enough info returned to build a valid blob reference")
	}
	if blob.ContentType != "image/jpeg" {
		t.Fatal("expected image/jpeg content type\ngot: " + blob.ContentType)
	}
	// Download data...
	res, err = http.Get(endpoint + "/" + blob.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	original, err := ioutil.ReadFile("test.jpg")
	if err != nil {
		t.Fatal(err)
	}
	downloaded, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, downloaded) {
		fmt.Println("downloaded:", string(downloaded))
		t.Fatal("expected downloaded bytes to equal original bytes")
	}

}
