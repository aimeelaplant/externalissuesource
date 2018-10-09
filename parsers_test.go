package externalissuesource

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
	"unicode/utf8"
	"sync"
	"fmt"
)

func TestCbParser_parse(t *testing.T) {
	file, err := os.Open("./testdata/cb_character.html")
	defer file.Close()
	if err != nil {
		t.Errorf(err.Error())
	}
	parser := CbParser{}
	character, err := parser.Character(file)
	if err != nil {
		t.Errorf(err.Error())
	}
	assert.Len(t, character.IssueLinks, 10)
	assert.Equal(t, "Jean Grey", character.OtherName)
}

func TestCbParser_Issue_TheEnd(t *testing.T) {
	file, err := os.Open("./testdata/cb_issue_end.html")
	defer file.Close()
	if err != nil {
		t.Errorf(err.Error())
	}
	parser := CbParser{}
	issue, err := parser.Issue(file)
	assert.Nil(t, err)
	assert.Equal(t, "1", issue.Number)
	assert.Equal(t, "26025", issue.Id)
	assert.Equal(t, "3152", issue.SeriesId)
	assert.Equal(t, "X-Men: The End: Book 1: Dreamers & Demons (2004)", issue.Series)
	assert.Equal(t, "Marvel", issue.Vendor)
	assert.Equal(t, 2004, int(issue.OnSaleDate.Year()))
	assert.Equal(t, time.Month(8), issue.OnSaleDate.Month())
	assert.Equal(t, 2004, int(issue.PublicationDate.Year()))
	assert.Equal(t, 10, int(issue.PublicationDate.Month()))
	assert.Equal(t, Standard, issue.Format)
}

func TestCbParser_parseIssueLinks(t *testing.T) {
	file, err := os.Open("./testdata/cb_character.html")
	defer file.Close()
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
	assert.False(t, issue.IsReprint)
	assert.Equal(t, 2007, int(issue.PublicationDate.Year()))
	assert.Equal(t, 10, int(issue.PublicationDate.Month()))
	assert.Equal(t, 2007, int(issue.OnSaleDate.Year()))
	assert.Equal(t, 8, int(issue.OnSaleDate.Month()))
	assert.Equal(t, "Astonishing X-Men (2004)", issue.Series)
	assert.Equal(t, "22", issue.Number)
	assert.Equal(t, "Marvel", issue.Vendor)
	assert.Equal(t, Standard, issue.Format)

	file2, err2 := os.Open("./testdata/cb_issue_tpb.html")
	defer file2.Close()
	assert.Nil(t, err2)
	issue2, err3 := parser.Issue(file2)
	assert.Nil(t, err3)
	assert.Equal(t, issue2.Series, "X-Men: The Adventures of Cyclops and Phoenix (2014)")
	assert.Empty(t, issue2.Number)
	assert.Equal(t, 2014, issue2.PublicationDate.Year())
	assert.False(t, issue2.IsVariant)
	assert.False(t, issue2.IsReprint)
	assert.Equal(t, TPB, issue2.Format)

	file3, err3 := os.Open("./testdata/cb_issue_nn.html")
	defer file3.Close()
	assert.Nil(t, err3)
	issue3, err4 := parser.Issue(file3)
	assert.Nil(t, err4)
	assert.Equal(t, "X-Men: Empire's End (1997)", issue3.Series)
	assert.Equal(t, "", issue3.Number)
	assert.Equal(t, 1998, issue3.PublicationDate.Year())
	assert.Equal(t, time.September, issue3.PublicationDate.Month())
	assert.Equal(t, 1998, issue3.OnSaleDate.Year())
	assert.Equal(t, time.July, issue3.OnSaleDate.Month())
	assert.Equal(t, Bookshelf, issue3.Format)

	file4, err4 := os.Open("./testdata/cb_issue_digital.html")
	defer file4.Close()
	assert.Nil(t, err4)
	issue4, err4 := parser.Issue(file4)
	assert.Nil(t, err4)
	assert.Equal(t, "X-Men Forever (2009)", issue4.Series)
	assert.Equal(t, "", issue4.Number)
	assert.Equal(t, 2009, issue4.PublicationDate.Year())
	assert.Equal(t, time.April, issue4.PublicationDate.Month())
	assert.Equal(t, 2009, issue4.OnSaleDate.Year())
	assert.Equal(t, time.February, issue4.OnSaleDate.Month())
	assert.Equal(t, DigitalMedia, issue4.Format)
}

