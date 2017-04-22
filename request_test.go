package request

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/blendlabs/go-assert"
	exception "github.com/blendlabs/go-exception"
)

type statusObject struct {
	Status string `json:"status" xml:"status"`
}

func statusOkObject() statusObject {
	return statusObject{"ok!"}
}

type testObject struct {
	ID           int       `json:"id" xml:"id"`
	Name         string    `json:"name" xml:"name"`
	TimestampUtc time.Time `json:"timestamp_utc" xml:"timestamp_utc"`
	Value        float64   `json:"value" xml:"value"`
}

func newTestObject() testObject {
	to := testObject{}
	to.ID = rand.Int()
	to.Name = fmt.Sprintf("Test Object %d", to.ID)
	to.TimestampUtc = time.Now().UTC()
	to.Value = rand.Float64()
	return to
}

func okMeta() *ResponseMeta {
	return &ResponseMeta{StatusCode: http.StatusOK}
}

func errorMeta() *ResponseMeta {
	return &ResponseMeta{StatusCode: http.StatusInternalServerError}
}

func notFoundMeta() *ResponseMeta {
	return &ResponseMeta{StatusCode: http.StatusNotFound}
}

func writeJSON(w http.ResponseWriter, meta *ResponseMeta, response interface{}) error {
	bytes, err := json.Marshal(response)
	if err == nil {
		if !isEmpty(meta.ContentType) {
			w.Header().Set("Content-Type", meta.ContentType)
		} else {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
		}

		for key, value := range meta.Headers {
			w.Header().Set(key, strings.Join(value, ";"))
		}

		w.WriteHeader(meta.StatusCode)
		count, err := w.Write(bytes)
		if count == 0 {
			return exception.New("writeJSON didnt write any bytes")
		}
		if err != nil {
			return err
		}
	} else {
		return err
	}
	return nil
}

func mockEchoEndpoint(meta *ResponseMeta) *httptest.Server {
	return getMockServer(func(w http.ResponseWriter, r *http.Request) {
		if !isEmpty(meta.ContentType) {
			w.Header().Set("Content-Type", meta.ContentType)
		} else {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
		}

		for key, value := range meta.Headers {
			w.Header().Set(key, strings.Join(value, ";"))
		}

		defer r.Body.Close()
		bytes, _ := ioutil.ReadAll(r.Body)
		w.Write(bytes)
	})
}

type validationFunc func(r *http.Request)

func mockEndpoint(meta *ResponseMeta, returnWithObject interface{}, validations validationFunc) *httptest.Server {
	return getMockServer(func(w http.ResponseWriter, r *http.Request) {
		if validations != nil {
			validations(r)
		}

		writeJSON(w, meta, returnWithObject)
	})
}

func mockTLSEndpoint(meta *ResponseMeta, returnWithObject interface{}, validations validationFunc) *httptest.Server {
	return getMockServer(func(w http.ResponseWriter, r *http.Request) {
		if validations != nil {
			validations(r)
		}

		writeJSON(w, meta, returnWithObject)
	})
}

func getMockServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func getTLSMockServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewTLSServer(handler)
}

func TestCreateHttpRequestWithUrl(t *testing.T) {
	assert := assert.New(t)
	sr := New().
		WithURL("http://localhost:5001/api/v1/path/2?env=dev&foo=bar")

	assert.Equal("http", sr.Scheme)
	assert.Equal("localhost:5001", sr.Host)
	assert.Equal("GET", sr.Verb)
	assert.Equal("/api/v1/path/2", sr.Path)
	assert.Equal([]string{"dev"}, sr.QueryString["env"])
	assert.Equal([]string{"bar"}, sr.QueryString["foo"])
	assert.Equal(2, len(sr.QueryString))
}

