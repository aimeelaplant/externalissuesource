package externalissuesource

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/aimeelaplant/externalissuesource/internal/dateutil"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
	"github.com/aimeelaplant/externalissuesource/internal/stringutil"
)

var (
	// ErrRecordNotFound record not found error, happens when haven't find any matched data when looking up with a struct
	ErrMySqlConnect = errors.New("page returned mysql_connect() connection issue")
	ErrParse        = errors.New("can't parse the page")
)

type IssueParser interface {
	Parse(body io.ReadCloser) ([]Issue, error)
}

type ExternalIssueParser interface {
	Issue(body io.ReadCloser) (*Issue, error)
}

type ExternalCharacterParser interface {
	Character(body io.ReadCloser) (*CharacterPage, error)
}

type ExternalCharacterSearchParser interface {
	CharacterSearch(body io.ReadCloser) (*CharacterSearchResult, error)
}

// An interface that defines parsing entities from a remote external source.
type ExternalSourceParser interface {
	ExternalIssueParser
	ExternalCharacterParser
	ExternalCharacterSearchParser
	BaseUrl() string
}

// This struct implements parsing entities from the cbdb.
type CbdbParser struct {
	baseUrl string // The base URL for constructing links. Default is http://comicbookdb.com if not provided.
}

// Parses a character's page and returns the corresponding struct.
// The caller is responsible for closing the body after it's done.
func (p *CbdbParser) Character(body io.ReadCloser) (*CharacterPage, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, ErrParse
	}
	if strings.Contains(doc.Text(), "mysql_connect()") {
		return nil, ErrMySqlConnect
	}
	// Get the name, publisher, and title of page.
	selection := doc.Find(".page_headline").Not("").First()
	titleText := selection.Text()
	characterPage := new(CharacterPage)
	firstParen := strings.Index(titleText, " (")
	secondParen := strings.Index(titleText, ")")
	characterPage.Name = titleText[:firstParen]
	characterPage.Publisher = titleText[firstParen+2 : secondParen]
	characterPage.Title = titleText

	// Get the issue links
	issueLinks := make([]string, 0)
	doc.Find("table").Each(func(i int, s *goquery.Selection) {
		width, exists := s.Attr("width")
		if exists && width == "884" {
			s.Find("a").Each(func(i int, a *goquery.Selection) {
				hrefValue, hrefExists := a.Attr("href")
				if hrefExists && strings.Contains(hrefValue, "issue.php?ID=") {
					issueLinks = append(issueLinks, fmt.Sprintf("%s/%s", p.BaseUrl(), hrefValue))
				}
			})
		}
	})
	characterPage.IssueLinks = issueLinks
	return characterPage, nil
}

// Gets the base URL for constructing links for the parser.
func (p *CbdbParser) BaseUrl() string {
	if p.baseUrl == "" {
		p.baseUrl = cbdbUrl
	}
	return p.baseUrl
}

// Parses the links to character profiles and their names from the search page.
// The caller is responsible for closing the body after it's done.
func (p *CbdbParser) CharacterSearch(body io.ReadCloser) (*CharacterSearchResult, error) {
	characterSearchResult := new(CharacterSearchResult)
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, ErrParse
	}
	if strings.Contains(doc.Text(), "mysql_connect()") {
		return nil, ErrMySqlConnect
	}
	characterLinks := make([]CharacterLink, 0)
	doc.Find("td").Each(func(i int, s *goquery.Selection) {
		tdWidth, exists := s.Attr("width")
		// Only get the results from the main td.
		if exists && tdWidth == "850" {
			s.ChildrenFiltered("a").Each(func(i int, s2 *goquery.Selection) {
				hrefValue, exists := s2.Attr("href")
				if exists && strings.Contains(hrefValue, "character.php") {
					characterLink := CharacterLink{Name: strings.TrimSpace(s2.Text()), Url: fmt.Sprintf("%s/%s", p.BaseUrl(), hrefValue)}
					characterLinks = append(characterLinks, characterLink)
				}
			})
		}
	})
	characterSearchResult.Results = characterLinks
	return characterSearchResult, nil
}

