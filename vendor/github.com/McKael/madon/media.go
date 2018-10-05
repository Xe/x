/*
Copyright 2017-2018 Mikael Berthe

Licensed under the MIT license.  Please see the LICENSE file is this directory.
*/

package madon

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
	"github.com/sendgrid/rest"
)

// UploadMedia uploads the given file and returns an attachment
// The description and focus arguments can be empty strings.
// 'focus' is the "focal point", written as two comma-delimited floating points.
func (mc *Client) UploadMedia(filePath, description, focus string) (*Attachment, error) {
	var b bytes.Buffer

	if filePath == "" {
		return nil, ErrInvalidParameter
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, errors.Wrap(err, "cannot read file")
	}
	defer f.Close()

	w := multipart.NewWriter(&b)
	formWriter, err := w.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, errors.Wrap(err, "media upload")
	}
	if _, err = io.Copy(formWriter, f); err != nil {
		return nil, errors.Wrap(err, "media upload")
	}

	w.Close()

	var params apiCallParams
	if description != "" || focus != "" {
		params = make(apiCallParams)
		if description != "" {
			params["description"] = description
		}
		if focus != "" {
			params["focus"] = focus
		}
	}

	req, err := mc.prepareRequest("media", rest.Post, params)
	if err != nil {
		return nil, errors.Wrap(err, "media prepareRequest failed")
	}
	req.Headers["Content-Type"] = w.FormDataContentType()
	req.Body = b.Bytes()

	// Make API call
	r, err := restAPI(req)
	if err != nil {
		return nil, errors.Wrap(err, "media upload failed")
	}

	// Check for error reply
	var errorResult Error
	if err := json.Unmarshal([]byte(r.Body), &errorResult); err == nil {
		// The empty object is not an error
		if errorResult.Text != "" {
			return nil, errors.New(errorResult.Text)
		}
	}

	// Not an error reply; let's unmarshal the data
	var attachment Attachment
	err = json.Unmarshal([]byte(r.Body), &attachment)
	if err != nil {
		return nil, errors.Wrap(err, "cannot decode API response (media)")
	}
	return &attachment, nil
}

// UpdateMedia updates the description and focal point of a media
// One of the description and focus arguments can be nil to not be updated.
func (mc *Client) UpdateMedia(mediaID int64, description, focus *string) (*Attachment, error) {
	params := make(apiCallParams)
	if description != nil {
		params["description"] = *description
	}
	if focus != nil {
		params["focus"] = *focus
	}

	endPoint := "media/" + strconv.FormatInt(mediaID, 10)
	var attachment Attachment
	if err := mc.apiCall(endPoint, rest.Put, params, nil, nil, &attachment); err != nil {
		return nil, err
	}
	return &attachment, nil
}