func TestHttpGet(t *testing.T) {
	assert := assert.New(t)
	returnedObject := newTestObject()
	ts := mockEndpoint(okMeta(), returnedObject, nil)
	testObject := testObject{}
	meta, err := New().AsGet().WithURL(ts.URL).JSONWithMeta(&testObject)
	assert.Nil(err)
	assert.Equal(http.StatusOK, meta.StatusCode)
	assert.Equal(returnedObject, testObject)
}

func TestHttpGetWithExpiringTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("This test involves a 500ms timeout.")
	}

	assert := assert.New(t)
	returnedObject := newTestObject()
	ts := mockEndpoint(okMeta(), returnedObject, func(r *http.Request) {
		time.Sleep(1000 * time.Millisecond)
	})
	testObject := testObject{}

	before := time.Now()
	_, err := New().AsGet().WithTimeout(250 * time.Millisecond).WithURL(ts.URL).JSONWithMeta(&testObject)
	after := time.Now()

	diff := after.Sub(before)
	assert.NotNil(err)
	assert.True(diff < 260*time.Millisecond, "Timeout was ineffective.")
}

func TestHttpGetWithTimeout(t *testing.T) {
	assert := assert.New(t)
	returnedObject := newTestObject()
	ts := mockEndpoint(okMeta(), returnedObject, func(r *http.Request) {
		assert.Equal("GET", r.Method)
	})
	testObject := testObject{}
	meta, err := New().AsGet().WithTimeout(250 * time.Millisecond).WithURL(ts.URL).JSONWithMeta(&testObject)
	assert.Nil(err)
	assert.Equal(http.StatusOK, meta.StatusCode)
	assert.Equal(returnedObject, testObject)
}

func TestTlsHttpGet(t *testing.T) {
	assert := assert.New(t)
	returnedObject := newTestObject()
	ts := mockTLSEndpoint(okMeta(), returnedObject, nil)
	testObject := testObject{}
	meta, err := New().AsGet().WithURL(ts.URL).JSONWithMeta(&testObject)
	assert.Nil(err)
	assert.Equal(http.StatusOK, meta.StatusCode)
	assert.Equal(returnedObject, testObject)
}

func TestHttpPostWithPostData(t *testing.T) {
	assert := assert.New(t)

	returnedObject := newTestObject()
	ts := mockEndpoint(okMeta(), returnedObject, func(r *http.Request) {
		value := r.PostFormValue("foo")
		assert.Equal("bar", value)
	})

	testObject := testObject{}
	meta, err := New().AsPost().WithURL(ts.URL).WithPostData("foo", "bar").JSONWithMeta(&testObject)
	assert.Nil(err)
	assert.Equal(http.StatusOK, meta.StatusCode)
	assert.Equal(returnedObject, testObject)
}

type parameterObject struct {
	ID     string `json:"identifier"`
	Name   string
	Values []int `json:"values"`
}

func TestHttpPostWithPostDataFromObject(t *testing.T) {
	assert := assert.New(t)

	returnedObject := newTestObject()
	ts := mockEndpoint(okMeta(), returnedObject, func(r *http.Request) {
		value := r.PostFormValue("identifier")
		assert.Equal("test", value)
	})

	p := parameterObject{ID: "test", Name: "this is the name", Values: []int{1, 2, 3, 4}}

	testObject := testObject{}
	req := New().AsPost().WithURL(ts.URL).WithPostDataFromObject(p)
	assert.NotEmpty(req.PostData)
	meta, err := req.JSONWithMeta(&testObject)
	assert.Nil(err)
	assert.Equal(http.StatusOK, meta.StatusCode)
	assert.Equal(returnedObject, testObject)
}

func TestHttpPostWithBasicAuth(t *testing.T) {
	assert := assert.New(t)

	ts := mockEndpoint(okMeta(), statusOkObject(), func(r *http.Request) {
		username, password, ok := r.BasicAuth()
		assert.True(ok)
		assert.Equal("test_user", username)
		assert.Equal("test_password", password)
	})

	testObject := statusObject{}
	meta, err := New().AsPost().WithURL(ts.URL).WithBasicAuth("test_user", "test_password").WithPostBody([]byte(`{"status":"ok!"}`)).JSONWithMeta(&testObject)
	assert.Nil(err)
	assert.Equal(http.StatusOK, meta.StatusCode)
	assert.Equal("ok!", testObject.Status)
}

