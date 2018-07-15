package externalissuesource

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"strings"
)

var retryCallCount = 0

func HandleCyclopsHttpCalls(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/character.php" {
		file, err := os.Open("./testdata/cyclops/detail.html")
		if err != nil {
			panic(err)
		}
		bytes, err := ioutil.ReadAll(file)
		if err != nil {
			panic(err)
		}
		w.Write(bytes)
		w.WriteHeader(http.StatusOK)
	}
	if r.URL.Path == "/issue.php" {
		id := r.URL.Query().Get("ID")
		if id == "bogus" {
			w.WriteHeader(http.StatusNotFound)
		}
		if id == "338389" {
			file, err := os.Open("./testdata/cyclops/issues/338389.html")
			if err != nil {
				panic(err)
			}
			bytes, err := ioutil.ReadAll(file)
			if err != nil {
				panic(err)
			}
			w.Write(bytes)
			w.WriteHeader(http.StatusOK)
		}
		if id == "339874" {
			file, err := os.Open("./testdata/cyclops/issues/339874.html")
			if err != nil {
				panic(err)
			}
			bytes, err := ioutil.ReadAll(file)
			if err != nil {
				panic(err)
			}
			w.Write(bytes)
			w.WriteHeader(http.StatusOK)
		}
		if id == "342821" {
			file, err := os.Open("./testdata/cyclops/issues/342821.html")
			if err != nil {
				panic(err)
			}
			bytes, err := ioutil.ReadAll(file)
			if err != nil {
				panic(err)
			}
			w.Write(bytes)
			w.WriteHeader(http.StatusOK)
		}
		if id == "344932" {
			if retryCallCount == 0 {
				file, err := os.Open("./testdata/cbdb_error.html")
				if err != nil {
					panic(err)
				}
				bytes, err := ioutil.ReadAll(file)
				if err != nil {
					panic(err)
				}
				w.Write(bytes)
				w.WriteHeader(http.StatusOK)
			} else {
				file, err := os.Open("./testdata/cyclops/issues/344932.html")
				if err != nil {
					panic(err)
				}
				bytes, err := ioutil.ReadAll(file)
				if err != nil {
					panic(err)
				}
				w.Write(bytes)
				w.WriteHeader(http.StatusOK)
			}
			retryCallCount += 1
		}
	}
}

func TestCbdbExternalSource_CharacterFails(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()
	url := fmt.Sprintf("%s/character.php?ID=82321", ts.URL)
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Error(err)
	}
	config := &CbdbExternalSourceConfig{}
	externalSource := NewCbdbExternalSource(ts.Client(), &CbdbParser{}, logger, config)
	character, err := externalSource.Character(url)
	assert.Nil(t, character)
	assert.Error(t, err)
}

func TestCbdbExternalSource_CharacterCyclops(t *testing.T) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Error(err)
	}
	ts := httptest.NewServer(http.HandlerFunc(HandleCyclopsHttpCalls))
	defer ts.Close()
	config := &CbdbExternalSourceConfig{}
	parser := NewCbdbParser(ts.URL)
	externalSource := NewCbdbExternalSource(ts.Client(), parser, logger, config)
	character, err := externalSource.Character(fmt.Sprintf("%s/character.php?ID=82321", ts.URL))
	if err != nil {
		t.Error(err)
	}
	assert.Len(t, character.Issues, 4)
	assert.Equal(t, "Cyclops", character.Name)
	assert.Equal(t, "Marvel", character.Publisher)
	for _, issue := range character.Issues {
		assert.NotEmpty(t, issue.Number)
		assert.NotEmpty(t, issue.SeriesId)
		assert.NotEmpty(t, issue.Series)
		assert.NotEmpty(t, issue.Vendor)
		assert.NotEmpty(t, issue.Id)
		assert.True(t, issue.PublicationDate.Year() > 1)
		assert.True(t, issue.OnSaleDate.Year() > 1)
	}
	retryCallCount = 0
}

func TestNewCbdbExternalSource_SearchCyclops(t *testing.T) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Error(err)
	}
	ts := httptest.NewServer(http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		file, err := os.Open("./testdata/cyclops/search.html")
		if err != nil {
			panic(err)
		}
		bytes, err := ioutil.ReadAll(file)
		if err != nil {
			panic(err)
		}
		w.Write(bytes)
		w.WriteHeader(http.StatusOK)
		assert.Equal(t, "/search.php", r.URL.Path)
		assert.Equal(t, "cyclops", r.URL.Query().Get("form_search"))
		assert.Equal(t, "Character", r.URL.Query().Get("form_searchtype"))
		cookie, err := r.Cookie("PHPSESSID")
		if err != nil {
			panic(err)
		}
		assert.Equal(t, "PHPSESSID", cookie.Name)
		assert.NotEmpty(t, cookie.Value)

	}))
	defer ts.Close()
	config := &CbdbExternalSourceConfig{}
	parser := NewCbdbParser(ts.URL)
	externalSource := NewCbdbExternalSource(ts.Client(), parser, logger, config)
	searchResult, err := externalSource.SearchCharacter("cyclops")
	if err != nil {
		t.Error(err)
	}
	assert.Len(t, searchResult.Results, 46)
	for _, result := range searchResult.Results {
		assert.True(t, strings.Contains(result.Name, "Cyclops"))
		assert.NotEmpty(t, result.Url)
	}
}

func TestNewCbdbExternalSource_SearchCyclopsFails(t *testing.T) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Error(err)
	}
	ts := httptest.NewServer(http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		assert.Equal(t, "/search.php", r.URL.Path)
		assert.Equal(t, "cyclops", r.URL.Query().Get("form_search"))
		assert.Equal(t, "Character", r.URL.Query().Get("form_searchtype"))
		cookie, err := r.Cookie("PHPSESSID")
		if err != nil {
			panic(err)
		}
		assert.Equal(t, "PHPSESSID", cookie.Name)
		assert.NotEmpty(t, cookie.Value)

	}))
	defer ts.Close()
	config := &CbdbExternalSourceConfig{}
	parser := NewCbdbParser(ts.URL)
	externalSource := NewCbdbExternalSource(ts.Client(), parser, logger, config)
	_, err = externalSource.SearchCharacter("cyclops")
	assert.Error(t, err)
}