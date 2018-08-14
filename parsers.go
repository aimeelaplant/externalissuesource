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
	"log"
	"golang.org/x/text/encoding/charmap"
	"github.com/andybalholm/cascadia"
)

const (
	regMonths = "(January|February|March|April|May|June|July|August|September|October|November|December)"
)

var (
	// ErrRecordNotFound record not found error, happens when haven't find any matched data when looking up with a struct
	ErrConnection = errors.New("page returned connection issue")
	ErrParse        = errors.New("can't parse the page")
	cdDatePrefixMap  = map[string]bool {
		"Mid": true,
		"Early": true,
		"Late": true,
		"Spring": true,
		"Summer": true,
		"Fall": true,
		"Winter": true,
		"Annual": true,
		"Holiday": true,
		"Jan/Feb": true, // keep year
		"Mar/Apr": true,
		"May/Jun": true,
		"Jul/Aug": true,
		"Sep/Oct": true,
		"Nov/Dec": true,
		"Dec/Jan": true, // keep year
		"Feb/Mar": true, // keep year
		"Apr/May": true,
		"Jun/Jul": true,
		"Aug/Sep": true,
		"Oct/Nov": true,
		"Jan/Mar": true, // keep year
		"Apr/Jun": true,
		"Jul/Sep": true,
		"Oct/Dec": true,
		"Feb/Apr": true,
		"May/Jul": true,
		"Aug/Oct": true,
		"Nov/Jan": true, // keep year
		"Mar/May": true,
		"Jun/Aug": true,
		"Sep/Nov": true,
		"Dec/Feb": true, // keep year
	}
	keepYears = map[string]bool{
		"Jan/Feb": true,
		"Dec/Jan": true,
		"Feb/Mar": true,
		"Jan/Mar": true,
		"Nov/Jan": true,
		"Dec/Feb": true,
	}
	regMY = regexp.MustCompile(fmt.Sprintf(`^%s \d{4}$`, regMonths))
	regMDY = regexp.MustCompile(fmt.Sprintf(`^%s \d{1,2} \d{4}$`, regMonths))
	regmY = regexp.MustCompile(`^\w{3} \d{4}$`)
	regY = regexp.MustCompile(`^(\d{4})$`)
	// CBDB uses Windows 1252 encoding for their pages. Need to decode it all to UTF8!
	cbDecoder = charmap.ISO8859_1.NewDecoder()
	cbIssueFormats = map[Format]string{
		Standard: "Standard Comic Issue",
		TPB: "Trade Paperback",
		Manga: "Manga",
		HC: "Hardcover",
		OGN: "Original Graphic Novel",
		Web: "Webcomic",
		Anthology: "Anthology",
		Bookshelf: "Bookshelf",
		Magazine: "Magazine",
		DigitalMedia: "Digital Media",
		MiniComic: "Minicomic",
		Prestige: "Prestige Format",
		Ashcan: "Ashcan",
		Flipbook: "Flipbook",
		Fanzine: "Fanzine",
		Other: "Other Comic-Related Media",
	}
	cascadiaWidth884 = cascadia.MustCompile("table[width=\"884\"]")
	cascadiaStrongA = cascadia.MustCompile("strong, a")
	cascadiaWidth850 = cascadia.MustCompile("td[width=\"850\"]")
	cascadiaCrazy = cascadia.MustCompile("body > table > tbody > tr:nth-child(2) > td:nth-child(3) > table > tbody > tr")
	cascadiaAStrongSpan = cascadia.MustCompile("a, strong, span")
	cascadiaColSpan3 = cascadia.MustCompile("td[colspan=\"3\"]")
)

type IssueParser interface {
	Parse(body io.Reader) ([]Issue, error)
}

type ExternalIssueParser interface {
	Issue(body io.Reader) (*Issue, error)
}

type ExternalCharacterParser interface {
	Character(body io.Reader) (*CharacterPage, error)
}

