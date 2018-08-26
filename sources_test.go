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

var config = &CbExternalSourceConfig{
	SessionId: "43lc83m4e51adm1u0v0c13j3j6",
	CbOne: "272498",
	CbTwo: "d5842aa60081b6881103ab5da286c511"}

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

func TestAdultIssue(t *testing.T) {
	client := NewHttpClient()
	cbdb := NewCbExternalSource(client, config)
	ish, err := cbdb.Issue("http://comicbookdb.com/issue.php?ID=234957")
	assert.Nil(t, err)
	assert.Equal(t, "234957", ish.Id)
}

func TestParsePage(t *testing.T) {
	// http://comicbookdb.com/issue.php?ID=22328
	cbdb := NewCbExternalSource(http.DefaultClient, config)
	ish, err := cbdb.Issue("http://comicbookdb.com/issue.php?ID=22328")
	assert.Nil(t, err)
	assert.Equal(t, "22328", ish.Id)
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
