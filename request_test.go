package request

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/blendlabs/go-assert"
)

type statusObject struct {
	Status string `json:"status" xml:"status"`
}

func statusOkObject() statusObject {
	return statusObject{"ok!"}
}

type testObject struct {
	Id           int       `json:"id" xml:"id"`
	Name         string    `json:"name" xml:"name"`
	TimestampUtc time.Time `json:"timestamp_utc" xml:"timestamp_utc"`
	Value        float64   `json:"value" xml:"value"`
}

func newTestObject() testObject {
	to := testObject{}
	to.Id = rand.Int()
	to.Name = fmt.Sprintf("Test Object %d", to.Id)
	to.TimestampUtc = time.Now().UTC()
	to.Value = rand.Float64()
	return to
}

func okMeta() *HttpResponseMeta {
	return &HttpResponseMeta{StatusCode: http.StatusOK}
}

func errorMeta() *HttpResponseMeta {
	return &HttpResponseMeta{StatusCode: http.StatusInternalServerError}
}

func notFoundMeta() *HttpResponseMeta {
	return &HttpResponseMeta{StatusCode: http.StatusNotFound}
}

func writeJson(w http.ResponseWriter, meta *HttpResponseMeta, response interface{}) error {
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
		count, write_error := w.Write(bytes)
		if count == 0 {
			return errors.New("WriteJson : Didnt write any bytes.")
		}
		if write_error != nil {
			return write_error
		}
	} else {
		return err
	}
	return nil
}

func mockEchoEndpoint(meta *HttpResponseMeta) *httptest.Server {
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

func mockEndpoint(meta *HttpResponseMeta, returnWithObject interface{}, validations validationFunc) *httptest.Server {
	return getMockServer(func(w http.ResponseWriter, r *http.Request) {
		if validations != nil {
			validations(r)
		}

		writeJson(w, meta, returnWithObject)
	})
}

func mockTlsEndpoint(meta *HttpResponseMeta, returnWithObject interface{}, validations validationFunc) *httptest.Server {
	return getMockServer(func(w http.ResponseWriter, r *http.Request) {
		if validations != nil {
			validations(r)
		}

		writeJson(w, meta, returnWithObject)
	})
}

func getMockServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func getTlsMockServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewTLSServer(handler)
}

func TestFormatLogLevel(t *testing.T) {
	assert := assert.New(t)

	errors := formatLogLevel(HTTPREQUEST_LOG_LEVEL_ERRORS)
	assert.Equal("ERRORS", errors)

	verbose := formatLogLevel(HTTPREQUEST_LOG_LEVEL_VERBOSE)
	assert.Equal("VERBOSE", verbose)

	debug := formatLogLevel(HTTPREQUEST_LOG_LEVEL_DEBUG)
	assert.Equal("DEBUG", debug)

	unknown := formatLogLevel(-1)
	assert.Equal("UNKNOWN", unknown)
}

func TestCreateHttpRequestWithUrl(t *testing.T) {
	assert := assert.New(t)
	sr := NewRequest().
		WithUrl("http://localhost:5001/api/v1/path/2?env=dev&foo=bar")

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
	returned_object := newTestObject()
	ts := mockEndpoint(okMeta(), returned_object, nil)
	test_object := testObject{}
	meta, err := NewRequest().AsGet().WithUrl(ts.URL).FetchJsonToObjectWithMeta(&test_object)
	assert.Nil(err)
	assert.Equal(http.StatusOK, meta.StatusCode)
	assert.Equal(returned_object, test_object)
}

func TestHttpGetWithExpiringTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("This test involves a 500ms timeout.")
	}

	assert := assert.New(t)
	returned_object := newTestObject()
	ts := mockEndpoint(okMeta(), returned_object, func(r *http.Request) {
		time.Sleep(1000 * time.Millisecond)
	})
	test_object := testObject{}

	before := time.Now()
	_, err := NewRequest().AsGet().WithTimeout(250 * time.Millisecond).WithUrl(ts.URL).FetchJsonToObjectWithMeta(&test_object)
	after := time.Now()

	diff := after.Sub(before)
	assert.NotNil(err)
	assert.True(diff < 260*time.Millisecond, "Timeout was ineffective.")
}

func TestHttpGetWithTimeout(t *testing.T) {
	assert := assert.New(t)
	returned_object := newTestObject()
	ts := mockEndpoint(okMeta(), returned_object, func(r *http.Request) {
		assert.Equal("GET", r.Method)
	})
	test_object := testObject{}
	meta, err := NewRequest().AsGet().WithTimeout(250 * time.Millisecond).WithUrl(ts.URL).FetchJsonToObjectWithMeta(&test_object)
	assert.Nil(err)
	assert.Equal(http.StatusOK, meta.StatusCode)
	assert.Equal(returned_object, test_object)
}

func TestTlsHttpGet(t *testing.T) {
	assert := assert.New(t)
	returned_object := newTestObject()
	ts := mockTlsEndpoint(okMeta(), returned_object, nil)
	test_object := testObject{}
	meta, err := NewRequest().AsGet().WithUrl(ts.URL).FetchJsonToObjectWithMeta(&test_object)
	assert.Nil(err)
	assert.Equal(http.StatusOK, meta.StatusCode)
	assert.Equal(returned_object, test_object)
}

