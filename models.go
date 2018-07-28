package externalissuesource

import "time"

// A transformed object from a remote source with all the issues attached.
type Character struct {
	Publisher string
	Name      string
	Issues    []Issue
	OtherIdentities []CharacterLink
}

func (c *Character) AddIssue(issue Issue) []Issue {
	c.Issues = append(c.Issues, issue)
	return c.Issues
}

// An issue, such as a comic book issue, with the publication date and sale date.
type Issue struct {
	Series          string
	Vendor          string	  // The publisher of the issue.
	Id              string    // unique identifier for the issue.
	Number          string    // The number of the issue - for example, Astonishing X-Men 1 with `1` being the issue number.
	IsVariant       bool      // Whether it's a variant, 2nd printing, etc.
	PublicationDate time.Time // The cover date or publication date that the issue was published.
	OnSaleDate      time.Time // The release date or on sale date that the issue was published.
	SeriesId        string    // unique identifier for the series/title of the issue.
	IsIssue         bool      // Sometimes an external source carries a TPB, graphic novel, or regular book.
	// This flag indicates it's an actual comic book issue.
}

// Issue dates for internal use for parsing a comics.org CSV.
type issueDates struct {
	OnSaleDate      string // empty string or 2015-08-12 or 2015-08-00
	PublicationDate string // empty string or [November 2015] or November 2015  or 2000 or [Early] June 2004
	KeyDate         string // empty string or 2015-10-00 or 2000-00-00 or
}

// Represents a character's detailed paged.
type CharacterPage struct {
	Publisher       string          // The name of the publisher.
	Title           string          // The title of the page
	Name            string          // The name of the character
	IssueLinks      []string        // Links to a character's issues.
	OtherIdentities []CharacterLink // Links to other identities for the character.
}

// A link to a character with its URL and name from the search results.
type CharacterLink struct {
	Url  string
	Name string
}

// Represents the search results returned for querying a character's name.
type CharacterSearchResult struct {
	Results []CharacterLink
}
