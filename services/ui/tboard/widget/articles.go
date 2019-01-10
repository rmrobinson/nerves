package widget

import (
	"strings"

	"github.com/rivo/tview"
)

// ArticleInfo is a data structure displayed by the article widget.
// TODO: replace with a proto defined object.
type ArticleInfo struct {
	Title  string
	Body   string
	Link   string
	Source string
}

// Articles is a widget to display a read-only list of articles with their details available.
type Articles struct {
	*tview.Flex

	app  *tview.Application
	next tview.Primitive

	articleList   *tview.List
	articleDetail *ArticleDetail

	articles []*ArticleInfo
}

// NewArticles creates a new instance of this widget with the supplied set of articles for viewing.
// It should be at least 50 characters wide for best performance.
func NewArticles(app *tview.Application, articles []*ArticleInfo) *Articles {
	a := &Articles{
		Flex:     tview.NewFlex(),
		app:      app,
		articles: articles,
	}

	a.articleList = tview.NewList().
		SetChangedFunc(a.onListEntrySelected).
		SetSelectedFunc(a.onListEntryEntered).
		SetDoneFunc(a.onListDone)

	a.articleDetail = NewArticleDetail(app, a.articleList)

	a.SetBorder(true).
		SetTitle("Articles").
		SetTitleAlign(tview.AlignLeft)

	a.SetDirection(tview.FlexRow).
		AddItem(a.articleList, 0, 1, true).
		AddItem(a.articleDetail, 0, 1, true)

	for _, article := range a.articles {
		titleWords := strings.Fields(article.Title)
		titleSize := 0
		title := ""
		body := ""
		for _, titleWord := range titleWords {
			if titleSize + 1 + len(titleWord) > 48 {
				body += " " + titleWord
			} else {
				title += titleWord + " "
				titleSize += len(titleWord) + 1
			}
		}
		a.articleList.AddItem(title, body, 0, nil)
	}

	return a
}

func (a *Articles) onListEntrySelected(idx int, mainText string, secondaryText string, shortcut rune) {
	a.articleDetail.Refresh(a.articles[idx])
}

func (a *Articles) onListEntryEntered(idx int, mainText string, secondaryText string, shortcut rune) {
}

func (a *Articles) onListDone() {
	if a.next != nil {
		a.app.SetFocus(a.next)
	}
}

// SetNextWidget controls where the focus is given should this list be left.
func (a *Articles) SetNextWidget(next tview.Primitive) {
	a.next = next
}