func TestHttpPostWithPostData(t *testing.T) {
	assert := assert.New(t)

	returned_object := newTestObject()
	ts := mockEndpoint(okMeta(), returned_object, func(r *http.Request) {
		value := r.PostFormValue("foo")
		assert.Equal("bar", value)
	})

	test_object := testObject{}
	meta, err := NewRequest().AsPost().WithUrl(ts.URL).WithPostData("foo", "bar").FetchJsonToObjectWithMeta(&test_object)
	assert.Nil(err)
	assert.Equal(http.StatusOK, meta.StatusCode)
	assert.Equal(returned_object, test_object)
}

type parameterObject struct {
	Id     string `json:"identifier"`
	Name   string
	Values []int `json:"values"`
}

func TestHttpPostWithPostDataFromObject(t *testing.T) {
	assert := assert.New(t)

	returned_object := newTestObject()
	ts := mockEndpoint(okMeta(), returned_object, func(r *http.Request) {
		value := r.PostFormValue("identifier")
		assert.Equal("test", value)
	})

	p := parameterObject{Id: "test", Name: "this is the name", Values: []int{1, 2, 3, 4}}

	test_object := testObject{}
	req := NewRequest().AsPost().WithUrl(ts.URL).WithPostDataFromObject(p)
	assert.NotEmpty(req.PostData)
	meta, err := req.FetchJsonToObjectWithMeta(&test_object)
	assert.Nil(err)
	assert.Equal(http.StatusOK, meta.StatusCode)
	assert.Equal(returned_object, test_object)
}

func TestHttpPostWithBasicAuth(t *testing.T) {
	assert := assert.New(t)

	ts := mockEndpoint(okMeta(), statusOkObject(), func(r *http.Request) {
		username, password, ok := r.BasicAuth()
		assert.True(ok)
		assert.Equal("test_user", username)
		assert.Equal("test_password", password)
	})

	test_object := statusObject{}
	meta, err := NewRequest().AsPost().WithUrl(ts.URL).WithBasicAuth("test_user", "test_password").WithRawBody(`{"status":"ok!"}`).FetchJsonToObjectWithMeta(&test_object)
	assert.Nil(err)
	assert.Equal(http.StatusOK, meta.StatusCode)
	assert.Equal("ok!", test_object.Status)
}

func TestHttpPostWithHeader(t *testing.T) {
	assert := assert.New(t)

	ts := mockEndpoint(okMeta(), statusOkObject(), func(r *http.Request) {
		value := r.Header.Get("test_header")
		assert.Equal(value, "foosballs")
	})

	test_object := statusObject{}
	meta, err := NewRequest().AsPost().WithUrl(ts.URL).WithHeader("test_header", "foosballs").WithRawBody(`{"status":"ok!"}`).FetchJsonToObjectWithMeta(&test_object)
	assert.Nil(err)
	assert.Equal(http.StatusOK, meta.StatusCode)
	assert.Equal("ok!", test_object.Status)
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
		read_cookie, read_cookie_err := r.Cookie("test")
		assert.Nil(read_cookie_err)
		assert.Equal(cookie.Value, read_cookie.Value)
	})

	test_object := statusObject{}
	meta, err := NewRequest().AsPost().WithUrl(ts.URL).WithCookie(cookie).WithRawBody(`{"status":"ok!"}`).FetchJsonToObjectWithMeta(&test_object)
	assert.Nil(err)
	assert.Equal(http.StatusOK, meta.StatusCode)
	assert.Equal("ok!", test_object.Status)
}

func TestHttpPostWithJsonBody(t *testing.T) {
	assert := assert.New(t)

	returned_object := newTestObject()
	ts := mockEchoEndpoint(okMeta())

	test_object := testObject{}
	meta, err := NewRequest().AsPost().WithUrl(ts.URL).WithJsonBody(&returned_object).FetchJsonToObjectWithMeta(&test_object)
	assert.Nil(err)
	assert.Equal(http.StatusOK, meta.StatusCode)
	assert.Equal(returned_object, test_object)
}

func TestHttpPostWithXmlBody(t *testing.T) {
	assert := assert.New(t)

	returned_object := newTestObject()
	ts := mockEchoEndpoint(okMeta())

	test_object := testObject{}
	meta, err := NewRequest().AsPost().WithUrl(ts.URL).WithXmlBody(&returned_object).FetchXmlToObjectWithMeta(&test_object)
	assert.Nil(err)
	assert.Equal(http.StatusOK, meta.StatusCode)
	assert.Equal(returned_object, test_object)
}

func TestMockedRequests(t *testing.T) {
	assert := assert.New(t)

	ts := mockEndpoint(errorMeta(), nil, func(r *http.Request) {
		assert.True(false, "This shouldnt run in a mocked context.")
	})

	verify_string, meta, err := NewRequest().AsPut().WithRawBody("foobar").WithUrl(ts.URL).WithMockedResponse(func(verb string, url *url.URL) (bool, *HttpResponseMeta, []byte, error) {
		return true, okMeta(), []byte("ok!"), nil
	}).FetchStringWithMeta()

	assert.Nil(err)
	assert.Equal(http.StatusOK, meta.StatusCode)
	assert.Equal("ok!", verify_string)
}
