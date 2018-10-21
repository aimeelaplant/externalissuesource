package externalissuesource

import (
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"regexp"
	"strings"
	"time"
	"log"
	"golang.org/x/text/encoding/charmap"
	"github.com/andybalholm/cascadia"
)

const (
	cbUrl = "http://comicbookdb.com"
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
	// CBDB uses Windows 1252 encoding for their pages. Need to decode it all to UTF8!
	// Create new decoder every time to make this method concurrent safe.
	doc, err := goquery.NewDocumentFromReader(charmap.ISO8859_1.NewDecoder().Reader(body))
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

	// get the other name
	doc.FindMatcher(cascadia.MustCompile("table[width=\"884\"]")).Each(func(i int, selection *goquery.Selection) {
		selectionText := selection.Text()
		if idx := strings.Index(selectionText, "Real Name: "); idx == -1 {
			return
		}
		for _, s := range strings.SplitAfter(selectionText, "\n") {
			if idx2 := strings.Index(s, "Real Name: "); idx2 != -1 && s != "Real Name: " && len(s) > idx2+11{
				// Get the real name
				characterPage.OtherName = strings.TrimSpace(s[idx2+11:])
				break
			}
		}
	})

	// Get the issue links
	issueLinks := make([]string, 0)
	otherIdentities := make([]CharacterLink, 0)
	otherIdentitiesSection := false
	issueAppearancesSection := false
	doc.FindMatcher(cascadia.MustCompile("table[width=\"884\"]")).FindMatcher(cascadia.MustCompile("strong, a")).Each(func(i int, s *goquery.Selection) {
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
	doc, err := goquery.NewDocumentFromReader(charmap.ISO8859_1.NewDecoder().Reader(body))
	if err != nil {
		return nil, ErrParse
	}
	if strings.Contains(doc.Text(), "mysql_connect()") {
		return nil, ErrConnection
	}
	characterSearchResult := new(CharacterSearchResult)
	characterLinks := make([]CharacterLink, 0)

	doc.FindMatcher(cascadia.MustCompile("td[width=\"850\"]")).Find("a").Each(func(i int, s *goquery.Selection) {
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
	doc, err := goquery.NewDocumentFromReader(charmap.ISO8859_1.NewDecoder().Reader(body))
	if err != nil {
		return nil, ErrParse
	}
	if strings.Contains(doc.Text(), "mysql_connect()") {
		return nil, ErrConnection
	}
	issue := new(Issue)

	doc.FindMatcher(cascadia.MustCompile("body > table > tbody > tr:nth-child(2) > td:nth-child(3) > table > tbody > tr")).FindMatcher(cascadia.MustCompile("a, strong, span")).Each(func(i int, s *goquery.Selection) {
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
			trimmedDateText = strings.TrimSpace(trimmedDateText)
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
		if issue.IsReprint == false && strings.Contains(strings.ToLower(s.Text()), "this is a version of the following issue") {
			issue.IsReprint = true
		}
	})

	foundFormat := false
	doc.FindMatcher(cascadia.MustCompile("td[width=\"850\"]")).FindMatcher(cascadia.MustCompile("td[colspan=\"3\"]")).Each(func(i int, s *goquery.Selection) {
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
	}

	return issue, nil
}

func (p *CbParser) IssueLinks(body io.Reader) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(charmap.ISO8859_1.NewDecoder().Reader(body))
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

func NewCbParser(baseUrl string) ExternalSourceParser {
	cbParser := CbParser{baseUrl: baseUrl}
	return &cbParser
}
