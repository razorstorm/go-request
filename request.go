package request

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	HTTPREQUEST_LOG_LEVEL_ERRORS    = 1
	HTTPREQUEST_LOG_LEVEL_VERBOSE   = 2
	HTTPREQUEST_LOG_LEVEL_DEBUG     = 3
	HTTPREQUEST_LOG_LEVEL_OVER_9000 = 9001
)

//--------------------------------------------------------------------------------
// Exported Util Functions
//--------------------------------------------------------------------------------

func CombinePathComponents(components ...string) string {
	slash := "/"
	fullPath := ""
	for index, component := range components {
		working_component := component
		if strings.HasPrefix(working_component, slash) {
			working_component = strings.TrimPrefix(working_component, slash)
		}

		if strings.HasSuffix(working_component, slash) {
			working_component = strings.TrimSuffix(working_component, slash)
		}

		if index != len(components)-1 {
			fullPath = fullPath + working_component + slash
		} else {
			fullPath = fullPath + working_component
		}
	}
	return fullPath
}

//--------------------------------------------------------------------------------
// HttpRequest
//--------------------------------------------------------------------------------

type HttpRequest struct {
	Scheme            string
	Host              string
	Path              string
	QueryString       url.Values
	Header            http.Header
	PostData          url.Values
	BasicAuthUsername string
	BasicAuthPassword string
	Verb              string
	ContentType       string
	Timeout           time.Duration
	TLSCertPath       string
	TLSKeyPath        string
	Body              string

	Logger   *log.Logger
	LogLevel int
}

func NewRequest() *HttpRequest {
	hr := HttpRequest{}
	hr.Scheme = "http"
	hr.Verb = "GET"
	return &hr
}

func (hr *HttpRequest) WithLogging() *HttpRequest {
	hr.LogLevel = HTTPREQUEST_LOG_LEVEL_ERRORS
	hr.Logger = log.New(os.Stdout, "", 0)
	return hr
}

func (hr *HttpRequest) WithLogLevel(logLevel int) *HttpRequest {
	hr.LogLevel = logLevel
	return hr
}

func (hr *HttpRequest) WithLogger(logLevel int, logger *log.Logger) *HttpRequest {
	hr.LogLevel = logLevel
	hr.Logger = logger
	return hr
}

func (hr *HttpRequest) fatalf(logLevel int, format string, args ...interface{}) {
	if hr.Logger != nil && logLevel <= hr.LogLevel {
		prefix := getLoggingPrefix(logLevel)
		hr.Logger.Fatalf(prefix+format, args...)
	}
}

func (hr *HttpRequest) fatalln(logLevel int, args ...interface{}) {
	if hr.Logger != nil && logLevel <= hr.LogLevel {
		prefix := getLoggingPrefix(logLevel)
		message := fmt.Sprint(args...)
		full_message := fmt.Sprintf("%s%s", prefix, message)
		hr.Logger.Fatalln(full_message)
	}
}

func (hr *HttpRequest) logf(logLevel int, format string, args ...interface{}) {
	if hr.Logger != nil && logLevel <= hr.LogLevel {
		prefix := getLoggingPrefix(logLevel)
		hr.Logger.Printf(prefix+format, args...)
	}
}

func (hr *HttpRequest) logln(logLevel int, args ...interface{}) {
	if hr.Logger != nil && logLevel <= hr.LogLevel {
		prefix := getLoggingPrefix(logLevel)
		message := fmt.Sprint(args...)
		full_message := fmt.Sprintf("%s%s", prefix, message)
		hr.Logger.Println(full_message)
	}
}

func (hr *HttpRequest) WithContentType(content_type string) *HttpRequest {
	hr.ContentType = content_type
	return hr
}

func (hr *HttpRequest) WithScheme(scheme string) *HttpRequest {
	hr.Scheme = scheme
	return hr
}

func (hr *HttpRequest) WithHost(host string) *HttpRequest {
	hr.Host = host
	return hr
}

func (hr *HttpRequest) WithPath(path_pattern string, args ...interface{}) *HttpRequest {
	hr.Path = fmt.Sprintf(path_pattern, args...)
	return hr
}

func (hr *HttpRequest) WithCombinedPath(components ...string) *HttpRequest {
	hr.Path = CombinePathComponents(components...)
	return hr
}

func (hr *HttpRequest) WithUrl(url_string string) *HttpRequest {
	working_url, _ := url.Parse(url_string)
	hr.Scheme = working_url.Scheme
	hr.Host = working_url.Host
	hr.Path = working_url.Path
	params := strings.Split(working_url.RawQuery, "&")
	hr.QueryString = url.Values{}
	var key_value []string
	for _, param := range params {
		if param != "" {
			key_value = strings.Split(param, "=")
			hr.QueryString.Set(key_value[0], key_value[1])
		}
	}
	return hr
}

func (hr *HttpRequest) WithHeader(field string, value string) *HttpRequest {
	if hr.Header == nil {
		hr.Header = http.Header{}
	}
	hr.Header.Set(field, value)
	return hr
}

