package toolkit

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
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
