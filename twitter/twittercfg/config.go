package twittercfg

import (
	"flag"

	"github.com/gobwas/rmtweet/twitter"
)

func Export(flag *flag.FlagSet) *twitter.Config {
	var c twitter.Config
	flag.StringVar(&c.Consumer,
		"consumer", "",
		"twitter consumer key",
	)
	flag.StringVar(&c.ConsumerSecret,
		"consumer-secret", "",
		"twitter consumer secret",
	)
	flag.StringVar(&c.Token,
		"token", "",
		"twitter access token",
	)
	flag.StringVar(&c.TokenSecret,
		"token-secret", "",
		"twitter access token secret",
	)
	return &c
}
