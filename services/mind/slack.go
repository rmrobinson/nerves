package mind

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/google/uuid"
	"github.com/nlopes/slack"
	"go.uber.org/zap"
)

// SlackBot is a message service implementation.
type SlackBot struct {
	logger *zap.Logger
	s *Service
	api *slack.Client
}

// NewSlackBot creates a new slackbot using the supplied slack implementatino.
func NewSlackBot(logger *zap.Logger, s *Service, api *slack.Client) *SlackBot {
	return &SlackBot{
		logger: logger,
		s: s,
		api: api,
	}
}

// Run begins the event loop and posts messages to the specified channel.
func (sb *SlackBot) Run(channelID string) {
	rtm := sb.api.NewRTM()
	go rtm.ManageConnection()

	var teamID string

	for msg := range rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.InvalidAuthEvent:
			sb.logger.Warn("credentials invalid")
			return

		case *slack.ConnectedEvent:
			sb.logger.Debug("connected to slack",
				zap.String("team_id", ev.Info.Team.ID),
				zap.String("team_name", ev.Info.Team.Name),
				zap.Int("connection_count", ev.ConnectionCount),
			)

			teamID = ev.Info.Team.ID

			channels, err := sb.api.GetChannels(false)
			if err != nil {
				sb.logger.Warn("error getting channels",
					zap.Error(err),
				)
				continue
			}

			for _, channel := range channels {
				sb.logger.Debug("channel",
					zap.String("channel_name", channel.Name),
					zap.String("channel_id", channel.ID),
				)
			}

			rtm.SendMessage(rtm.NewOutgoingMessage("I'm alive!", channelID))

		case *slack.MessageEvent:
			ts, err := parseUnixTime(ev.Timestamp)
			if err != nil {
				sb.logger.Info("error parsing timestamp for message",
					zap.String("user_name", ev.User),
					zap.String("message", ev.Text),
				)
				continue
			}
			createAt, err := ptypes.TimestampProto(ts)
			if err != nil {
				sb.logger.Info("error parsing proto timestamp for message",
					zap.String("user_name", ev.User),
					zap.String("message", ev.Text),
				)
				continue
			}

			req := &SendStatementRequest{
				Name: "/messages/" + teamID + "/" + ev.User,
				RequestId: uuid.New().String(),
				Statement: &Statement{
					CreateAt: createAt,
					LanguageCode: "en-US",
					MimeType: "text/plain",
					Content: []byte(ev.Text),
				},
			}

			reply, err := sb.s.SendStatement(context.Background(), req)
			if err != nil {
				sb.logger.Info("error sending statement",
					zap.Error(err),
				)
			}

			err = sb.replyMessage(ev.User, reply)
			if err != nil {
				sb.logger.Info("error replying to statement",
					zap.Error(err),
				)
			}

		case *slack.PresenceChangeEvent:
			sb.logger.Debug("presence changed",
				zap.String("username", ev.User),
			)

		case *slack.LatencyReport:
			sb.logger.Debug("latency report",
				zap.Float64("latency_secs", ev.Value.Seconds()),
			)

		case *slack.RTMError:
			sb.logger.Info("rtm error",
				zap.Error(ev),
			)
		}
	}
}

func (sb *SlackBot) replyMessage(userID string, statement *Statement) error {
	if len(userID) < 1 {
		return nil
	}

	_, _, channelID, err := sb.api.OpenIMChannel(userID)
	if err != nil {
		sb.logger.Debug("error creating IM channel",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return err
	}

	_, _, err = sb.api.PostMessage(channelID, slack.MsgOptionText(string(statement.Content), false))
	return err
}

func parseUnixTime(ts string) (time.Time, error) {
	parts := strings.Split(ts, ".")
	sec, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Now(), err
	}
	nsec, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return time.Now(), err
	}

	return time.Unix(sec, nsec), nil
}
