package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gobwas/flagutil"
	"github.com/gobwas/flagutil/parse/env"
	"github.com/gobwas/flagutil/parse/file"
	"github.com/gobwas/flagutil/parse/file/json"
	"github.com/gobwas/flagutil/parse/pargs"
	"github.com/gobwas/rmtweet"
	"github.com/gobwas/rmtweet/twitter"
	"github.com/gobwas/rmtweet/twitter/twittercfg"
)

func main() {
	log.SetFlags(0)

	flags := flag.NewFlagSet("", flag.ExitOnError)
	flags.String("c", "", "path to the configuration file")

	var config *twitter.Config
	flagutil.Subset(flags, "twitter", func(flags *flag.FlagSet) {
		config = twittercfg.Export(flags)
	})
	var (
		criteria  = rmtweet.ExportCriteria(flags)
		resources = rmtweet.ExportResource(flags)
	)
	var (
		force   bool
		verbose bool
	)
	flags.BoolVar(&force,
		"force", false,
		"do not ask before destroying object",
	)
	flags.BoolVar(&verbose,
		"verbose", false,
		"be verbose",
	)

	flagutil.Parse(flags,
		flagutil.WithParser(&pargs.Parser{
			Args: os.Args[1:],
		}),
		flagutil.WithParser(&env.Parser{
			Prefix:       "RMTWEET_",
			SetSeparator: "_",
		}),
		flagutil.WithParser(&file.Parser{
			PathFlag: "c",
			Syntax:   &json.Syntax{},
		}),
	)

	get, del, err := resources()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan os.Signal, 1)
	go func() {
		var n int
		for sig := range ch {
			fmt.Println("caught signal", sig)
			if n > 0 {
				os.Exit(2)
			}
			n++
			cancel()
		}
	}()
	signal.Notify(ch, syscall.SIGINT)

	d := rmtweet.Destroyer{
		Client: twitter.Client{
			Config: config,
		},
		Get:      get,
		Del:      del,
		Criteria: criteria,
		Prompt: func(t twitter.Tweet) bool {
			if force {
				return true
			}
			return ask("destroy this?\n  %s\n", line(t))
		},
		Destroyed: func(t twitter.Tweet) {
			if !verbose || !force {
				return
			}
			fmt.Fprintf(os.Stderr, "destroyed\n  %s\n", line(t))
		},
	}
	if err := d.Destroy(ctx); err != nil {
		log.Fatal(err)
	}
}

func line(t twitter.Tweet) string {
	n := len(t.Text)
	if n > 100 {
		n = 100
	}
	i := strings.IndexByte(t.Text[:n], '\n')
	if i != -1 {
		n = i
	}
	if n < len(t.Text) {
		t.Text = t.Text[:n] + "..."
	}
	return fmt.Sprintf(
		"%s: %s%s: %s",
		twitter.MustParseTime(t.CreatedAt).Format("02.01.2006"),
		t.User.ScreenName, ifstr(t.InReplyToScreenName != "", fmt.Sprintf(
			" to @%s", t.InReplyToScreenName,
		)),
		t.Text,
	)
}

func ifstr(cond bool, str string) string {
	if cond {
		return str
	}
	return ""
}

func ask(f string, args ...interface{}) bool {
	fmt.Fprintf(os.Stderr, f, args...)

	r := bufio.NewReaderSize(os.Stdin, 32)
	line, err := r.ReadBytes('\n')
	if err != nil {
		panic(err)
	}
	return bytes.Contains(line, []byte("y"))
}
