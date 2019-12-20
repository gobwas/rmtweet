package rmtweet

import (
	"flag"
	"time"

	"github.com/gobwas/flagvar"
	"github.com/gobwas/rmtweet/twitter"
)

type Criteria struct {
	MaxDate     time.Time
	MinDate     time.Time
	RepliesOnly bool
	RepliesTo   string
}

var zeroCriteria Criteria

func (c *Criteria) Empty() bool {
	return *c == zeroCriteria
}

func (c *Criteria) Match(t twitter.Tweet) bool {
	if c.Empty() {
		return true
	}
	if name := c.RepliesTo; name != "" && t.InReplyToScreenName != name {
		return false
	}
	if c.RepliesOnly && t.InReplyToStatusID == 0 {
		return false
	}

	date := twitter.MustParseTime(t.CreatedAt)
	if !c.MinDate.IsZero() && date.Before(c.MinDate) {
		return false
	}
	if !c.MaxDate.IsZero() && date.After(c.MaxDate) {
		return false
	}

	return true
}

const ddmmyyyy = "02.01.2006"

func ExportCriteria(flag *flag.FlagSet) *Criteria {
	var c Criteria
	flag.Var(flagvar.Time(&c.MinDate, ddmmyyyy),
		"min-date", "date to filter out tweets in form `DD.MM.YYYY`",
	)
	flag.Var(flagvar.Time(&c.MaxDate, ddmmyyyy),
		"max-date", "date to filter out tweets in form `DD.MM.YYYY`",
	)
	flag.BoolVar(&c.RepliesOnly,
		"replies", false,
		"filter out only replies",
	)
	flag.StringVar(&c.RepliesTo,
		"replies-to", "",
		"filter out only replies to this user",
	)
	return &c
}
