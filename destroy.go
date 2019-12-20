package rmtweet

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/gobwas/rmtweet/twitter"
)

type Destroyer struct {
	Client    twitter.Client
	Get       twitter.Resource
	Del       twitter.Resource
	Criteria  *Criteria
	Prompt    func(twitter.Tweet) bool
	Destroyed func(twitter.Tweet)
}

func (d *Destroyer) Destroy(ctx context.Context) error {
	next := d.Client.ForEachTweet(d.Get,
		twitter.WithNumber("count", 200),
	)
	for {
		t, err := next(ctx)
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return err
		}
		if !d.Criteria.Match(t) {
			continue
		}
		if ask := d.Prompt; ask != nil && !ask(t) {
			continue
		}
		for {
			_, err := d.Client.Do(ctx, d.Del,
				twitter.WithNumber("id", t.ID),
			)
			limit, limited := twitter.IsRateLimited(err)
			if limited {
				err = twitter.Relax(ctx, limit)
			}
			if err == nil {
				break
			}
			return err
		}
		if fn := d.Destroyed; fn != nil {
			fn(t)
		}
	}
	return nil
}

func ExportResource(flag *flag.FlagSet) func() (get, del twitter.Resource, err error) {
	var (
		tweets    bool
		favorites bool
	)
	flag.BoolVar(&tweets,
		"tweets", false,
		"work with tweets",
	)
	flag.BoolVar(&favorites,
		"favorites", false,
		"work with favorites",
	)
	return func() (get, del twitter.Resource, err error) {
		switch {
		case tweets:
			get = twitter.UserTimeline
			del = twitter.DestroyTweet
			return

		case favorites:
			get = twitter.Favorites
			del = twitter.DestroyFavorite
			return

		default:
			err = fmt.Errorf(
				"need resource to work with; consider tweets or favorites",
			)
			return
		}
	}
}
