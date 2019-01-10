package widget

import (
	"github.com/rivo/tview"
)

// ArticleDetail is a widget that provides for viewing article details.
type ArticleDetail struct {
	*tview.Flex

	app    *tview.Application
	parent tview.Primitive

	bodyText *tview.TextView
	urlText  *tview.TextView

	article *ArticleInfo
}

// NewArticleDetail creates a new instance of the ArticleDetail view.
// Nothing will be displayed until an article is set on this view using Refresh()
func NewArticleDetail(app *tview.Application, parent tview.Primitive) *ArticleDetail {
	// Create the view
	dd := &ArticleDetail{
		Flex:     tview.NewFlex(),
		app:      app,
		parent:   parent,
		bodyText: tview.NewTextView(),
		urlText:  tview.NewTextView(),
	}

	dd.bodyText.
		SetTextAlign(tview.AlignLeft).
		SetTitle("Article").
		SetBorder(true)

	dd.urlText.
		SetTitleAlign(tview.AlignLeft).
		SetTitle("URL").
		SetBorder(true)

	// Set the layout of the parent and return.
	dd.SetDirection(tview.FlexRow).
		AddItem(dd.bodyText, 0, 5, false).
		AddItem(dd.urlText, 0, 1, false)

	return dd
}

// Refresh updates the content being displayed by this widget.
func (a *ArticleDetail) Refresh(article *ArticleInfo) {
	a.app.QueueUpdateDraw(func() {
		a.article = article

		a.SetTitle(a.article.Title)
		a.bodyText.SetText(a.article.Body)
		a.urlText.SetText(a.article.Link)

	})
}
