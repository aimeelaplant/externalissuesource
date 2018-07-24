package externalissuesource

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func TestCoParser_cleanDateStrings(t *testing.T) {
	parser := CoParser{}
	expected := "December 1998"
	var s = parser.cleanDateStrings("[December] 1998")
	assert.Equal(t, expected, s)
	s = parser.cleanDateStrings(" December 1998 ")
	assert.Equal(t, expected, s)
}

func TestCoParser_getBestDate(t *testing.T) {
	var date1 = "2017-05-17"
	var date2 = "2017-07-00"
	var date3 = "7/1/2017"
	parser := CoParser{}
	date, err := parser.getBestDate(date1, date2, date3)
	if date.Year() != 2017 || date.Month().String() != "May" {
		t.Errorf("Got wrong date %s for %s", date, date1)
	}
	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestTestCoParser_getBestDateForIncompleteDay(t *testing.T) {
	var date1 = ""
	var date2 = "2000-12-00"
	var date3 = "December 2000"
	parser := CoParser{}
	date, err := parser.getBestDate(date1, date2, date3)
	if date.Year() != 2000 || date.Month().String() != "December" {
		t.Errorf("Got wrong date %s for %s", date, date1)
	}
	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestCoParser_getBestDateForIncompleteMonth(t *testing.T) {
	var date1 = ""
	var date2 = "2009-00-00"
	var date3 = "2009"
	parser := CoParser{}
	date, err := parser.getBestDate(date1, date2, date3)
	if err != nil {
		t.Errorf(err.Error())
	}
	if date.Year() != 2009 {
		t.Errorf("Got wrong date %s for %s", date, date1)
	}
}

func TestCoParser_determineSaleDate(t *testing.T) {
	issueDatesObj := issueDates{
		OnSaleDate:      "",
		PublicationDate: "May 2000",
		KeyDate:         "May 2000",
	}
	parser := CoParser{}
	saleDate, err := parser.determineSaleDate(issueDatesObj)
	if err != nil {
		t.Errorf(err.Error())
	}
	if saleDate.Year() != 2000 || saleDate.Month().String() != "March" {
		t.Errorf("Expected sale date of March 2000, got: %s", saleDate)
	}

	issueDatesObj = issueDates{
		OnSaleDate:      "2000-03-00",
		PublicationDate: "May 2000",
		KeyDate:         "May 2000",
	}

	saleDate, err = parser.determineSaleDate(issueDatesObj)
	if err != nil {
		t.Errorf(err.Error())
	}
	if saleDate.Year() != 2000 || saleDate.Month().String() != "March" {
		t.Errorf("Expected sale date of March 2000, got: %s", saleDate)
	}
}

func TestCbParser_parse(t *testing.T) {
	file, err := os.Open("./testdata/cb_character.html")
	if err != nil {
		t.Errorf(err.Error())
	}
	parser := CbParser{}
	character, err := parser.Character(file)
	if err != nil {
		t.Errorf(err.Error())
	}
	assert.Len(t, character.IssueLinks, 10)
}

func TestCbParser_Issue_TheEnd(t *testing.T) {
	file, err := os.Open("./testdata/cb_issue_end.html")
	if err != nil {
		t.Errorf(err.Error())
	}
	parser := CbParser{}
	issue, err := parser.Issue(file)
	assert.Nil(t, err)
	assert.Equal(t, "1", issue.Number)
	assert.True(t, issue.IsIssue)
	assert.Equal(t, "26025", issue.Id)
	assert.Equal(t, "3152", issue.SeriesId)
	assert.Equal(t, "X-Men: The End: Book 1: Dreamers & Demons (2004)", issue.Series)
	assert.Equal(t, "Marvel", issue.Vendor)
	assert.Equal(t, 2004, int(issue.OnSaleDate.Year()))
	assert.Equal(t, time.Month(8), issue.OnSaleDate.Month())
	assert.Equal(t, 2004, int(issue.PublicationDate.Year()))
	assert.Equal(t, 10, int(issue.PublicationDate.Month()))
}

func TestCbParser_parseIssueLinks(t *testing.T) {
	file, err := os.Open("./testdata/cb_character.html")
	if err != nil {
		t.Errorf(err.Error())
	}
	parser := CbParser{}
	links, err := parser.IssueLinks(file)
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(links) != 10 {
		t.Errorf("Expected 10, got: %d", len(links))
	}
}

func TestCbParser_parseIssuePages(t *testing.T) {
	file, err := os.Open("./testdata/cb_issue.html")
	defer file.Close()
	if err != nil {
		t.Errorf(err.Error())
	}
	parser := CbParser{}
	issue, err := parser.Issue(file)
	assert.False(t, issue.IsVariant)
	assert.True(t, issue.IsIssue)
	assert.Equal(t, 2007, int(issue.PublicationDate.Year()))
	assert.Equal(t, 10, int(issue.PublicationDate.Month()))
	assert.Equal(t, 2007, int(issue.OnSaleDate.Year()))
	assert.Equal(t, 8, int(issue.OnSaleDate.Month()))
	assert.Equal(t, "Astonishing X-Men (2004)", issue.Series)
	assert.Equal(t, "22", issue.Number)
	assert.Equal(t, "Marvel", issue.Vendor)

	file2, err2 := os.Open("./testdata/cb_issue_tpb.html")
	defer file2.Close()
	assert.Nil(t, err2)
	issue2, err3 := parser.Issue(file2)
	assert.Nil(t, err3)
	assert.Equal(t, issue2.Series, "X-Men: The Adventures of Cyclops and Phoenix (2014)")
	assert.Equal(t, issue2.Number, "")
	assert.Equal(t, 2014, issue2.PublicationDate.Year())
	assert.False(t, issue2.IsVariant)
	assert.False(t, issue2.IsIssue)

	file3, err3 := os.Open("./testdata/cb_issue_nn.html")
	defer file3.Close()
	assert.Nil(t, err3)
	issue3, err4 := parser.Issue(file3)
	assert.Nil(t, err4)
	assert.Equal(t, "X-Men: Empire's End (1997)", issue3.Series)
	assert.Equal(t, "", issue3.Number)
	assert.False(t, issue3.IsIssue)
	assert.Equal(t, 1998, issue3.PublicationDate.Year())
	assert.Equal(t, time.September, issue3.PublicationDate.Month())
	assert.Equal(t, 1998, issue3.OnSaleDate.Year())
	assert.Equal(t, time.July, issue3.OnSaleDate.Month())

	file4, err4 := os.Open("./testdata/cb_issue_digital.html")
	defer file4.Close()
	assert.Nil(t, err4)
	issue4, err4 := parser.Issue(file4)
	assert.Nil(t, err4)
	assert.Equal(t, "X-Men Forever (2009)", issue4.Series)
	assert.Equal(t, "", issue4.Number)
	assert.False(t, issue4.IsIssue)
	assert.Equal(t, 2009, issue4.PublicationDate.Year())
	assert.Equal(t, time.April, issue4.PublicationDate.Month())
	assert.Equal(t, 2009, issue4.OnSaleDate.Year())
	assert.Equal(t, time.February, issue4.OnSaleDate.Month())
}

func TestCbParser_Error(t *testing.T) {
	file, err := os.Open("./testdata/cb_error.html")
	if err != nil {
		t.Errorf(err.Error())
	}
	parser := CbParser{}
	issue, err := parser.Issue(file)
	assert.Nil(t, issue)
	assert.Error(t, err)

	file.Seek(0, 0)
	issueLinks, err := parser.IssueLinks(file)
	assert.Nil(t, issueLinks)
	assert.Error(t, err)

	file.Seek(0, 0)
	character, err := parser.Character(file)
	assert.Nil(t, character)
	assert.Error(t, err)

	file.Seek(0, 0)
	search, err := parser.CharacterSearch(file)
	assert.Nil(t, search)
	assert.Error(t, err)
}

func TestCbParser_Character(t *testing.T) {
	file, err := os.Open("./testdata/1.html")
	if err != nil {
		t.Errorf(err.Error())
	}
	parser := CbParser{}
	i, err := parser.Character(file)
	if err != nil {
		t.Errorf(err.Error())
	}
	assert.Equal(t, "Emmaline Frost", i.Name)
}

func TestCbParser_Issue_Mid_Date(t *testing.T) {
	file, err := os.Open("./testdata/cb_issue_mid_date.html")
	if err != nil {
		t.Errorf(err.Error())
	}
	parser := CbParser{}
	i, err := parser.Issue(file)
	if err != nil {
		t.Errorf(err.Error())
	}
	assert.True(t, i.Id != "")
	assert.Equal(t, i.PublicationDate.Year(), 1989)
	assert.Equal(t, i.OnSaleDate.Year(), 1989)
	assert.Equal(t, i.PublicationDate.Month(), time.December)
	assert.Equal(t, i.OnSaleDate.Month(), time.October)
}

func TestCbParser_Character_With_Other_Identities(t *testing.T) {
	file, err := os.Open("./testdata/cb_character_other_identities.html")
	assert.Nil(t, err)
	parser := CbParser{}
	i, err := parser.Character(file)
	assert.Nil(t, err)
	assert.Len(t, i.OtherIdentities, 3)
	assert.Equal(t, "Black Queen (Marvel)(05 - Emma Frost)", i.OtherIdentities[0].Name)
	assert.Equal(t, "http://comicbookdb.com/character.php?ID=34860", i.OtherIdentities[0].Url)
	assert.Equal(t, "Perfection (Marvel)", i.OtherIdentities[1].Name)
	assert.Equal(t, "http://comicbookdb.com/character.php?ID=28653", i.OtherIdentities[1].Url)
	assert.Equal(t, "White Queen (Marvel)(02 - Emma Frost)", i.OtherIdentities[2].Name)
	assert.Equal(t, "http://comicbookdb.com/character.php?ID=3679", i.OtherIdentities[2].Url)
}