// Parses an issue page and returns the corresponding struct.
// The caller is responsible for closing the body after it's done.
func (p *CbdbParser) Issue(body io.ReadCloser) (*Issue, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, ErrParse
	}
	if strings.Contains(doc.Text(), "mysql_connect()") {
		return nil, ErrMySqlConnect
	}
	issue := new(Issue)

	pageHeadlineSelection := doc.Find(".page_headline")

	pageHeadlineSelection.Children().Each(func(i int, s *goquery.Selection) {
		hrefValue, ex := s.Attr("href")
		if ex && strings.Contains(hrefValue, "title.php") {
			titleIdIndex := strings.LastIndex(hrefValue, "=")
			if titleIdIndex != -1 {
				issue.SeriesId = hrefValue[titleIdIndex+1:]
			}
		}
	})

	doc.Find("td").Children().Each(func(i int, s *goquery.Selection) {
		hrefValue, ex := s.Attr("href")
		if ex && strings.Contains(hrefValue, "coverdate.php") {
			pubDate, err := time.Parse("January 2006", strings.TrimSpace(s.Text()))
			if err == nil {
				issue.PublicationDate = pubDate
				// Determine the on sale date with 2 months ago.
				issue.OnSaleDate = pubDate.AddDate(0, -2, 0)
			} else {
				pubDate, err := time.Parse("2006", strings.TrimSpace(s.Text()))
				if err == nil {
					issue.PublicationDate = pubDate
					issue.OnSaleDate = pubDate
				} else {
					pubDate, err := time.Parse("January 1 2006", strings.TrimSpace(s.Text()))
					if err == nil {
						issue.PublicationDate = pubDate
						issue.PublicationDate = pubDate
						// Determine the on sale date with 2 months ago.
						issue.OnSaleDate = pubDate.AddDate(0, -2, 0)
					}
				}
			}
		}

		// Get the ID of the issue.
		if ex && strings.Contains(hrefValue, "issue_clone.php") {
			equalSignIndex := strings.LastIndex(hrefValue, "=")
			if equalSignIndex != -1 {
				issue.Id = hrefValue[equalSignIndex+1:]
			}
		}

		// Check if the issue is a variant.
		classValue, ex := s.Attr("class")
		if ex && classValue == "page_subheadline test" {
			r, _ := regexp.Compile("(Cover [A-Za-z])|(\\(2nd Printing\\))|(Variant)")
			issue.IsVariant = r.MatchString(s.Text())
		}

		if ex && strings.Contains(hrefValue, "publisher.php") {
			issue.Vendor = strings.TrimSpace(s.Text())
		}

		s.Children().Each(func(i int, s *goquery.Selection) {
			if strings.Contains(s.Text(), "Standard Comic Issue") {
				issue.IsIssue = strings.Contains(s.Text(), "Standard Comic Issue")
			}
			hrefValue, ex := s.Attr("href")
			if ex && strings.Contains(hrefValue, "title.php") {
				issue.Series = strings.TrimSpace(s.Text())
			}
			if ex && strings.Contains(hrefValue, "issue_number.php") {
				issue.Number = strings.TrimSpace(strings.TrimLeft(s.Text(), "#"))
			}
		})

	})

	// Get the issue number in case there was no link to the `issue_number.php` and it wasn't parsed.
	if issue.Number == "" {
		// Get the issue number.
		hashBangIndex := strings.LastIndex(pageHeadlineSelection.Text(), "#")
		if hashBangIndex != -1 {
			issueNumber := pageHeadlineSelection.Text()[hashBangIndex+1:]
			issue.Number = issueNumber
		}
	}

	return issue, nil
}

func (p *CbdbParser) IssueLinks(body io.ReadCloser) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, ErrParse
	}
	if strings.Contains(doc.Text(), "mysql_connect()") {
		return nil, ErrMySqlConnect
	}
	issueLinks := make([]string, 0)
	doc.Find("table").Each(func(i int, s *goquery.Selection) {
		width, exists := s.Attr("width")
		if exists && width == "884" {
			s.Find("a").Each(func(i int, a *goquery.Selection) {
				hrefValue, hrefExists := a.Attr("href")
				if hrefExists && strings.Contains(hrefValue, "issue.php?ID=") {
					issueLinks = append(issueLinks, fmt.Sprintf("%s/%s", p.BaseUrl(), hrefValue))
				}
			})
		}
	})
	return issueLinks, nil
}

