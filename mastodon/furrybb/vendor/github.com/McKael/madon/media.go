/*
Copyright 2017 Mikael Berthe

Licensed under the MIT license.  Please see the LICENSE file is this directory.
*/

package madon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/sendgrid/rest"
)

// UploadMedia uploads the given file and returns an attachment
func (mc *Client) UploadMedia(filePath string) (*Attachment, error) {
	var b bytes.Buffer

	if filePath == "" {
		return nil, ErrInvalidParameter
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot read file: %s", err.Error())
	}
	defer f.Close()

	w := multipart.NewWriter(&b)
	formWriter, err := w.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("media upload: cannot create form: %s", err.Error())
	}
	if _, err = io.Copy(formWriter, f); err != nil {
		return nil, fmt.Errorf("media upload: cannot create form: %s", err.Error())
	}

	w.Close()

	req, err := mc.prepareRequest("media", rest.Post, nil)
	if err != nil {
		return nil, fmt.Errorf("media prepareRequest failed: %s", err.Error())
	}
	req.Headers["Content-Type"] = w.FormDataContentType()
	req.Body = b.Bytes()

	// Make API call
	r, err := restAPI(req)
	if err != nil {
		return nil, fmt.Errorf("media upload failed: %s", err.Error())
	}

	// Check for error reply
	var errorResult Error
	if err := json.Unmarshal([]byte(r.Body), &errorResult); err == nil {
		// The empty object is not an error
		if errorResult.Text != "" {
			return nil, fmt.Errorf("%s", errorResult.Text)
		}
	}

	// Not an error reply; let's unmarshal the data
	var attachment Attachment
	err = json.Unmarshal([]byte(r.Body), &attachment)
	if err != nil {
		return nil, fmt.Errorf("cannot decode API response (media): %s", err.Error())
	}
	return &attachment, nil
}
