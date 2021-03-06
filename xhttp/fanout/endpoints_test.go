package fanout

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestEndpointsFunc(t *testing.T) {
	var (
		assert = assert.New(t)

		original      = httptest.NewRequest("GET", "/", nil)
		expectedURLs  = []*url.URL{new(url.URL)}
		expectedError = errors.New("expected")

		ef = EndpointsFunc(func(actual *http.Request) ([]*url.URL, error) {
			assert.True(original == actual)
			return expectedURLs, expectedError
		})
	)

	actualURLs, actualError := ef.NewEndpoints(original)
	assert.Equal(expectedURLs, actualURLs)
	assert.Equal(expectedError, actualError)
}

func testMustNewEndpointsPanics(t *testing.T) {
	var (
		assert    = assert.New(t)
		endpoints = new(mockEndpoints)
	)

	endpoints.On("NewEndpoints", mock.MatchedBy(func(*http.Request) bool { return true })).Return(nil, errors.New("expected")).Once()
	assert.Panics(func() {
		MustNewEndpoints(endpoints, httptest.NewRequest("GET", "/", nil))
	})

	endpoints.AssertExpectations(t)
}

func testMustNewEndpointsSuccess(t *testing.T) {
	var (
		assert       = assert.New(t)
		expectedURLs = []*url.URL{new(url.URL)}
		endpoints    = new(mockEndpoints)
	)

	endpoints.On("NewEndpoints", mock.MatchedBy(func(*http.Request) bool { return true })).Return(expectedURLs, error(nil)).Once()
	assert.NotPanics(func() {
		assert.Equal(expectedURLs, MustNewEndpoints(endpoints, httptest.NewRequest("GET", "/", nil)))
	})

	endpoints.AssertExpectations(t)
}

func TestMustNewEndpoints(t *testing.T) {
	t.Run("Panics", testMustNewEndpointsPanics)
	t.Run("Success", testMustNewEndpointsSuccess)
}

func testNewFixedEndpointsEmpty(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		fe, err = NewFixedEndpoints()
	)

	require.NotNil(fe)
	assert.Empty(fe)
	assert.NoError(err)
}

func testNewFixedEndpointsInvalid(t *testing.T) {
	var (
		assert  = assert.New(t)
		fe, err = NewFixedEndpoints("%%")
	)

	assert.Empty(fe)
	assert.Error(err)
}

func testNewFixedEndpointsValid(t *testing.T, urls []string, originalURL string, expected []string) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		fe, err = NewFixedEndpoints(urls...)
	)

	require.NotNil(fe)
	require.Len(fe, len(urls))
	require.NoError(err)

	actual, err := fe.NewEndpoints(httptest.NewRequest("GET", originalURL, nil))
	require.Equal(len(expected), len(actual))
	require.NoError(err)

	for i := 0; i < len(expected); i++ {
		assert.Equal(expected[i], actual[i].String())
	}
}

func TestNewFixedEndpoints(t *testing.T) {
	t.Run("Empty", testNewFixedEndpointsEmpty)
	t.Run("Invalid", testNewFixedEndpointsInvalid)

	t.Run("Valid", func(t *testing.T) {
		testData := []struct {
			urls        []string
			originalURL string
			expected    []string
		}{
			{
				[]string{"http://localhost:8080"},
				"/api/v2/something?value=1#mark",
				[]string{"http://localhost:8080/api/v2/something?value=1#mark"},
			},
			{
				[]string{"http://host1.someplace.com", "https://host2.someplace.net:1234"},
				"/api/v2/something",
				[]string{"http://host1.someplace.com/api/v2/something", "https://host2.someplace.net:1234/api/v2/something"},
			},
		}

		for _, record := range testData {
			testNewFixedEndpointsValid(t, record.urls, record.originalURL, record.expected)
		}
	})
}

func testMustNewFixedEndpointsPanics(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() {
		MustNewFixedEndpoints("%%")
	})
}

func testMustNewFixedEndpointsSuccess(t *testing.T) {
	assert := assert.New(t)
	assert.NotPanics(func() {
		fe := MustNewFixedEndpoints("http://foobar.com")
		assert.Len(fe, 1)
		assert.Equal("http://foobar.com", fe[0].String())
	})
}

func TestMustNewFixedEndpoints(t *testing.T) {
	t.Run("Panics", testMustNewFixedEndpointsPanics)
	t.Run("Success", testMustNewFixedEndpointsSuccess)
}