func (hr *HttpRequest) WithQueryString(field string, value string) *HttpRequest {
	if hr.QueryString == nil {
		hr.QueryString = url.Values{}
	}
	hr.QueryString.Add(field, value)
	return hr
}

func (hr *HttpRequest) WithPostData(field string, value string) *HttpRequest {
	if hr.PostData == nil {
		hr.PostData = url.Values{}
	}
	hr.PostData.Add(field, value)
	return hr
}

func (hr *HttpRequest) WithBasicAuth(username, password string) *HttpRequest {
	hr.BasicAuthUsername = username
	hr.BasicAuthPassword = password
	return hr
}

func (hr *HttpRequest) WithTimeout(timeout time.Duration) *HttpRequest {
	hr.Timeout = timeout
	return hr
}

func (hr *HttpRequest) WithTLSCert(cert_path string) *HttpRequest {
	hr.TLSCertPath = cert_path
	return hr
}

func (hr *HttpRequest) WithTLSKey(key_path string) *HttpRequest {
	hr.TLSKeyPath = key_path
	return hr
}

func (hr *HttpRequest) WithVerb(verb string) *HttpRequest {
	hr.Verb = verb
	return hr
}

func (hr *HttpRequest) AsGet() *HttpRequest {
	hr.Verb = "GET"
	return hr
}
func (hr *HttpRequest) AsPost() *HttpRequest {
	hr.Verb = "POST"
	return hr
}
func (hr *HttpRequest) AsPut() *HttpRequest {
	hr.Verb = "PUT"
	return hr
}
func (hr *HttpRequest) AsPatch() *HttpRequest {
	hr.Verb = "PATCH"
	return hr
}
func (hr *HttpRequest) AsDelete() *HttpRequest {
	hr.Verb = "DELETE"
	return hr
}

func (hr *HttpRequest) WithJsonBody(object interface{}) *HttpRequest {
	return hr.WithBody(object, serializeJson).WithContentType("application/json")
}

func (hr *HttpRequest) WithXmlBody(object interface{}) *HttpRequest {
	return hr.WithBody(object, serializeXml).WithContentType("application/xml")
}

func (hr *HttpRequest) WithBody(object interface{}, serialize func(interface{}) string) *HttpRequest {
	return hr.WithRawBody(serialize(object))
}

func (hr *HttpRequest) WithRawBody(body string) *HttpRequest {
	hr.Body = body
	return hr
}

func (hr *HttpRequest) createUrl() url.URL {
	working_url := url.URL{Scheme: hr.Scheme, Host: hr.Host, Path: hr.Path}
	working_url.RawQuery = hr.QueryString.Encode()
	return working_url
}

func (hr *HttpRequest) createHttpRequest() (*http.Request, error) {
	working_url := hr.createUrl()

	if hr.Body != "" && hr.PostData != nil && len(hr.PostData) > 0 {
		return nil, errors.New("Cant set both a body and have post data!")
	}

	var req *http.Request
	if hr.Body != "" {
		body_req, _ := http.NewRequest(hr.Verb, working_url.String(), bytes.NewBufferString(hr.Body))
		req = body_req
	} else {
		if hr.PostData != nil {
			post_req, post_req_error := http.NewRequest(hr.Verb, working_url.String(), bytes.NewBufferString(hr.PostData.Encode()))
			if post_req_error != nil {
				return nil, post_req_error
			}
			req = post_req
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		} else {
			empty_req, _ := http.NewRequest(hr.Verb, working_url.String(), nil)
			req = empty_req
		}
	}

	if hr.BasicAuthUsername != "" {
		req.SetBasicAuth(hr.BasicAuthUsername, hr.BasicAuthPassword)
	}

	if hr.ContentType != "" {
		req.Header.Set("Content-Type", hr.ContentType)
	}

	for key, values := range hr.Header {
		for _, value := range values {
			req.Header.Set(key, value)
		}
	}

	return req, nil
}

func (hr *HttpRequest) FetchRawResponse() (*http.Response, error) {
	req, req_err := hr.createHttpRequest()
	if req_err != nil {
		return nil, req_err
	}

	var client *http.Client

	var transport *http.Transport
	var transport_error error
	if hr.requiresCustomTransport() {
		transport, transport_error = hr.createHttpTransport()
		if transport_error != nil {
			return nil, transport_error
		}
		client.Transport = transport
	}

	if hr.Timeout != time.Duration(0) {
		client.Timeout = hr.Timeout
	}

	hr.logf(HTTPREQUEST_LOG_LEVEL_VERBOSE, "Service Request %v\n", req.URL)
	return client.Do(req)
}

func (hr *HttpRequest) Execute() error {
	_, err := hr.FetchRawResponse()
	return err
}

func (hr *HttpRequest) FetchString() (string, error) {
	res, err := hr.FetchRawResponse()
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	bytes, read_err := ioutil.ReadAll(res.Body)
	if read_err != nil {
		return "", read_err
	}

	hr.logf(HTTPREQUEST_LOG_LEVEL_VERBOSE, "Service Response %s", string(bytes))

	return string(bytes), nil
}

