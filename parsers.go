package externalissuesource

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/aimeelaplant/externalissuesource/internal/dateutil"
	"github.com/aimeelaplant/externalissuesource/internal/stringutil"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	regMonths = "(January|February|March|April|May|June|July|August|September|October|November|December)"
)

var (
	// ErrRecordNotFound record not found error, happens when haven't find any matched data when looking up with a struct
	ErrMySqlConnect = errors.New("page returned mysql_connect() connection issue")
	ErrParse        = errors.New("can't parse the page")
	cbDatePrefixes  = []string{
		"Mid ",
		"Early ",
		"Late ",
		"Spring ",
		"Summer ",
		"Fall ",
		"Winter ",
		"Annual ",
		"Jan/Feb ",
		"Mar/Apr ",
		"May/Jun ",
		"Jul/Aug ",
		"Sep/Oct ",
		"Nov/Dec ",
		"Holiday ",
		"Dec/Jan ",
		"Feb/Mar ",
		"Apr/May ",
		"Jun/Jul ",
		"Aug/Sep ",
		"Oct/Nov ",
		"Jan/Mar ",
		"Apr/Jun ",
		"Jul/Sep ",
		"Oct/Dec ",
		"Feb/Apr ",
		"May/Jul ",
		"Aug/Oct ",
		"Nov/Jan ",
		"Mar/May ",
		"Jun/Aug ",
		"Sep/Nov ",
		"Dec/Feb ",
	}
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

// This struct implements parsing entities from the cb source.
type CbParser struct {
	baseUrl string // The base URL for constructing links. Default is http://comicbookdb.com if not provided.
}

// Parses a character's page and returns the corresponding struct.
// The caller is responsible for closing the body after it's done.
func (p *CbParser) Character(body io.ReadCloser) (*CharacterPage, error) {
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
	if firstParen != -1 {
		characterPage.Name = titleText[:firstParen]
		if secondParen != -1 {
			characterPage.Publisher = titleText[firstParen+2 : secondParen]
		}
	} else {
		characterPage.Name = titleText
	}
	characterPage.Title = titleText

	// Get the issue links
	issueLinks := make([]string, 0)
	otherIdentities := make([]CharacterLink, 0)
	doc.Find("table").Each(func(i int, s *goquery.Selection) {
		width, exists := s.Attr("width")
		if exists && width == "884" {
			s.Find("a").Each(func(i int, a *goquery.Selection) {
				otherIdentitiesSectionIsEnded := false
				if strings.Contains(a.Text(), "Issue Appearances") || strings.Contains(a.Text(), "Previous Character") || strings.Contains(a.Text(), "Next Character") {
					otherIdentitiesSectionIsEnded = true
				}
				hrefValue, hrefExists := a.Attr("href")
				if hrefExists && strings.Contains(hrefValue, "issue.php?ID=") {
					issueLinks = append(issueLinks, fmt.Sprintf("%s/%s", p.BaseUrl(), hrefValue))
				}
				if hrefExists && strings.Contains(hrefValue, "character.php?ID=") && !otherIdentitiesSectionIsEnded {
					otherIdentities = append(otherIdentities, CharacterLink{Url: fmt.Sprintf("%s/%s", p.BaseUrl(), hrefValue), Name: a.Text()})
				}
			})
		}
	})
	characterPage.IssueLinks = issueLinks
	characterPage.OtherIdentities = otherIdentities
	return characterPage, nil
}

// Gets the base URL for constructing links for the parser.
func (p *CbParser) BaseUrl() string {
	if p.baseUrl == "" {
		p.baseUrl = cbUrl
	}
	return p.baseUrl
}

// Parses the links to character profiles and their names from the search page.
// The caller is responsible for closing the body after it's done.
func (p *CbParser) CharacterSearch(body io.ReadCloser) (*CharacterSearchResult, error) {
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
func (p *CbParser) Issue(body io.ReadCloser) (*Issue, error) {
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
			dateText := strings.TrimSpace(s.Text())
			for _, prefix := range cbDatePrefixes {
				if !strings.Contains(dateText, "/") {
					if strings.Contains(dateText, prefix) {
						dateText = strings.TrimPrefix(dateText, prefix)
						issue.MonthUncertain = true
					}
				} else {
					dateText = dateText[0:strings.Index(dateText, "/")] + dateText[strings.Index(dateText, " "):]
					// @TODO: Month may not be so uncertain!
					issue.MonthUncertain = true
				}
			}
			format := ""
			if regexp.MustCompile(fmt.Sprintf(`^%s \d{4}$`, regMonths)).MatchString(dateText) {
				format = "January 2006"
			} else if regexp.MustCompile(fmt.Sprintf(`^%s \d{1,2} \d{4}$`, regMonths)).MatchString(dateText) {
				format = "January 2 2006"
			} else if regexp.MustCompile(`^\w{3} \d{4}$`).MatchString(dateText) {
				format = "Jan 2006"
			} else if regexp.MustCompile(`^(\d{4})$`).MatchString(dateText) {
				format = "2006"
				issue.MonthUncertain = true
			}
			pubDate, _ := time.Parse(format, dateText)
			issue.PublicationDate = pubDate
			if format != "2006" {
				// Determine the on sale date was 2 months ago.
				issue.OnSaleDate = pubDate.AddDate(0, -2, 0)
			} else {
				issue.OnSaleDate = pubDate
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

	// If the publication date wasn't parsed, return an error.
	if issue.PublicationDate.Year() <= 1 {
		// TODO: Wtf? 
		issue.MonthUncertain = true
	}

	return issue, nil
}

func (p *CbParser) IssueLinks(body io.ReadCloser) ([]string, error) {
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

type CoParser struct {
}

const cbUrl = "http://comicbookdb.com"

type IssueResult struct {
	Issue *Issue
	Error error
}

func (p *CoParser) Parse(body io.ReadCloser) ([]Issue, error) {
	issues, err := p.parseSlow(body)
	if err != nil {
		return nil, err
	}
	return issues, nil
}

func (p *CoParser) parseSlow(body io.ReadCloser) ([]Issue, error) {
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

func (p *CoParser) parseLine(line []string) (*Issue, error) {
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

func (p *CoParser) determineSaleDate(dateObj issueDates) (time.Time, error) {
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

func (p *CoParser) cleanDateStrings(date string) string {
	var trim []string
	trim = append(trim, "[", "]", "Early", "Late")
	return stringutil.TrimStrings(date, trim)
}

func (p *CoParser) getBestDate(dates ...string) (time.Time, error) {
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

func NewCoParser() IssueParser {
	CoParser := CoParser{}
	return &CoParser
}

func NewCbParser(baseUrl string) ExternalSourceParser {
	cbParser := CbParser{baseUrl: baseUrl}
	return &cbParser
}
