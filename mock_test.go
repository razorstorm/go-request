package request

import (
	"encoding/xml"
	"io"
	"testing"

	assert "github.com/blendlabs/go-assert"
)

type mockObject struct {
	XMLName      xml.Name `json:"-" xml:"Borrower"`
	ID           int      `json:"id" xml:"id,attr"`
	Email        string   `json:"email" xml:"-"`
	DeploymentID int      `json:"deployment_id" xml:"-"`
}

func testServiceRequest(t *testing.T, additionalTests ...func(*Request)) {
	assert := assert.New(t)
	sr := New().
		WithMockedResponse(MockedResponseInjector).
		AsDelete().
		AsPatch().
		AsPut().
		AsPost().
		AsGet().
		WithScheme("http").
		WithHost("localhost:5001").
		WithPath("/api/v1/borrowers/2").
		WithHeader("deployment", "test").
		WithPostData("test", "regressions").
		WithQueryString("foo", "bar").
		WithTimeout(500).
		WithQueryString("moobar", "zoobar").
		WithScheme("http")

	assert.Equal("http", sr.Scheme)
	assert.Equal("localhost:5001", sr.Host)
	assert.Equal("GET", sr.Verb)

	req, err := sr.Request()
	assert.Nil(err)
	assert.NotNil(req)

	for _, additionalTest := range additionalTests {
		additionalTest(sr)
	}
}

func testForID(id int, assert *assert.Assertions) func(sr *Request) {
	return func(sr *Request) {
		res := mockObject{}
		err := sr.JSON(&res)
		assert.Nil(err)
		assert.Equal(id, res.ID)
	}
}

func TestFileServiceRequestScheduler(t *testing.T) {
	assert := assert.New(t)
	defer ClearMockedResponses()
	res := []string{
		"{\"id\" : 0, \"deployment_id\": 2 }",
		"{\"id\" : 1, \"deployment_id\": 2 }",
		"{\"id\" : 2, \"deployment_id\": 2 }",
	}
	i := 0
	MockResponse(
		"GET",
		"http://localhost:5001/api/v1/borrowers/2?foo=bar&moobar=zoobar",
		func(io.ReadCloser) MockedResponse {
			r := res[i]
			i++
			return MockedResponse{
				ResponseBody: []byte(r),
				StatusCode:   200,
			}
		},
	)

	testServiceRequest(
		t,
		testForID(0, assert),
		testForID(1, assert),
		testForID(2, assert),
	)
}
