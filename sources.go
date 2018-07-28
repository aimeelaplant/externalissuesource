package externalissuesource

import (
	"errors"
	"fmt"
	"github.com/aimeelaplant/externalissuesource/internal/stringutil"
	"github.com/avast/retry-go"
	"math"
	"net/http"
	"strings"
)

const cbSearchPath = "/search.php"

var (
	ErrIssueNotFound = errors.New("issue URL not found")
)

type ExternalSource interface {
	CharacterPage(url string) (*CharacterPage, error)
	Character(url string, doFetchIssue func(issueId string) bool) (*Character, error)
	SearchCharacter(query string) (CharacterSearchResult, error)
}

// Configuration options
type CbExternalSourceConfig struct {
	WorkerPoolLimit int            // Default is 20. Limit the amount of goroutines to parse issues for a character.
	RetryOpts       []retry.Option // A slice of options for retrying when getting an issue fails.
}

type CbExternalSource struct {
	httpClient *http.Client
	parser     ExternalSourceParser
	config     *CbExternalSourceConfig
}

type issueResult struct {
	Issue *Issue
	Error error
}

// Fetches the character page.
func (s *CbExternalSource) CharacterPage(url string) (*CharacterPage, error) {
	characterPage := new(CharacterPage)
	resp, err := s.httpClient.Get(url)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("got bad status code from URL %s: %d", url, resp.StatusCode))
	}
	characterPage, err = s.parser.Character(resp.Body)
	return characterPage, err
}

// Fetches the character from the URL and concurrently gets the issues that match true for the `doFetchIssue` callback.
func (s *CbExternalSource) Character(url string, doFetchIssue func(id string) bool) (*Character, error) {
	character := new(Character)
	characterPage, err := s.CharacterPage(url)
	if err != nil {
		return nil, err
	}
	character.Name = characterPage.Name
	character.Publisher = characterPage.Publisher
	character.OtherIdentities = characterPage.OtherIdentities

	issuesToFetch := make([]string, 0)
	for _, issueLink := range characterPage.IssueLinks {
		issueIdIndex := strings.LastIndex(issueLink, "=")
		if issueIdIndex != -1 {
			id := issueLink[issueIdIndex+1:]
			if doFetchIssue(id) {
				issuesToFetch = append(issuesToFetch, issueLink)
			}
		} else {
			return nil, errors.New(fmt.Sprintf("can't get issue ID from %s", issueLink))
		}
	}

	// setup a worker pool of about 20 links per pool
	poolLength := len(issuesToFetch)
	left := 0
	right := 0
	var concurrencyLimit int
	if s.config.WorkerPoolLimit == 0 {
		concurrencyLimit = 20
	} else {
		concurrencyLimit = s.config.WorkerPoolLimit
	}
	if len(issuesToFetch) > concurrencyLimit {
		poolLength = int(math.Ceil(float64(poolLength / concurrencyLimit)))
		right = concurrencyLimit
	} else {
		right = 1
		concurrencyLimit = 1
	}
	for i := 0; i <= poolLength; i++ {
		var chunked []string
		// if we're at the last chunk, make sure we grab the last X items
		if left+concurrencyLimit > len(issuesToFetch) {
			chunked = issuesToFetch[left:]
		} else {
			chunked = issuesToFetch[left:right]
		}
		issueCh := make(chan *issueResult, len(chunked))
		left += concurrencyLimit
		right += concurrencyLimit
		for x := range chunked {
			// Concurrently gets the page link to parse the page.
			go func(x int) {
				link := chunked[x]
				fmt.Println(fmt.Sprintf("LINK %s", link))
				retry.Do(func() error {
					issueResp, err := s.httpClient.Get(link)
					defer issueResp.Body.Close()
					if err != nil {
						return err
					}
					if issueResp.StatusCode != http.StatusOK && issueResp.StatusCode != http.StatusNotModified {
						if issueResp.StatusCode == http.StatusNotFound {
							issueCh <- &issueResult{Error: ErrIssueNotFound}
							return nil
						}
						err = errors.New(fmt.Sprintf("got status code %d from url %s. retrying.", issueResp.StatusCode, link))
						return err
					}
					issue, err := s.parser.Issue(issueResp.Body)
					if err != nil {
						if err == ErrMySqlConnect {
							return err
						} else {
							issueCh <- &issueResult{Error: err}
							return nil
						}
					}
					issueCh <- &issueResult{Issue: issue}
					return nil
				}, s.config.RetryOpts...)
			}(x)
		}
		for j := 0; j < len(chunked); j++ {
			issueResult := <-issueCh
			if issueResult.Error != nil {
				// all or nothing -- if there's an error, return it.
				if issueResult.Error != ErrIssueNotFound {
					return character, issueResult.Error
				}
			} else {
				character.AddIssue(*issueResult.Issue)
			}
		}
	}
	return character, nil
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
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return CharacterSearchResult{}, errors.New(fmt.Sprintf("got bad status code from search: %d", response.StatusCode))
	}
	if err != nil {
		return CharacterSearchResult{}, err
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
