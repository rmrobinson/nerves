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

type SlackBot struct {
	logger *zap.Logger
	s *Service
	api *slack.Client
}

func NewSlackBot(logger *zap.Logger, s *Service, api *slack.Client) *SlackBot {
	return &SlackBot{
		logger: logger,
		s: s,
		api: api,
	}
}

func (sb *SlackBot) Run(channelID string) {
	rtm := sb.api.NewRTM()
	go rtm.ManageConnection()

	var teamID string

	for msg := range rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.ConnectedEvent:
			sb.logger.Debug("connected to slack",
				zap.String("team_id", ev.Info.Team.ID),
				zap.String("team_name", ev.Info.Team.Name),
				zap.Int("connection_count", ev.ConnectionCount),
			)

			teamID = ev.Info.Team.ID

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

			_, err = sb.s.SendStatement(context.Background(), req)
			if err != nil {
				sb.logger.Info("error sending statement",
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

		case *slack.InvalidAuthEvent:
			sb.logger.Warn("credentials invalid")
			return
		}
	}
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