func TestCbParser_Error(t *testing.T) {
	file, err := os.Open("./testdata/cb_error.html")
	defer file.Close()
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
	defer file.Close()
	if err != nil {
		t.Errorf(err.Error())
	}
	parser := CbParser{}
	i, err := parser.Character(file)
	if err != nil {
		t.Errorf(err.Error())
	}
	assert.Equal(t, "Emmaline Frost", i.Name)
	assert.Equal(t, "", i.OtherName)
}

func TestCbParser_Issue_Mid_Date(t *testing.T) {
	file, err := os.Open("./testdata/cb_issue_mid_date.html")
	defer file.Close()
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
	assert.Equal(t, Standard, i.Format)
}

func TestCbParser_Character_With_Other_Identities(t *testing.T) {
	file, err := os.Open("./testdata/cb_character_other_identities.html")
	defer file.Close()
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

func TestCbParser_Issue_Foreign(t *testing.T) {
	file, err := os.Open("./testdata/cb_issue_deutschland.html")
	defer file.Close()
	if err != nil {
		t.Errorf(err.Error())
	}
	parser := CbParser{}
	issue, err := parser.Issue(file)
	assert.Nil(t, err)
	assert.Equal(t, "6", issue.Number)
	assert.Equal(t, "352823", issue.Id)
	assert.Equal(t, "50014", issue.SeriesId)
	assert.Equal(t, "Action Comics (2001)", issue.Series)
	assert.Equal(t, "Panini Verlags GmbH", issue.Vendor)
	assert.Equal(t, 2002, int(issue.OnSaleDate.Year()))
	assert.Equal(t, time.February, issue.OnSaleDate.Month())
	assert.Equal(t, 2002, int(issue.PublicationDate.Year()))
	assert.Equal(t, time.April, issue.PublicationDate.Month())
	assert.Equal(t, Standard, issue.Format)
}

func TestCbParser_Issue_Dec_Jan(t *testing.T) {
	file, err := os.Open("./testdata/cb_issue_dec_jan.html")
	defer file.Close()
	if err != nil {
		t.Errorf(err.Error())
	}
	parser := CbParser{}
	issue, err := parser.Issue(file)
	assert.Nil(t, err)
	assert.False(t, issue.MonthUncertain)
	assert.Equal(t, 1971, issue.OnSaleDate.Year())
	assert.Equal(t, time.October, issue.OnSaleDate.Month())
	assert.Equal(t, 1972, issue.PublicationDate.Year())
	assert.Equal(t, time.January, issue.PublicationDate.Month())
	assert.Equal(t, Standard, issue.Format)
}

func TestCbParser_Issue_Year_Only(t *testing.T) {
	file, err := os.Open("./testdata/cb_issue_year_only.html")
	defer file.Close()
	if err != nil {
		t.Errorf(err.Error())
	}
	parser := CbParser{}
	issue, err := parser.Issue(file)
	assert.Nil(t, err)
	assert.True(t, issue.MonthUncertain)
	assert.Equal(t, 2014, issue.OnSaleDate.Year())
	assert.Equal(t, 2014, issue.PublicationDate.Year())
	assert.Equal(t, time.January, issue.PublicationDate.Month())
	assert.Equal(t, time.January, issue.OnSaleDate.Month())
	assert.Equal(t, OGN, issue.Format)
}

func TestCbParser_Issue_Jul_Sep(t *testing.T) {
	file, err := os.Open("./testdata/cb_issue_jul_sep.html")
	defer file.Close()
	if err != nil {
		t.Errorf(err.Error())
	}
	parser := CbParser{}
	issue, err := parser.Issue(file)
	assert.Nil(t, err)
	assert.False(t, issue.MonthUncertain)
	assert.Equal(t, 1971, issue.OnSaleDate.Year())
	assert.Equal(t, time.June, issue.OnSaleDate.Month())
	assert.Equal(t, 1971, issue.PublicationDate.Year())
	assert.Equal(t, time.September, issue.PublicationDate.Month())
	assert.Equal(t, Standard, issue.Format)
}

func TestCbParser_Issue_ForeignChars(t *testing.T) {
	file, err := os.Open("./testdata/cb_issue_windows_1252.html")
	defer file.Close()
	if err != nil {
		t.Errorf(err.Error())
	}
	parser := CbParser{}
	issue, err := parser.Issue(file)
	assert.Nil(t, err)
	assert.True(t, utf8.Valid([]byte(issue.Series)))
	assert.Equal(t, Standard, issue.Format)
}

func TestCbParser_Issue_Hc(t *testing.T) {
	file, err := os.Open("./testdata/cb_issue_hc.html")
	defer file.Close()
	if err != nil {
		t.Errorf(err.Error())
	}
	parser := CbParser{}
	issue, err := parser.Issue(file)
	assert.Nil(t, err)
	assert.Equal(t, HC, issue.Format)
}

func TestCbParser_Character_Link_In_Bio(t *testing.T) {
	file, err := os.Open("./testdata/cb_character_link_in_bio.html")
	defer file.Close()
	if err != nil {
		t.Errorf(err.Error())
	}
	parser := CbParser{}
	c, err := parser.Character(file)
	assert.Nil(t, err)
	assert.Len(t, c.IssueLinks, 168)
	assert.Equal(t, "Jean Karen Grant Grey", c.OtherName)
}

func TestCbParser_CharacterSearch(t *testing.T) {
	file, err := os.Open("./testdata/cyclops/search.html")
	defer file.Close()
	if err != nil {
		t.Errorf(err.Error())
	}
	parser := CbParser{}
	c, err := parser.CharacterSearch(file)
	assert.Nil(t, err)
	assert.Len(t, c.Results, 46)
}

func TestCbParser_Issue_No_Edit(t *testing.T) {
	file, err := os.Open("./testdata/cb_issue_no_edit.html")
	defer file.Close()
	if err != nil {
		t.Errorf(err.Error())
	}
	parser := CbParser{}
	issue, err := parser.Issue(file)
	assert.Nil(t, err)
	assert.Equal(t, "39067", issue.Id)
	assert.Equal(t, Standard, issue.Format)
}

func TestCbParser_Issue_Annual(t *testing.T) {
	file, err := os.Open("./testdata/cb_issue_annual.html")
	defer file.Close()
	if err != nil {
		t.Errorf(err.Error())
	}
	parser := CbParser{}
	issue, err := parser.Issue(file)
	assert.Nil(t, err)
	assert.Equal(t, "28458", issue.Id)
	assert.Equal(t, Standard, issue.Format)
	assert.Equal(t, "Annual '96", issue.Number)
}


func TestConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	//parser := CbParser{}
	parser := CbParser{}
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func(wg *sync.WaitGroup, i int) {
			fmt.Println(i)
			defer wg.Done()
			file, err := os.Open("./testdata/cb_issue_annual.html")
			defer file.Close()
			issue, err := parser.Issue(file)
			assert.Nil(t, err)
			fmt.Println(issue.Id)

		}(&wg, i)
	}
	wg.Wait()
	// done
}

func TestCbParser_Issue_Reprint(t *testing.T) {
	file, err := os.Open("./testdata/cb_issue_reprint.html")
	defer file.Close()
	if err != nil {
		t.Errorf(err.Error())
	}
	parser := CbParser{}
	issue, err := parser.Issue(file)
	assert.Nil(t, err)
	assert.True(t, issue.IsReprint)
}

func TestCbParser_Issue_No_Reprint(t *testing.T) {
	file, err := os.Open("./testdata/cb_issue_no_reprint.html")
	defer file.Close()
	if err != nil {
		t.Errorf(err.Error())
	}
	parser := CbParser{}
	issue, err := parser.Issue(file)
	assert.Nil(t, err)
	assert.False(t, issue.IsReprint)
}