type ComicsOrgParser struct {
}

const cbdbUrl = "http://comicbookdb.com"

type IssueResult struct {
	Issue *Issue
	Error error
}

func (p *ComicsOrgParser) Parse(body io.ReadCloser) ([]Issue, error) {
	issues, err := p.parseSlow(body)
	if err != nil {
		return nil, err
	}
	return issues, nil
}

func (p *ComicsOrgParser) parseSlow(body io.ReadCloser) ([]Issue, error) {
	r := csv.NewReader(bufio.NewReader(body))
	var issues []Issue
	var lineCount = 0
	// File gets streamed in memory ... OK for now.
	lines, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	for _, line := range lines {
		if err == io.EOF {
			break
		} else if err != nil {
			break
		}
		if lineCount == 0 {
			lineCount++
			continue
		}
		lineCount++
		issue, err := p.parseLine(line)
		if err == nil && !issue.IsVariant {
			issues = append(issues, *issue)
		}
	}
	return issues, nil
}

func (p *ComicsOrgParser) parseLine(line []string) (*Issue, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(line[0]), 10, 64)
	if err != nil {
		return nil, err
	}
	seriesId, err := strconv.ParseInt(strings.TrimSpace(line[30]), 10, 64)
	if err != nil {
		return nil, err
	}
	// line[17] = publication date, line[18] = key date, line[19] = sale date
	publicationDate, err := p.getBestDate(line[17], line[18])
	if err != nil {
		return nil, err
	}
	onSaleDate, err := p.determineSaleDate(issueDates{
		OnSaleDate:      line[19],
		PublicationDate: line[17],
		KeyDate:         line[18],
	})
	if err != nil {
		return nil, err
	}
	searchResultRow := Issue{
		Vendor:          "comics.org",
		Id:              string(id),
		Number:          strings.TrimSpace(line[1]),
		IsVariant:       len(strings.TrimSpace(line[11])) > 0,
		OnSaleDate:      onSaleDate,
		SeriesId:        string(seriesId),
		PublicationDate: publicationDate,
	}
	return &searchResultRow, nil
}

func (p *ComicsOrgParser) determineSaleDate(dateObj issueDates) (time.Time, error) {
	publicationDate, err := p.getBestDate(dateObj.PublicationDate, dateObj.KeyDate)
	if err != nil {
		return publicationDate, err
	}
	if dateObj.OnSaleDate != "" {
		saleDate, err := p.getBestDate(dateObj.OnSaleDate)
		if err != nil {
			return saleDate, err
		}
		if dateutil.CompareMonths(saleDate, publicationDate) <= -2 {
			return saleDate, nil
		}
	}
	return publicationDate.AddDate(0, -2, 0), nil
}

func (p *ComicsOrgParser) cleanDateStrings(date string) string {
	var trim []string
	trim = append(trim, "[", "]", "Early", "Late")
	return stringutil.TrimStrings(date, trim)
}

func (p *ComicsOrgParser) getBestDate(dates ...string) (time.Time, error) {
	for _, dateString := range dates {
		dateString := p.cleanDateStrings(dateString)
		if dateString == "" || len(dateString) < 7 {
			continue
		}
		// case when date has -
		if strings.Contains(dateString, "-") {
			// 2006-07
			if len(dateString) == 7 {
				if strings.Contains(dateString, "-00") {
					return time.Parse("2006", strings.TrimSuffix(dateString, "-00"))
				} else {
					return time.Parse("2006-01", dateString)
				}
			} else {
				if strings.Contains(dateString, "-00") {
					date, err := time.Parse("2006-01", strings.TrimSuffix(dateString, "-00"))
					if err != nil {
						return time.Parse("2006", strings.Replace(dateString, "-00", "", -1))
					}
					return date, nil
				}
				return time.Parse("2006-01-02", dateString)
			}
		} else {
			return time.Parse("January 2006", dateString)
		}
	}
	return time.Now(), errors.New("cannot parse the date strings")
}

func NewComicsOrgParser() IssueParser {
	comicsOrgParser := ComicsOrgParser{}
	return &comicsOrgParser
}

func NewCbdbParser(baseUrl string) ExternalSourceParser {
	cbdbParser := CbdbParser{baseUrl: baseUrl}
	return &cbdbParser
}