type ExternalCharacterSearchParser interface {
	CharacterSearch(body io.Reader) (*CharacterSearchResult, error)
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
func (p *CbParser) Character(body io.Reader) (*CharacterPage, error) {
	utf8Body := cbDecoder.Reader(body)
	doc, err := goquery.NewDocumentFromReader(utf8Body)
	if err != nil {
		return nil, ErrParse
	}
	if strings.Contains(doc.Text(), "mysql_connect()") {
		return nil, ErrConnection
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
	otherIdentitiesSection := false
	issueAppearancesSection := false
	doc.FindMatcher(cascadiaWidth884).FindMatcher(cascadiaStrongA).Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		if strings.Contains(text, "Other Identities:") {
			otherIdentitiesSection = true
		}
		if strings.Contains(text, "Issue Appearances:") {
			issueAppearancesSection = true
			otherIdentitiesSection = false
		}
		hrefValue, hrefExists := s.Attr("href")
		if hrefExists && strings.Contains(text, "Previous Character") || strings.Contains(text, "Next Character") {
			issueAppearancesSection = false
		}
		if issueAppearancesSection && hrefExists && strings.HasPrefix(hrefValue, "issue.php?ID=") {
			issueLinks = append(issueLinks, fmt.Sprintf("%s/%s", p.BaseUrl(), hrefValue))
		}
		if otherIdentitiesSection && hrefExists && strings.HasPrefix(hrefValue, "character.php?ID=") {
			otherIdentities = append(otherIdentities, CharacterLink{Url: fmt.Sprintf("%s/%s", p.BaseUrl(), hrefValue), Name: s.Text()})
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
func (p *CbParser) CharacterSearch(body io.Reader) (*CharacterSearchResult, error) {
	utf8Body := cbDecoder.Reader(body)
	doc, err := goquery.NewDocumentFromReader(utf8Body)
	if err != nil {
		return nil, ErrParse
	}
	if strings.Contains(doc.Text(), "mysql_connect()") {
		return nil, ErrConnection
	}
	characterSearchResult := new(CharacterSearchResult)
	characterLinks := make([]CharacterLink, 0)

	doc.FindMatcher(cascadiaWidth850).Find("a").Each(func(i int, s *goquery.Selection) {
		hrefValue, exists := s.Attr("href")
		if exists && strings.HasPrefix(hrefValue, "character.php") {
			characterLink := CharacterLink{Name: strings.TrimSpace(s.Text()), Url: fmt.Sprintf("%s/%s", p.BaseUrl(), hrefValue)}
			characterLinks = append(characterLinks, characterLink)
		}
	})
	characterSearchResult.Results = characterLinks
	return characterSearchResult, nil
}

// Parses an issue page and returns the corresponding struct.
func (p *CbParser) Issue(body io.Reader) (*Issue, error) {
	utf8Body := cbDecoder.Reader(body)
	doc, err := goquery.NewDocumentFromReader(utf8Body)
	if err != nil {
		return nil, ErrParse
	}
	if strings.Contains(doc.Text(), "mysql_connect()") {
		return nil, ErrConnection
	}
	issue := new(Issue)

	doc.FindMatcher(cascadiaCrazy).FindMatcher(cascadiaAStrongSpan).Each(func(i int, s *goquery.Selection) {
		hrefValue, ex := s.Attr("href")
		if issue.Vendor == "" && ex && strings.HasPrefix(hrefValue, "publisher.php"){
			issue.Vendor = strings.TrimSpace(s.Text())
		}
		if issue.PublicationDate.Year() <= 1 && ex && strings.HasPrefix(hrefValue, "coverdate.php") {
			dualDate := false
			dateText := strings.TrimSpace(s.Text())
			var trimmedDateText string
			spaceIndex := strings.Index(dateText, " ")
			if spaceIndex != -1 && cdDatePrefixMap[dateText[:spaceIndex]] {
				if strings.Contains(dateText, "/") {
					dualDate = true
					trimmedDateText = dateText[strings.Index(dateText, "/")+1:]
				} else {
					issue.MonthUncertain = true
					trimmedDateText = dateText[strings.Index(dateText, " ")+1:]
				}
			} else {
				trimmedDateText = dateText
			}
			format := ""
			if regMY.MatchString(trimmedDateText) {
				format = "January 2006"
			} else if regMDY.MatchString(trimmedDateText) {
				format = "January 2 2006"
			} else if regmY.MatchString(trimmedDateText) {
				format = "Jan 2006"
			} else if regY.MatchString(trimmedDateText) {
				format = "2006"
			}
			pubDate, err := time.Parse(format, trimmedDateText)
			if err != nil {
				// fail silently but log it.
				log.Println(fmt.Sprintf("ERROR: %s", err))
			}
			issue.PublicationDate = pubDate
			if format != "2006" {
				// Determine the on sale date was 2 months ago.
				if !dualDate {
					issue.OnSaleDate = pubDate.AddDate(0, -2, 0)
				} else {
					// If it's a dual date, we wanna go 3 months back from the latest month.
					// Check if we should keep the year for the issue.
					if keepYears[dateText[0:7]] {
						// Corrects the issues that fall within a new year, such as Dec/Jan 1971. CBDB lists the year for December and not
						// the year for January. So we want January 1972 as the publication date and December 1971 as the sale date.
						issue.PublicationDate = pubDate.AddDate(1, 0, 0)
						issue.OnSaleDate = issue.PublicationDate.AddDate(0, -3, 0)
					} else {
						issue.PublicationDate = pubDate
						issue.OnSaleDate = pubDate.AddDate(0, -3, 0)
					}
				}
			} else {
				issue.OnSaleDate = issue.PublicationDate
				issue.MonthUncertain = true
			}
		}
		// Sometimes they lock cloning of the issue or editing the issue, so check the issue_history.php
		// to get the ID of the issue.
		if issue.Id == "" && ex && strings.HasPrefix(hrefValue, "issue_history.php") {
			equalSignIndex := strings.Index(hrefValue, "=")
			if equalSignIndex != -1 {
				issue.Id = hrefValue[equalSignIndex+1:]
			}
		}
		if issue.SeriesId == "" && ex && strings.HasPrefix(hrefValue, "title.php") {
			issue.Series = strings.TrimSpace(s.Text())
			if equalIndex := strings.Index(hrefValue, "="); equalIndex != -1 {
				issue.SeriesId = hrefValue[equalIndex+1:]
			}
		}
		classValue, ex := s.Attr("class")
		if ex && classValue == "page_subheadline test" {
			r := regexp.MustCompile("(Cover [A-Za-z])|(\\(2nd Printing\\))|(Variant)")
			issue.IsVariant = r.MatchString(strings.TrimSpace(s.Text()))
		}
		if issue.Number == "" && ex && classValue == "page_headline" {
			// Get the issue number.
			text := strings.TrimSpace(s.Text())
			if hashBangIndex := strings.LastIndex(text, "#"); hashBangIndex != -1 {
				issue.Number = text[hashBangIndex+1:]
			} else if strings.Contains(text, " - Annual") {
				if annualIndex := strings.LastIndex(text, "Annual"); annualIndex != -1 {
					issue.Number = text[annualIndex:]
				}
			}
		}
	})

	foundFormat := false
	doc.FindMatcher(cascadiaWidth850).FindMatcher(cascadiaColSpan3).Each(func(i int, s *goquery.Selection) {
		trimmedText := strings.TrimSpace(s.Text())
		formatIndex := strings.Index(trimmedText, "Format:")
		semiColonIndex := strings.LastIndex(trimmedText, ";")
		thereAreIndex := strings.Index(trimmedText, "Story Arc(s)")
		formatText := ""
		if formatIndex != -1 && semiColonIndex != -1 {
			formatText = trimmedText[formatIndex:semiColonIndex]
		}
		if formatIndex != -1 && thereAreIndex != -1 {
			formatText = trimmedText[formatIndex:thereAreIndex]
		}
		// If we didn't find it yet ...  grab all the text.
		if formatText == "" && formatIndex != -1 {
			formatText = trimmedText[formatIndex:]
		}
		if formatText != "" {
			for format, s := range cbIssueFormats {
				if strings.Contains(formatText, s) {
					foundFormat = true
					issue.Format = format
					break
				}
			}
		}
	})
	if !foundFormat {
		issue.Format = Unknown
	} else {
		if issue.Format == Standard {
			issue.IsIssue = true
		}
	}

	return issue, nil
}

func (p *CbParser) IssueLinks(body io.Reader) ([]string, error) {
	utf8Body := cbDecoder.Reader(body)
	doc, err := goquery.NewDocumentFromReader(utf8Body)
	if err != nil {
		return nil, ErrParse
	}
	if strings.Contains(doc.Text(), "mysql_connect()") {
		return nil, ErrConnection
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

func (p *CoParser) Parse(body io.Reader) ([]Issue, error) {
	issues, err := p.parseSlow(body)
	if err != nil {
		return nil, err
	}
	return issues, nil
}

func (p *CoParser) parseSlow(body io.Reader) ([]Issue, error) {
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
