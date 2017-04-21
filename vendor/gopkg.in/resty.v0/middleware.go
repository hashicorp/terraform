// Copyright (c) 2015-2016 Jeevanandam M (jeeva@myjeeva.com), All rights reserved.
// resty source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package resty

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

//
// Request Middleware(s)
//

func parseRequestURL(c *Client, r *Request) error {
	// Parsing request URL
	reqURL, err := url.Parse(r.URL)
	if err != nil {
		return err
	}

	// If Request.Url is relative path then added c.HostUrl into
	// the request URL otherwise Request.Url will be used as-is
	if !reqURL.IsAbs() {
		if !strings.HasPrefix(r.URL, "/") {
			r.URL = "/" + r.URL
		}

		reqURL, err = url.Parse(c.HostURL + r.URL)
		if err != nil {
			return err
		}
	}

	// Adding Query Param
	query := reqURL.Query()
	for k, v := range c.QueryParam {
		for _, iv := range v {
			query.Add(k, iv)
		}
	}

	for k, v := range r.QueryParam {
		// remove query param from client level by key
		// since overrides happens for that key in the request
		query.Del(k)

		for _, iv := range v {
			query.Add(k, iv)
		}
	}

	reqURL.RawQuery = query.Encode()
	r.URL = reqURL.String()

	return nil
}

func parseRequestHeader(c *Client, r *Request) error {
	hdr := http.Header{}
	for k := range c.Header {
		hdr.Set(k, c.Header.Get(k))
	}
	for k := range r.Header {
		hdr.Set(k, r.Header.Get(k))
	}

	if IsStringEmpty(hdr.Get(hdrUserAgentKey)) {
		hdr.Set(hdrUserAgentKey, fmt.Sprintf(hdrUserAgentValue, Version))
	} else {
		hdr.Set("X-"+hdrUserAgentKey, fmt.Sprintf(hdrUserAgentValue, Version))
	}

	if IsStringEmpty(hdr.Get(hdrAcceptKey)) && !IsStringEmpty(hdr.Get(hdrContentTypeKey)) {
		hdr.Set(hdrAcceptKey, hdr.Get(hdrContentTypeKey))
	}

	r.Header = hdr

	return nil
}

func parseRequestBody(c *Client, r *Request) (err error) {
	if isPayloadSupported(r.Method) {

		// Handling Multipart
		if r.isMultiPart && !(r.Method == PATCH) {
			if err = handleMultipart(c, r); err != nil {
				return
			}

			goto CL
		}

		// Handling Form Data
		if len(c.FormData) > 0 || len(r.FormData) > 0 {
			handleFormData(c, r)

			goto CL
		}

		// Handling Request body
		if r.Body != nil {
			handleContentType(c, r)

			if err = handleRequestBody(c, r); err != nil {
				return
			}
		}
	} else {
		r.Header.Del(hdrContentTypeKey)
	}

CL:
	// by default resty won't set content length, you can if you want to :)
	if c.setContentLength || r.setContentLength {
		r.Header.Set(hdrContentLengthKey, fmt.Sprintf("%d", r.bodyBuf.Len()))
	}

	return
}

func createHTTPRequest(c *Client, r *Request) (err error) {
	if r.bodyBuf == nil {
		r.RawRequest, err = http.NewRequest(r.Method, r.URL, nil)
	} else {
		r.RawRequest, err = http.NewRequest(r.Method, r.URL, r.bodyBuf)
	}

	if err != nil {
		return
	}

	// Assign close connection option
	r.RawRequest.Close = c.closeConnection

	// Add headers into http request
	r.RawRequest.Header = r.Header

	// Add cookies into http request
	for _, cookie := range c.Cookies {
		r.RawRequest.AddCookie(cookie)
	}

	// it's for non-http scheme option
	if r.RawRequest.URL != nil && r.RawRequest.URL.Scheme == "" {
		r.RawRequest.URL.Scheme = c.scheme
		r.RawRequest.URL.Host = r.URL
	}

	return
}

func addCredentials(c *Client, r *Request) error {
	var isBasicAuth bool
	// Basic Auth
	if r.UserInfo != nil { // takes precedence
		r.RawRequest.SetBasicAuth(r.UserInfo.Username, r.UserInfo.Password)
		isBasicAuth = true
	} else if c.UserInfo != nil {
		r.RawRequest.SetBasicAuth(c.UserInfo.Username, c.UserInfo.Password)
		isBasicAuth = true
	}

	if !c.DisableWarn {
		if isBasicAuth && !strings.HasPrefix(r.URL, "https") {
			c.Log.Println("WARNING - Using Basic Auth in HTTP mode is not secure.")
		}
	}

	// Token Auth
	if !IsStringEmpty(r.Token) { // takes precedence
		r.RawRequest.Header.Set(hdrAuthorizationKey, "Bearer "+r.Token)
	} else if !IsStringEmpty(c.Token) {
		r.RawRequest.Header.Set(hdrAuthorizationKey, "Bearer "+c.Token)
	}

	return nil
}

func requestLogger(c *Client, r *Request) error {
	if c.Debug {
		rr := r.RawRequest
		c.Log.Println()
		c.disableLogPrefix()
		c.Log.Println("---------------------- REQUEST LOG -----------------------")
		c.Log.Printf("%s  %s  %s\n", r.Method, rr.URL.RequestURI(), rr.Proto)
		c.Log.Printf("HOST   : %s", rr.URL.Host)
		c.Log.Println("HEADERS:")
		for h, v := range rr.Header {
			c.Log.Printf("%25s: %v", h, strings.Join(v, ", "))
		}
		c.Log.Printf("BODY   :\n%v", r.fmtBodyString())
		c.Log.Println("----------------------------------------------------------")
		c.enableLogPrefix()
	}

	return nil
}

