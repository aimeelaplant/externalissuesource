package externalissuesource

import (
	"errors"
	"fmt"
	"github.com/aimeelaplant/externalissuesource/internal/stringutil"
	"net/http"
	"strings"
	"net"
	"time"
	"net/url"
)

const cbSearchPath = "/search.php"
const cbLoginUrl = "http://comicbookdb.com/login.php"
var cbdbUrl = &url.URL{
	Host: "http://comicbookdb.com",
}

type ExternalSource interface {
	Issue(url string) (*Issue, error)
	CharacterPage(url string) (*CharacterPage, error)
	SearchCharacter(query string) (CharacterSearchResult, error)
}

// Configuration options
type CbExternalSourceConfig struct {
	SessionId string
	CbOne string
	CbTwo string
}

type CbExternalSource struct {
	httpClient *http.Client
	parser     ExternalSourceParser
	config     *CbExternalSourceConfig
	isLoggedIn bool
}


// Fetches an issue from the issue page.
func (s *CbExternalSource) Issue(url string) (*Issue, error) {
	var issue *Issue
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("PHPSESSID", s.config.SessionId)
	req.AddCookie(&http.Cookie{
		Name: "PHPSESSID",
		Value: s.config.SessionId,
	})
	req.AddCookie(&http.Cookie{
		Name: "cbdb1",
		Value: s.config.CbOne,
	})
	req.AddCookie(&http.Cookie{
		Name: "cbdb2",
		Value: s.config.CbTwo,
	})
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("got bad status code from URL %s: %d", url, resp.StatusCode))
	}
	issue, err = s.parser.Issue(resp.Body)
	if err != nil {
		return nil, err
	}
	return issue, nil
}

// Fetches the character page.
func (s *CbExternalSource) CharacterPage(url string) (*CharacterPage, error) {
	characterPage := new(CharacterPage)
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("got bad status code from URL %s: %d", url, resp.StatusCode))
	}
	characterPage, err = s.parser.Character(resp.Body)
	return characterPage, err
}

// Performs a search on the provided query and returns the search result for found characters.
func (s *CbExternalSource) SearchCharacter(query string) (CharacterSearchResult, error) {
	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s", s.parser.BaseUrl(), cbSearchPath), nil)
	if err != nil {
		return CharacterSearchResult{}, err
	}
	q := request.URL.Query()
	q.Add("form_search", strings.TrimSpace(query))
	q.Add("form_searchtype", "Character")
	request.URL.RawQuery = q.Encode()
	request.Header.Add("Cookie", fmt.Sprintf("PHPSESSID=%s", stringutil.RandString(26)))
	response, err := s.httpClient.Do(request)
	if err != nil {
		return CharacterSearchResult{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return CharacterSearchResult{}, errors.New(fmt.Sprintf("got bad status code from search: %d", response.StatusCode))
	}
	characterSearchResult, err := s.parser.CharacterSearch(response.Body)
	if err != nil {
		return CharacterSearchResult{}, err
	}
	return *characterSearchResult, nil
}

func NewCbExternalSource(httpClient *http.Client, config *CbExternalSourceConfig) ExternalSource {
	return &CbExternalSource{
		httpClient: httpClient,
		parser:     &CbParser{},
		config:     config,
	}
}

func NewHttpClient() (*http.Client) {
	tp := http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   45 * time.Second,
			KeepAlive: 90 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   30 * time.Second,
		ExpectContinueTimeout: 10 * time.Second,
	}
	client := &http.Client{
		Transport: &tp,
		Timeout: 45 * time.Second,
	}
	return client
}