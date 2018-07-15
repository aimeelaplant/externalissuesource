package externalissuesource

import (
	"errors"
	"fmt"
	"github.com/aimeelaplant/externalissuesource/internal/stringutil"
	"github.com/avast/retry-go"
	"go.uber.org/zap"
	"math"
	"net/http"
	"strings"
)

const cbdbSearchPath = "/search.php"

var (
	ErrIssueNotFound = errors.New("issue URL not found")
)

type ExternalSource interface {
	Character(url string) (*Character, error)
	SearchCharacter(query string) (CharacterSearchResult, error)
}

// Configuration options
type CbdbExternalSourceConfig struct {
	WorkerPoolLimit int            // Default is 20. Limit the amount of goroutines to parse issues for a character.
	RetryOpts       []retry.Option // A slice of options for retrying when getting an issue fails.
}

type CbdbExternalSource struct {
	httpClient *http.Client
	parser     ExternalSourceParser
	logger     *zap.Logger
	config     *CbdbExternalSourceConfig
}

type issueResult struct {
	Issue *Issue
	Error error
}

// Fetches the character from the URL and concurrently gets all the issues for the character.
func (s *CbdbExternalSource) Character(url string) (*Character, error) {
	character := new(Character)
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return character, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New(fmt.Sprintf("character from URL %s not found.", url))
	}
	characterPage, err := s.parser.Character(resp.Body)
	if err != nil {
		return character, err
	}
	character.Name = characterPage.Name
	character.Publisher = characterPage.Publisher

	// setup a worker pool of about 20 links per pool
	poolLength := len(characterPage.IssueLinks)
	left := 0
	right := 0
	var concurrencyLimit int
	if s.config.WorkerPoolLimit == 0 {
		concurrencyLimit = 20
	} else {
		concurrencyLimit = s.config.WorkerPoolLimit
	}
	s.logger.Info(fmt.Sprintf("%d character links to parse for %s", poolLength, character.Name))
	if len(characterPage.IssueLinks) > concurrencyLimit {
		poolLength = int(math.Ceil(float64(poolLength / concurrencyLimit)))
		right = concurrencyLimit
	} else {
		right = 1
		concurrencyLimit = 1
	}
	for i := 0; i <= poolLength; i++ {
		var chunked []string
		// if we're at the last chunk, make sure we grab the last X items
		if left+concurrencyLimit > len(characterPage.IssueLinks) {
			chunked = characterPage.IssueLinks[left:]
			s.logger.Info(fmt.Sprintf("at offset %d: for %s", left, character.Name))
		} else {
			chunked = characterPage.IssueLinks[left:right]
			s.logger.Info(fmt.Sprintf("at offset %d:%d for %s", left, right, character.Name))
		}
		issueCh := make(chan *issueResult, len(chunked))
		left += concurrencyLimit
		right += concurrencyLimit
		for x := range chunked {
			// Concurrently gets the page link to parse the page.
			go func(x int) {
				link := chunked[x]
				s.logger.Info(fmt.Sprintf("started goroutine for %s", link))
				retry.Do(func() error {
					issueResp, err := s.httpClient.Get(link)
					defer issueResp.Body.Close()
					if err != nil {
						s.logger.Fatal(fmt.Sprintf("error from %s: %s", link, err.Error()))
						return err
					}
					if issueResp.StatusCode != http.StatusOK && issueResp.StatusCode != http.StatusNotModified {
						if issueResp.StatusCode == http.StatusNotFound {
							s.logger.Info(fmt.Sprintf("got status code %d from url %s. skipping.", issueResp.StatusCode, link))
							issueCh <- &issueResult{Error: ErrIssueNotFound}
							return nil
						}
						err = errors.New(fmt.Sprintf("got status code %d from url %s. retrying.", issueResp.StatusCode, link))
						s.logger.Info(err.Error())
						return err
					}
					issue, err := s.parser.Issue(issueResp.Body)
					if err != nil {
						if err == ErrMySqlConnect {
							s.logger.Info(fmt.Sprintf("received connection issue from url: %s. error: %s. retrying.", link, err))
							return err
						} else {
							s.logger.Fatal(fmt.Sprintf("got parse issue for %s:  %s", link, err.Error()))
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
					s.logger.Error(issueResult.Error.Error())
					return character, issueResult.Error
				}
			} else {
				s.logger.Info(fmt.Sprintf("received %s", issueResult.Issue.Id))
				character.AddIssue(*issueResult.Issue)
			}
		}
	}

	return character, nil
}

// Performs a search on the provided query and returns the search result for found characters.
func (s *CbdbExternalSource) SearchCharacter(query string) (CharacterSearchResult, error) {
	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s", s.parser.BaseUrl(), cbdbSearchPath), nil)
	if err != nil {
		return CharacterSearchResult{}, err
	}
	q := request.URL.Query()
	q.Add("form_search", strings.TrimSpace(query))
	q.Add("form_searchtype", "Character")
	request.URL.RawQuery = q.Encode()
	request.Header.Add("Cookie", fmt.Sprintf("PHPSESSID=%s", stringutil.RandString(26)))
	response, err := s.httpClient.Do(request)
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

func NewCbdbExternalSource(httpClient *http.Client, parser ExternalSourceParser, logger *zap.Logger, config *CbdbExternalSourceConfig) ExternalSource {
	return &CbdbExternalSource{
		httpClient: httpClient,
		parser:     parser,
		logger:     logger,
		config:     config,
	}
}
