package toolkit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

func TestTools_RandomString(t *testing.T) {
	var testTools Tools

	s := testTools.RandomString(10)

	if len(s) != 10 {
		t.Error("didn't return correct length")
	}
}

var uploadTests = []struct {
	name          string
	allowedTypes  []string
	renameFile    bool
	errorExpected bool
}{
	{
		name:          "allowed no rename",
		allowedTypes:  []string{"image/jpeg", "image/png"},
		renameFile:    false,
		errorExpected: false,
	},
	{
		name:          "allowed to rename",
		allowedTypes:  []string{"image/jpeg", "image/png"},
		renameFile:    true,
		errorExpected: false,
	},
	{
		name:          "not allowed",
		allowedTypes:  []string{"image/jpeg"},
		renameFile:    false,
		errorExpected: true,
	},
}

func createImage(writer *multipart.Writer, t *testing.T) {
	t.Helper()
	part, err := writer.CreateFormFile("file", "./testadata/img.png")
	if err != nil {
		t.Error(err)
	}

	f, err := os.Open("./testdata/img.png")
	if err != nil {
		t.Error(err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		t.Error("error decoding image", err)
	}

	err = png.Encode(part, img)
	if err != nil {
		t.Error(err)
	}
}

func TestTools_Uploadfiles(t *testing.T) {
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	var testTools Tools
	t.Run("Upload multiple files", func(t *testing.T) {
		for _, e := range uploadTests {
			var wg sync.WaitGroup

			wg.Add(1)

			go func() {
				defer writer.Close()
				defer wg.Done()
				createImage(writer, t)
			}()

			request := httptest.NewRequest("POST", "/", pr)
			request.Header.Add("Content-Type", writer.FormDataContentType())

			testTools.AllowedFileTypes = e.allowedTypes

			uploadedFiles, err := testTools.UploadFiles(request, "./testdata/uploads/", e.renameFile)
			if err != nil && !e.errorExpected {
				t.Error(err)
			}

			if !e.errorExpected {
				file := fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName)
				if _, err := os.Stat(file); os.IsNotExist(err) {
					t.Errorf("%s: expected file to exist: %s", e.name, err.Error())
				}
				_ = os.Remove(file)
			}

			if !e.errorExpected && err != nil {
				t.Errorf("%s: error expected but not recieved", e.name)
			}

			wg.Wait()
		}
	})

}

func TestTools_UploadOneFile(t *testing.T) {
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	var testTools Tools
	t.Run("upload a single file", func(t *testing.T) {
		go func() {
			defer writer.Close()
			createImage(writer, t)
		}()
		request := httptest.NewRequest("POST", "/", pr)
		request.Header.Add("Content-Type", writer.FormDataContentType())

		uploadedFiles, err := testTools.UploadOneFile(request, "./testdata/uploads/", true)
		if err != nil {
			t.Error(err)
		}

		file := fmt.Sprintf("./testdata/uploads/%s", uploadedFiles.NewFileName)
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("expected file to exist: %s", err.Error())
		}
		_ = os.Remove(file)
	})
}

func TestTools_CreateDirIfNotExist(t *testing.T) {
	var testTools Tools
	path := "./testdata/myDir"

	err := testTools.CreateDirIfNotExist(path)
	if err != nil {
		t.Error(err)
	}

	err = testTools.CreateDirIfNotExist(path)
	if err != nil {
		t.Error(err)
	}
	_ = os.Remove(path)
}

var jsonTests = []struct {
	name          string
	json          string
	errorExpected bool
	maxSize       uint
	allowUnknown  bool
}{
	{name: "good json", json: `{"foo": "bar"}`, errorExpected: false, maxSize: 1024, allowUnknown: false},
	{name: "badly formatted json", json: `{"foo":}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "incorrect type", json: `{"foo": 1}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "two json files", json: `{"foo": "1"}{"alpha": "beta"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "empty body", json: ``, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "syntax error in json", json: `{"foo": 1"`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "unknown field in json", json: `{"fooo": "1"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "allow unknown fields in json", json: `{"fooo": "1"}`, errorExpected: false, maxSize: 1024, allowUnknown: true},
	{name: "missing field name", json: `{jack: "1"}`, errorExpected: true, maxSize: 1024, allowUnknown: true},
	{name: "file too large", json: `{"foo": "bar"}`, errorExpected: true, maxSize: 5, allowUnknown: true},
	{name: "not json", json: `Hello, world!`, errorExpected: true, maxSize: 1024, allowUnknown: true},
}

func TestTools_ReadJSON(t *testing.T) {
	var testTool Tools

	for _, e := range jsonTests {
		testTool.MaxJSONSize = e.maxSize
		testTool.AllowUnknownFields = e.allowUnknown

		var decodedJSON struct {
			Foo string `json:"foo"`
		}

		req, err := http.NewRequest("POST", "/", bytes.NewReader([]byte(e.json)))
		if err != nil {
			t.Log("Error:", err)
		}

		rr := httptest.NewRecorder()

		err = testTool.ReadJSON(rr, req, &decodedJSON)

		if e.errorExpected && err == nil {
			t.Errorf("%s: error expected, but none received", e.name)
		}

		if !e.errorExpected && err != nil {
			t.Errorf("%s: error not expected, but one received: %s", e.name, err.Error())
		}

		req.Body.Close()
	}
}

func TestTools_WriteJson(t *testing.T) {
	var testoTools Tools

	rr := httptest.NewRecorder()
	payload := JSONResponse{
		Error:   false,
		Message: "foo",
	}

	headers := make(http.Header)
	headers.Add("FOO", "BAR")

	err := testoTools.WriteJSON(rr, http.StatusOK, payload, headers)
	if err != nil {
		t.Errorf("failed to write JSON: %v", err)
	}
}

func TestTools_ErrorJSON(t *testing.T) {
	var testTools Tools

	rr := httptest.NewRecorder()
	err := testTools.ErrorJSON(rr, errors.New("some error"), http.StatusServiceUnavailable)
	if err != nil {
		t.Error(err)
	}

	var payload JSONResponse
	decoder := json.NewDecoder(rr.Body)
	err = decoder.Decode(&payload)
	if err != nil {
		t.Error("received error when decoding JSON", err)
	}

	if !payload.Error {
		t.Error("error set to false in JSON, and it should be true")
	}

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("wrong status code returned; expected 503, but got %d", rr.Code)
	}
}
