package request

import (
	"testing"

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