func (hr *HttpRequest) FetchJsonToObject(to_object interface{}) error {
	_, err := hr.handleFetch(newJsonHandler(to_object), doNothingWithReader)
	return err
}

func (hr *HttpRequest) FetchJsonToObjectWithError(success_object interface{}, error_object interface{}) (int, error) {
	return hr.handleFetch(newJsonHandler(success_object), newJsonHandler(error_object))
}

func (hr *HttpRequest) FetchJsonError(error_object interface{}) (int, error) {
	return hr.handleFetch(doNothingWithReader, newJsonHandler(error_object))
}

func (hr *HttpRequest) FetchXmlToObject(to_object interface{}) error {
	_, err := hr.handleFetch(newXmlHandler(to_object), doNothingWithReader)
	return err
}

func (hr *HttpRequest) FetchXmlToObjectWithError(success_object interface{}, error_object interface{}) (int, error) {
	return hr.handleFetch(newXmlHandler(success_object), newXmlHandler(error_object))
}

func (hr *HttpRequest) requiresCustomTransport() bool {
	return !isEmpty(hr.TLSCertPath) && !isEmpty(hr.TLSKeyPath)
}

func (hr *HttpRequest) createHttpTransport() (*http.Transport, error) {
	transport := &http.Transport{
		DisableCompression: false,
	}

	if hr.Timeout != time.Duration(0) {
		transport.TLSHandshakeTimeout = hr.Timeout
		transport.ResponseHeaderTimeout = hr.Timeout
		transport.Dial = (&net.Dialer{
			Timeout:   hr.Timeout,
			KeepAlive: 30 * time.Second,
		}).Dial
		transport.DialTLS = (&net.Dialer{
			Timeout:   hr.Timeout,
			KeepAlive: 30 * time.Second,
		}).Dial
	}

	if !isEmpty(hr.TLSCertPath) && !isEmpty(hr.TLSKeyPath) {
		if cert, err := tls.LoadX509KeyPair(hr.TLSCertPath, hr.TLSKeyPath); err != nil {
			return nil, err
		} else {
			tlsConfig := &tls.Config{
				Certificates: []tls.Certificate{cert},
			}
			transport.TLSClientConfig = tlsConfig
		}
	}

	return transport, nil
}

func (hr *HttpRequest) handleFetch(okHandler httpResponseBodyHandler, errorHandler httpResponseBodyHandler) (status int, err error) {
	res, err := hr.FetchRawResponse()
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}
	hr.logf(HTTPREQUEST_LOG_LEVEL_VERBOSE, "Service Response %s", string(body))

	if res.StatusCode == http.StatusOK {
		err = okHandler(body)
	} else {
		err = errorHandler(body)
	}
	return res.StatusCode, err
}

//--------------------------------------------------------------------------------
// Unexported Utility Functions
//--------------------------------------------------------------------------------

type httpResponseBodyHandler func([]byte) error

func newJsonHandler(object interface{}) httpResponseBodyHandler {
	return func(body []byte) error {
		return deserializeJson(object, string(body))
	}
}

func newXmlHandler(object interface{}) httpResponseBodyHandler {
	return func(body []byte) error {
		return deserializeXml(object, string(body))
	}
}

func doNothingWithReader([]byte) error {
	return nil
}

func deserializeJson(object interface{}, body string) error {
	decoder := json.NewDecoder(bytes.NewBufferString(body))
	return decoder.Decode(object)
}

func deserializeJsonFromReader(object interface{}, body io.Reader) error {
	decoder := json.NewDecoder(body)
	return decoder.Decode(object)
}

func deserializePostBody(object interface{}, body io.ReadCloser) error {
	defer body.Close()
	bodyBytes, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}

	return deserializeJson(object, string(bodyBytes))
}

func serializeJson(object interface{}) string {
	b, _ := json.Marshal(object)
	return string(b)
}

func serializeJsonToReader(object interface{}) io.Reader {
	b, _ := json.Marshal(object)
	return bytes.NewBufferString(string(b))
}

func deserializeXml(object interface{}, body string) error {
	return deserializeXmlFromReader(object, bytes.NewBufferString(body))
}

func deserializeXmlFromReader(object interface{}, reader io.Reader) error {
	decoder := xml.NewDecoder(reader)
	return decoder.Decode(object)
}

func serializeXml(object interface{}) string {
	b, _ := xml.Marshal(object)
	return string(b)
}

func serializeXmlToReader(object interface{}) io.Reader {
	b, _ := xml.Marshal(object)
	return bytes.NewBufferString(string(b))
}

func getLoggingPrefix(logLevel int) string {
	return fmt.Sprintf("HttpRequest (%s): ", formatLogLevel(logLevel))
}

func formatLogLevel(logLevel int) string {
	switch logLevel {
	case HTTPREQUEST_LOG_LEVEL_ERRORS:
		return "ERRORS"
	case HTTPREQUEST_LOG_LEVEL_VERBOSE:
		return "VERBOSE"
	case HTTPREQUEST_LOG_LEVEL_DEBUG:
		return "DEBUG"
	default:
		return "UNKNOWN"
	}
}

func isEmpty(str string) bool {
	return len(str) == 0
}
