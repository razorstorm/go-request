package request

import (
	"fmt"
	"math"
	"testing"
	"time"

	"gopkg.in/stretchr/testify.v1/assert"
)

func TestHttpRequestWithUrl(t *testing.T) {
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

func TestTimeout(t *testing.T) {
	assert := assert.New(t)

	timeout := 1000 * time.Millisecond

	begin := time.Now()
	_, err := NewRequest().AsGet().WithUrl("http://localhost:5001/test/timeout").WithTimeout(timeout).FetchRawResponse()
	end := time.Now()

	assert.Nil(err)

	elapsed := end.Sub(begin)
	diff := math.Abs(float64(elapsed) - float64(timeout))

	assert.True(diff < 1000, fmt.Sprintf("Elapsed time was %v\n", time.Duration(int64(diff))))
}
