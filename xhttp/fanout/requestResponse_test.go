package fanout

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testOriginalBodyNoBody(t *testing.T, originalBody []byte) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		ctx      = context.WithValue(context.Background(), "foo", "bar")
		original = httptest.NewRequest("GET", "/", nil)
		fanout   = &http.Request{
			Header:        http.Header{"Content-Type": []string{"foo"}},
			ContentLength: 123,
			Body:          ioutil.NopCloser(new(bytes.Reader)),
			GetBody: func() (io.ReadCloser, error) {
				assert.Fail("GetBody should not be called")
				return nil, nil
			},
		}
		rf = OriginalBody(true)
	)

	require.NotNil(rf)

	assert.Equal(ctx, rf(ctx, original, fanout, originalBody))
	assert.Empty(fanout.Header.Get("Content-Type"))
	assert.Zero(fanout.ContentLength)
	assert.Nil(fanout.Body)
	assert.Nil(fanout.GetBody)
}

func testOriginalBodyFollowRedirects(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		originalBody = "here is a lovely HTTP entity"

		ctx      = context.WithValue(context.Background(), "foo", "bar")
		original = httptest.NewRequest("GET", "/", nil)
		fanout   = &http.Request{
			Header:        http.Header{"Content-Type": []string{"foo"}},
			ContentLength: 123,
			Body:          ioutil.NopCloser(new(bytes.Reader)),
			GetBody: func() (io.ReadCloser, error) {
				assert.Fail("GetBody should have been updated")
				return nil, nil
			},
		}
		rf = OriginalBody(true)
	)

	require.NotNil(rf)
	original.Header.Set("Content-Type", "text/plain")

	assert.Equal(ctx, rf(ctx, original, fanout, []byte(originalBody)))
	assert.Equal("text/plain", fanout.Header.Get("Content-Type"))
	assert.Equal(int64(len(originalBody)), fanout.ContentLength)

	require.NotNil(fanout.Body)
	actualBody, err := ioutil.ReadAll(fanout.Body)
	require.NoError(err)
	assert.Equal(originalBody, string(actualBody))

	require.NotNil(fanout.GetBody)
	newBody, err := fanout.GetBody()
	require.NoError(err)
	require.NotNil(newBody)
	actualBody, err = ioutil.ReadAll(newBody)
	require.NoError(err)
	assert.Equal(originalBody, string(actualBody))
}

func testOriginalBodyNoFollowRedirects(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		originalBody = "here is a lovely HTTP entity"

		ctx      = context.WithValue(context.Background(), "foo", "bar")
		original = httptest.NewRequest("GET", "/", nil)
		fanout   = &http.Request{
			Header:        http.Header{"Content-Type": []string{"foo"}},
			ContentLength: 123,
			Body:          ioutil.NopCloser(new(bytes.Reader)),
			GetBody: func() (io.ReadCloser, error) {
				assert.Fail("GetBody should have been updated")
				return nil, nil
			},
		}
		rf = OriginalBody(false)
	)

	require.NotNil(rf)
	original.Header.Set("Content-Type", "text/plain")

	assert.Equal(ctx, rf(ctx, original, fanout, []byte(originalBody)))
	assert.Equal("text/plain", fanout.Header.Get("Content-Type"))
	assert.Equal(int64(len(originalBody)), fanout.ContentLength)

	require.NotNil(fanout.Body)
	actualBody, err := ioutil.ReadAll(fanout.Body)
	require.NoError(err)
	assert.Equal(originalBody, string(actualBody))

	assert.Nil(fanout.GetBody)
}

func TestOriginalBody(t *testing.T) {
	t.Run("NilBody", func(t *testing.T) { testOriginalBodyNoBody(t, nil) })
	t.Run("EmptyBody", func(t *testing.T) { testOriginalBodyNoBody(t, make([]byte, 0)) })
	t.Run("FollowRedirects=true", testOriginalBodyFollowRedirects)
	t.Run("FollowRedirects=false", testOriginalBodyNoFollowRedirects)
}

func testOriginalHeaders(t *testing.T, originalHeader http.Header, headersToCopy []string, expectedFanoutHeader http.Header) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		ctx     = context.WithValue(context.Background(), "foo", "bar")

		original = &http.Request{
			Header: originalHeader,
		}

		fanout = &http.Request{
			Header: make(http.Header),
		}

		rf = OriginalHeaders(headersToCopy...)
	)

	require.NotNil(rf)
	assert.Equal(ctx, rf(ctx, original, fanout, nil))
	assert.Equal(expectedFanoutHeader, fanout.Header)
}