//
// Response Middleware(s)
//

func responseLogger(c *Client, res *Response) error {
	if c.Debug {
		c.Log.Println()
		c.disableLogPrefix()
		c.Log.Println("---------------------- RESPONSE LOG -----------------------")
		c.Log.Printf("STATUS 		: %s", res.Status())
		c.Log.Printf("RECEIVED AT	: %v", res.ReceivedAt())
		c.Log.Printf("RESPONSE TIME	: %v", res.Time())
		c.Log.Println("HEADERS:")
		for h, v := range res.Header() {
			c.Log.Printf("%30s: %v", h, strings.Join(v, ", "))
		}
		if res.Request.isSaveResponse {
			c.Log.Printf("BODY   :\n***** RESPONSE WRITTEN INTO FILE *****")
		} else {
			c.Log.Printf("BODY   :\n%v", res.fmtBodyString())
		}
		c.Log.Println("----------------------------------------------------------")
		c.enableLogPrefix()
	}

	return nil
}

func parseResponseBody(c *Client, res *Response) (err error) {
	// Handles only JSON or XML content type
	ct := res.Header().Get(hdrContentTypeKey)
	if IsJSONType(ct) || IsXMLType(ct) {
		// Considered as Result
		if res.StatusCode() > 199 && res.StatusCode() < 300 {
			if res.Request.Result != nil {
				err = Unmarshal(ct, res.body, res.Request.Result)
			}
		}

		// Considered as Error
		if res.StatusCode() > 399 {
			// global error interface
			if res.Request.Error == nil && c.Error != nil {
				res.Request.Error = reflect.New(c.Error).Interface()
			}

			if res.Request.Error != nil {
				err = Unmarshal(ct, res.body, res.Request.Error)
			}
		}
	}

	return
}

func handleMultipart(c *Client, r *Request) (err error) {
	r.bodyBuf = &bytes.Buffer{}
	w := multipart.NewWriter(r.bodyBuf)

	for k, v := range c.FormData {
		for _, iv := range v {
			w.WriteField(k, iv)
		}
	}

	for k, v := range r.FormData {
		for _, iv := range v {
			if strings.HasPrefix(k, "@") { // file
				err = addFile(w, k[1:], iv)
				if err != nil {
					return
				}
			} else { // form value
				w.WriteField(k, iv)
			}
		}
	}

	// #21 - adding io.Reader support
	if len(r.multipartFiles) > 0 {
		for _, f := range r.multipartFiles {
			err = addFileReader(w, f)
			if err != nil {
				return
			}
		}
	}

	r.Header.Set(hdrContentTypeKey, w.FormDataContentType())
	err = w.Close()

	return
}

func handleFormData(c *Client, r *Request) {
	formData := url.Values{}

	for k, v := range c.FormData {
		for _, iv := range v {
			formData.Add(k, iv)
		}
	}

	for k, v := range r.FormData {
		// remove form data field from client level by key
		// since overrides happens for that key in the request
		formData.Del(k)

		for _, iv := range v {
			formData.Add(k, iv)
		}
	}

	r.bodyBuf = bytes.NewBuffer([]byte(formData.Encode()))
	r.Header.Set(hdrContentTypeKey, formContentType)
	r.isFormData = true
}

func handleContentType(c *Client, r *Request) {
	contentType := r.Header.Get(hdrContentTypeKey)
	if IsStringEmpty(contentType) {
		contentType = DetectContentType(r.Body)
		r.Header.Set(hdrContentTypeKey, contentType)
	}
}

func handleRequestBody(c *Client, r *Request) (err error) {
	var bodyBytes []byte
	contentType := r.Header.Get(hdrContentTypeKey)
	kind := kindOf(r.Body)

	if reader, ok := r.Body.(io.Reader); ok {
		r.bodyBuf = &bytes.Buffer{}
		r.bodyBuf.ReadFrom(reader)
	} else if b, ok := r.Body.([]byte); ok {
		bodyBytes = b
	} else if s, ok := r.Body.(string); ok {
		bodyBytes = []byte(s)
	} else if IsJSONType(contentType) &&
		(kind == reflect.Struct || kind == reflect.Map || kind == reflect.Slice) {
		bodyBytes, err = json.Marshal(r.Body)
	} else if IsXMLType(contentType) && (kind == reflect.Struct) {
		bodyBytes, err = xml.Marshal(r.Body)
	}

	if bodyBytes == nil && r.bodyBuf == nil {
		err = errors.New("Unsupported 'Body' type/value")
	}

	// if any errors during body bytes handling, return it
	if err != nil {
		return
	}

	// []byte into Buffer
	if bodyBytes != nil && r.bodyBuf == nil {
		r.bodyBuf = bytes.NewBuffer(bodyBytes)
	}

	return
}

func saveResponseIntoFile(c *Client, res *Response) error {
	if res.Request.isSaveResponse {
		file := ""

		if len(c.outputDirectory) > 0 && !filepath.IsAbs(res.Request.outputFile) {
			file += c.outputDirectory + string(filepath.Separator)
		}

		file = filepath.Clean(file + res.Request.outputFile)
		err := createDirectory(filepath.Dir(file))
		if err != nil {
			return err
		}

		outFile, err := os.Create(file)
		if err != nil {
			return err
		}
		defer outFile.Close()

		// io.Copy reads maximum 32kb size, it is perfect for large file download too
		defer res.RawResponse.Body.Close()
		written, err := io.Copy(outFile, res.RawResponse.Body)
		if err != nil {
			return err
		}

		res.size = written
	}

	return nil
}