func TestHttpPostWithHeader(t *testing.T) {
	assert := assert.New(t)

	ts := mockEndpoint(okMeta(), statusOkObject(), func(r *http.Request) {
		value := r.Header.Get("test_header")
		assert.Equal(value, "foosballs")
	})

	testObject := statusObject{}
	meta, err := New().AsPost().WithURL(ts.URL).WithHeader("test_header", "foosballs").WithPostBody([]byte(`{"status":"ok!"}`)).JSONWithMeta(&testObject)
	assert.Nil(err)
	assert.Equal(http.StatusOK, meta.StatusCode)
	assert.Equal("ok!", testObject.Status)
}

func TestHttpPostWithCookies(t *testing.T) {
	assert := assert.New(t)

	cookie := &http.Cookie{
		Name:     "test",
		Value:    "foosballs",
		Secure:   true,
		HttpOnly: true,
		Path:     "/test",
		Expires:  time.Now().UTC().AddDate(0, 0, 30),
	}

	ts := mockEndpoint(okMeta(), statusOkObject(), func(r *http.Request) {
		readCookie, readCookieErr := r.Cookie("test")
		assert.Nil(readCookieErr)
		assert.Equal(cookie.Value, readCookie.Value)
	})

	testObject := statusObject{}
	meta, err := New().AsPost().WithURL(ts.URL).WithCookie(cookie).WithPostBody([]byte(`{"status":"ok!"}`)).JSONWithMeta(&testObject)
	assert.Nil(err)
	assert.Equal(http.StatusOK, meta.StatusCode)
	assert.Equal("ok!", testObject.Status)
}

func TestHttpPostWithJSONBody(t *testing.T) {
	assert := assert.New(t)

	returnedObject := newTestObject()
	ts := mockEchoEndpoint(okMeta())

	testObject := testObject{}
	meta, err := New().AsPost().WithURL(ts.URL).WithPostBodyAsJSON(&returnedObject).JSONWithMeta(&testObject)
	assert.Nil(err)
	assert.Equal(http.StatusOK, meta.StatusCode)
	assert.Equal(returnedObject, testObject)
}

func TestHttpPostWithXMLBody(t *testing.T) {
	assert := assert.New(t)

	returnedObject := newTestObject()
	ts := mockEchoEndpoint(okMeta())

	testObject := testObject{}
	meta, err := New().AsPost().WithURL(ts.URL).WithPostBodyAsXML(&returnedObject).XMLWithMeta(&testObject)
	assert.Nil(err)
	assert.Equal(http.StatusOK, meta.StatusCode)
	assert.Equal(returnedObject, testObject)
}

func TestMockedRequests(t *testing.T) {
	assert := assert.New(t)

	ts := mockEndpoint(errorMeta(), nil, func(r *http.Request) {
		assert.True(false, "This shouldnt run in a mocked context.")
	})

	verifyString, meta, err := New().AsPut().WithPostBody([]byte("foobar")).WithURL(ts.URL).WithMockedResponse(func(*http.Request) (bool, *ResponseMeta, []byte, error) {
		return true, okMeta(), []byte("ok!"), nil
	}).StringWithMeta()

	assert.Nil(err)
	assert.Equal(http.StatusOK, meta.StatusCode)
	assert.Equal("ok!", verifyString)
}

func TestOnRequestHook(t *testing.T) {
	assert := assert.New(t)

	ts := mockEchoEndpoint(okMeta())

	called := false
	_, _, err := New().AsPut().WithPostBody([]byte("foobar")).WithURL(ts.URL).OnRequest(func(meta *Meta) {
		called = true
	}).StringWithMeta()
	assert.Nil(err)
	assert.True(called)
}
