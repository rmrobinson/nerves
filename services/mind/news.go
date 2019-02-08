package mind

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/rmrobinson/nerves/services/news"
	"go.uber.org/zap"
)

const (
	newsRegex = "what('| i)?s (the news|happening|going on( in the world)?)"
)

// News is a news-request handler
type News struct {
	logger *zap.Logger

	client news.NewsServiceClient
}

// NewNews creates a new news handler
func NewNews(logger *zap.Logger, client news.NewsServiceClient) *News {
	return &News{
		logger: logger,
		client: client,
	}
}

// ProcessStatement implements the handler interface. Logs and returns the statement.
func (n *News) ProcessStatement(ctx context.Context, stmt *Statement) (*Statement, error) {
	if stmt.MimeType != "text/plain" {
		return nil, ErrStatementNotHandled.Err()
	}

	content := string(stmt.Content)
	content = strings.ToLower(content)
	if ok, _ := regexp.MatchString(newsRegex, content); !ok {
		return nil, ErrStatementNotHandled.Err()
	}

	return n.getNews(), nil
}

func (n *News) getNews() *Statement {
	resp, err := n.client.ListArticles(context.Background(), &news.ListArticlesRequest{})
	if err != nil {
		n.logger.Warn("unable to get news articles",
			zap.Error(err),
		)

		return statementFromText("Can't get the news right now :(")
	} else if resp == nil {
		n.logger.Warn("unable to get news (empty response)")

		return statementFromText("Not sure what the news is right now")
	}

	return statementFromArticles(resp.Articles)
}

func statementFromArticles(articles []*news.Article) *Statement {
	newsText := "Here's the news!\n"
	for idx, record := range articles {
		if idx > 0 {
			newsText += "\n"
		}
		newsText += fmt.Sprintf("*%s*\n```%s\n```%s\n", record.Title, record.Description, record.Link)
		if idx > 10 {
			break
		}
	}

	return statementFromText(newsText)
}
