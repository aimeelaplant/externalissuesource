package externalissuesource

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
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
				file, err := os.Open("./testdata/cb_error.html")
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

func HandleMadDogHttpCalls(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/character.php" {
		file, err := os.Open("./testdata/maddog/detail.html")
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
		if id == "369851" {
			file, err := os.Open("./testdata/maddog/369851.html")
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
	}
}


func TestCbExternalSource_CharacterFails(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()
	url := fmt.Sprintf("%s/character.php?ID=82321", ts.URL)
	config := &CbExternalSourceConfig{}
	externalSource := NewCbExternalSource(ts.Client(), config)
	character, err := externalSource.Character(url, func(issueId string) bool {
		return true
	})
	assert.Nil(t, character)
	assert.Error(t, err)
}

func TestCbExternalSource_CharacterCyclops(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(HandleCyclopsHttpCalls))
	defer ts.Close()
	config := &CbExternalSourceConfig{}
	parser := NewCbParser(ts.URL)
	externalSource := CbExternalSource{
		httpClient: ts.Client(),
		parser:     parser,
		config:     config,
	}
	character, err := externalSource.Character(fmt.Sprintf("%s/character.php?ID=82321", ts.URL), func(issueId string) bool {
		return true
	})
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

func TestNewCbExternalSource_SearchCyclops(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	config := &CbExternalSourceConfig{}
	parser := NewCbParser(ts.URL)
	externalSource := CbExternalSource{
		httpClient: ts.Client(),
		parser:     parser,
		config:     config,
	}
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

func TestNewCbExternalSource_SearchCyclopsFails(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	config := &CbExternalSourceConfig{}
	parser := NewCbParser(ts.URL)
	externalSource := CbExternalSource{
		httpClient: ts.Client(),
		parser:     parser,
		config:     config,
	}
	_, err := externalSource.SearchCharacter("cyclops")
	assert.Error(t, err)
}

func TestCbExternalSource_CharacterPage_Cyclops(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))
	config := &CbExternalSourceConfig{}
	parser := NewCbParser(ts.URL)
	externalSource := CbExternalSource{
		httpClient: ts.Client(),
		parser:     parser,
		config:     config,
	}
	character, err := externalSource.CharacterPage(fmt.Sprintf("%s/character.php?ID=82321", ts.URL))
	if err != nil {
		t.Error(err)
	}
	assert.Len(t, character.OtherIdentities, 0)
	assert.Equal(t, "Cyclops", character.Name)
	assert.Equal(t, "Marvel", character.Publisher)
	assert.Len(t, character.IssueLinks, 5)
}

func TestCbExternalSource_CharacterPage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, err := os.Open("./testdata/cb_character_other_identities.html")
		if err != nil {
			panic(err)
		}
		bytes, err := ioutil.ReadAll(file)
		if err != nil {
			panic(err)
		}
		w.Write(bytes)
		w.WriteHeader(http.StatusOK)
	}))
	config := &CbExternalSourceConfig{}
	parser := NewCbParser(ts.URL)
	externalSource := CbExternalSource{
		httpClient: ts.Client(),
		parser:     parser,
		config:     config,
	}
	character, err := externalSource.CharacterPage(fmt.Sprintf("%s/character.php?ID=82321", ts.URL))
	if err != nil {
		t.Error(err)
	}
	assert.Len(t, character.OtherIdentities, 3)
	assert.Equal(t, "Emma Grace Frost", character.Name)
	assert.Equal(t, "Marvel", character.Publisher)
}

func TestCbExternalSource_Character_Maddog(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(HandleMadDogHttpCalls))
	config := &CbExternalSourceConfig{}
	parser := NewCbParser(ts.URL)
	externalSource := &CbExternalSource{
		httpClient: ts.Client(),
		parser:     parser,
		config:     config,
	}
	character, err := externalSource.Character(fmt.Sprintf("%s/character.php?ID=82321", ts.URL), func(id string) bool {
		return true
	})
	if err != nil {
		t.Error(err)
	}
	assert.Len(t, character.OtherIdentities, 0)
	assert.Equal(t, "Martin 'Mad Dog' Hawkins", character.Name)
	assert.Len(t, character.Issues, 1)
	for _, issue := range character.Issues {
		assert.Empty(t, issue.Number)
		assert.NotEmpty(t, issue.SeriesId)
		assert.NotEmpty(t, issue.Series)
		assert.NotEmpty(t, issue.Vendor)
		assert.NotEmpty(t, issue.Id)
		assert.True(t, issue.PublicationDate.Year() > 1)
		assert.True(t, issue.OnSaleDate.Year() > 1)
	}
}

func TestAdultIssue(t *testing.T) {
	cbdb := NewCbExternalSource(http.DefaultClient, &CbExternalSourceConfig{})
	ish, err := cbdb.Issue("http://comicbookdb.com/issue.php?ID=234957")
	assert.Nil(t, err)
	assert.Equal(t, "234957", ish.Id)
}

func TestThereAre(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, err := os.Open("./testdata/cb_issue_there_are.html")
		if err != nil {
			panic(err)
		}
		bytes, err := ioutil.ReadAll(file)
		if err != nil {
			panic(err)
		}
		w.Write(bytes)
		w.WriteHeader(http.StatusOK)
	}))
	cbdb := NewCbExternalSource(ts.Client(), &CbExternalSourceConfig{})
	ish, err := cbdb.Issue(ts.URL)
	assert.Nil(t, err)
	assert.Equal(t, HC, ish.Format)
}