func TestOriginalHeaders(t *testing.T) {
	testData := []struct {
		originalHeader       http.Header
		headersToCopy        []string
		expectedFanoutHeader http.Header
	}{
		{
			http.Header{"X-Test-1": []string{"foo"}, "X-Test-2": []string{"foo", "bar"}, "X-Test-3": []string{}},
			nil,
			http.Header{},
		},
		{
			http.Header{"X-Test-1": []string{"foo"}, "X-Test-2": []string{"foo", "bar"}, "X-Test-3": []string{}},
			[]string{"X-Does-Not-Exist"},
			http.Header{},
		},
		{
			http.Header{"X-Test-1": []string{"foo"}, "X-Test-2": []string{"foo", "bar"}, "X-Test-3": []string{}},
			[]string{"X-Does-Not-Exist", "X-Test-1"},
			http.Header{"X-Test-1": []string{"foo"}},
		},
		{
			http.Header{"X-Test-1": []string{"foo"}, "X-Test-2": []string{"foo", "bar"}, "X-Test-3": []string{}},
			[]string{"X-Does-Not-Exist", "x-test-1"},
			http.Header{"X-Test-1": []string{"foo"}},
		},
		{
			http.Header{"X-Test-1": []string{"foo"}, "X-Test-2": []string{"foo", "bar"}, "X-Test-3": []string{}},
			[]string{"X-Test-1"},
			http.Header{"X-Test-1": []string{"foo"}},
		},
		{
			http.Header{"X-Test-1": []string{"foo"}, "X-Test-2": []string{"foo", "bar"}, "X-Test-3": []string{}},
			[]string{"X-Test-3", "X-Test-1"},
			http.Header{"X-Test-1": []string{"foo"}},
		},
		{
			http.Header{"X-Test-1": []string{"foo"}, "X-Test-2": []string{"foo", "bar"}, "X-Test-3": []string{}},
			[]string{"x-TeST-3", "X-tESt-1"},
			http.Header{"X-Test-1": []string{"foo"}},
		},
		{
			http.Header{"X-Test-1": []string{"foo"}, "X-Test-2": []string{"foo", "bar"}, "X-Test-3": []string{}},
			[]string{"X-Test-3", "X-Test-1", "X-Test-2"},
			http.Header{"X-Test-1": []string{"foo"}, "X-Test-2": []string{"foo", "bar"}},
		},
		{
			http.Header{"X-Test-1": []string{"foo"}, "X-Test-2": []string{"foo", "bar"}, "X-Test-3": []string{}},
			[]string{"X-TEST-3", "x-TEsT-1", "x-TesT-2"},
			http.Header{"X-Test-1": []string{"foo"}, "X-Test-2": []string{"foo", "bar"}},
		},
	}

	for i, record := range testData {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Logf("%#v", record)
			testOriginalHeaders(t, record.originalHeader, record.headersToCopy, record.expectedFanoutHeader)
		})
	}
}

func testFanoutHeaders(t *testing.T, fanoutResponse *http.Response, headersToCopy []string, expectedResponseHeader http.Header) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		ctx     = context.WithValue(context.Background(), "foo", "bar")

		response = httptest.NewRecorder()
		rf       = FanoutHeaders(headersToCopy...)
	)

	require.NotNil(rf)
	assert.Equal(ctx, rf(ctx, response, Result{Response: fanoutResponse}))
	assert.Equal(expectedResponseHeader, response.Header())
}

func TestFanoutHeaders(t *testing.T) {
	testData := []struct {
		fanoutResponse         *http.Response
		headersToCopy          []string
		expectedResponseHeader http.Header
	}{
		{
			nil,
			nil,
			http.Header{},
		},
		{
			&http.Response{},
			nil,
			http.Header{},
		},
		{
			&http.Response{Header: http.Header{"X-Test-1": []string{"foo"}}},
			nil,
			http.Header{},
		},
		{
			&http.Response{Header: http.Header{"X-Test-1": []string{"foo"}, "X-Test-2": []string{"foo", "bar"}, "X-Test-3": []string{}}},
			[]string{"X-Does-Not-Exist"},
			http.Header{},
		},
		{
			&http.Response{Header: http.Header{"X-Test-1": []string{"foo"}, "X-Test-2": []string{"foo", "bar"}, "X-Test-3": []string{}}},
			[]string{"X-Does-Not-Exist", "X-Test-1"},
			http.Header{"X-Test-1": []string{"foo"}},
		},
		{
			&http.Response{Header: http.Header{"X-Test-1": []string{"foo"}, "X-Test-2": []string{"foo", "bar"}, "X-Test-3": []string{}}},
			[]string{"X-Does-Not-Exist", "x-TeSt-1"},
			http.Header{"X-Test-1": []string{"foo"}},
		},
		{
			&http.Response{Header: http.Header{"X-Test-1": []string{"foo"}, "X-Test-2": []string{"foo", "bar"}, "X-Test-3": []string{}}},
			[]string{"X-Test-3", "X-Test-1"},
			http.Header{"X-Test-1": []string{"foo"}},
		},
		{
			&http.Response{Header: http.Header{"X-Test-1": []string{"foo"}, "X-Test-2": []string{"foo", "bar"}, "X-Test-3": []string{}}},
			[]string{"x-TeST-3", "X-tESt-1"},
			http.Header{"X-Test-1": []string{"foo"}},
		},
		{
			&http.Response{Header: http.Header{"X-Test-1": []string{"foo"}, "X-Test-2": []string{"foo", "bar"}, "X-Test-3": []string{}}},
			[]string{"X-Test-3", "X-Test-1", "X-Test-2"},
			http.Header{"X-Test-1": []string{"foo"}, "X-Test-2": []string{"foo", "bar"}},
		},
		{
			&http.Response{Header: http.Header{"X-Test-1": []string{"foo"}, "X-Test-2": []string{"foo", "bar"}, "X-Test-3": []string{}}},
			[]string{"X-TEST-3", "x-TEsT-1", "x-TesT-2"},
			http.Header{"X-Test-1": []string{"foo"}, "X-Test-2": []string{"foo", "bar"}},
		},
	}

	for i, record := range testData {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Logf("%#v", record)
			testFanoutHeaders(t, record.fanoutResponse, record.headersToCopy, record.expectedResponseHeader)
		})
	}
}
