package twitter

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dghubble/oauth1"
	"github.com/mailru/easyjson"
	"golang.org/x/time/rate"
)

var (
	UserTimeline    = resource("GET", "https://api.twitter.com/1.1/statuses/user_timeline.json")
	DestroyTweet    = resource("POST", "https://api.twitter.com/1.1/statuses/destroy/:id.json")
	Favorites       = resource("GET", "https://api.twitter.com/1.1/favorites/list.json")
	DestroyFavorite = resource("POST", "https://api.twitter.com/1.1/favorites/destroy.json")
)

const (
	TimeLayout = "Mon Jan 2 15:04:05 -0700 2006"
)

func MustParseTime(s string) time.Time {
	t, err := time.Parse(TimeLayout, s)
	if err != nil {
		panic(err)
	}
	return t
}

type QueryOption func(url.Values)

func QueryOptions(opts ...QueryOption) []QueryOption {
	return opts
}

func WithNumber(name string, id int64) QueryOption {
	return func(q url.Values) {
		q.Set(name, strconv.FormatInt(id, 10))
	}
}

func WithOptions(opts ...QueryOption) QueryOption {
	return func(q url.Values) {
		for _, opt := range opts {
			opt(q)
		}
	}
}

type Config struct {
	Consumer       string
	ConsumerSecret string
	Token          string
	TokenSecret    string
}

type Client struct {
	Config *Config

	once sync.Once
	http *http.Client
}

func (c *Client) init() {
	c.once.Do(func() {
		c.http = oauth1.NewClient(context.Background(),
			&oauth1.Config{
				ConsumerKey:    c.Config.Consumer,
				ConsumerSecret: c.Config.ConsumerSecret,
			},
			&oauth1.Token{
				Token:       c.Config.Token,
				TokenSecret: c.Config.TokenSecret,
			},
		)
	})
}

func (c *Client) Do(ctx context.Context, res Resource, opts ...QueryOption) ([]byte, error) {
	c.init()

	req := request(res, opts...)
	//bts, _ := httputil.DumpRequest(req, true)
	//fmt.Println(string(bts))
	resp, err := c.http.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	//bts, _ := httputil.DumpResponse(resp, true)
	//fmt.Println(string(bts))
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if x := resp.StatusCode; x != http.StatusOK {
		var errResp ErrorResponse
		easyjson.Unmarshal(body, &errResp)
		limit, _ := parseRateLimit(resp.Header)
		return nil, &Error{
			StatusCode: resp.StatusCode,
			Errors:     errResp.Errors,
			RateLimit:  limit,
		}
	}

	return body, nil
}

type RateLimit struct {
	Limit     int
	Remaining int
	Reset     time.Time
}

func parseRateLimit(h http.Header) (r RateLimit, err error) {
	if s := h.Get("X-Rate-Limit-Limit"); s != "" {
		r.Limit, err = strconv.Atoi(s)
		if err != nil {
			return
		}
	}
	if s := h.Get("X-Rate-Limit-Remaining"); s != "" {
		r.Remaining, err = strconv.Atoi(s)
		if err != nil {
			return
		}
	}
	if s := h.Get("X-Rate-Limit-Reset"); s != "" {
		t, err := strconv.Atoi(s)
		if err != nil {
			return r, err
		}
		r.Reset = time.Unix(int64(t), 0)
	}
	return r, nil
}

type Error struct {
	StatusCode int
	Errors     []ErrorInfo
	RateLimit  RateLimit
}

func (err *Error) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "twitter: bad status code: %d", err.StatusCode)
	if len(err.Errors) != 0 {
		fmt.Fprintf(&b, ": ")
		for i, info := range err.Errors {
			if i > 0 {
				fmt.Fprintf(&b, "; ")
			}
			fmt.Fprintf(&b, "%s (code %d)",
				strings.Trim(info.Message, "."),
				info.Code,
			)
		}
	}
	return b.String()
}

type TweetIterator func(context.Context) (Tweet, error)

func (c *Client) ForEachTweet(res Resource, opts ...QueryOption) TweetIterator {
	var (
		tweets Tweets
		pos    int
	)
	it := Iterator{
		Client:   c,
		Resource: res,
		Options:  opts,
		OnData: func(p []byte) (id int64, err error) {
			err = easyjson.Unmarshal(p, &tweets)
			if err != nil {
				return 0, err
			}
			n := len(tweets)
			if n == 0 {
				return 0, io.EOF
			}
			return tweets[n-1].ID, nil
		},
	}
	return func(ctx context.Context) (Tweet, error) {
		if pos >= len(tweets) {
			if !it.Next(ctx) {
				return Tweet{}, it.Err()
			}
			pos = 0
		}
		ret := tweets[pos]
		pos++
		return ret, nil
	}
}

func request(r Resource, opts ...QueryOption) *http.Request {
	if len(opts) > 0 {
		v := make(url.Values, len(opts))
		for _, opt := range opts {
			opt(v)
		}
		var e error
		r, e = r.Render(func(name string) string {
			ret := v.Get(name)
			v.Del(name)
			return ret
		})
		if e != nil {
			panic(fmt.Sprintf(
				"render resource %q error: %v",
				r.url.Path, e,
			))
		}
		for name, values := range r.url.Query() {
			if _, has := v[name]; has {
				panic(fmt.Sprintf(
					"options provided option %q which is already exist",
					name,
				))
			}
			for _, val := range values {
				v.Add(name, val)
			}
		}
		r.url.RawQuery = v.Encode()
	}
	s := r.url.String()
	req, err := http.NewRequest(r.method, s, nil)
	if err != nil {
		panic(err)
	}
	return req
}

type Resource struct {
	method string
	url    url.URL
}

func (r Resource) Render(lookup func(string) string) (res Resource, err error) {
	var (
		s          = r.url.Path
		withinName = -1
		offset     = 0

		b strings.Builder
	)
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case 'a' <= c && c <= 'z':
			continue

		case c == ':':
			withinName = i + 1
			b.WriteString(s[offset:i])

		default:
			if withinName == -1 {
				continue
			}
			var (
				name  = s[withinName:i]
				value = lookup(name)
			)
			if value == "" {
				err = fmt.Errorf("no parameter %q provided", name)
				return
			}
			b.WriteString(value)
			withinName = -1
			offset = i
		}
	}
	res = r
	if b.Len() > 0 {
		b.WriteString(s[offset:])
		res.url.Path = b.String()
	}

	return res, nil
}

func resource(method, path string, params ...string) Resource {
	u, err := url.Parse(path)
	if err != nil {
		panic(err)
	}
	u.User = nil
	query := u.Query()
	for i := 0; i < len(params); i += 2 {
		key := params[i+0]
		val := params[i+1]
		query.Add(key, val)
	}
	u.RawQuery = query.Encode()
	return Resource{
		method: method,
		url:    *u,
	}
}

type Iterator struct {
	Client   *Client
	Resource Resource
	Options  []QueryOption
	OnData   func([]byte) (id int64, err error)

	err     error
	limiter *rate.Limiter
	offset  int64
}

func (it *Iterator) init() {
	if it.limiter != nil {
		return
	}
	it.limiter = rate.NewLimiter(1, 1)
}

func (it *Iterator) Next(ctx context.Context) bool {
	if it.err != nil {
		return false
	}

	it.init()

	id, err := it.fetch(ctx)
	if err != nil {
		it.err = err
		return false
	}
	it.offset = id - 1

	return true
}

func (it *Iterator) Err() error {
	return it.err
}

func (it *Iterator) fetch(ctx context.Context) (int64, error) {
	for {
		if err := it.limiter.Wait(ctx); err != nil {
			return 0, err
		}
		var opts []QueryOption
		if it.offset > 0 {
			opts = append(opts, WithNumber("max_id", it.offset))
		}
		bts, err := it.Client.Do(ctx, it.Resource,
			WithOptions(it.Options...),
			WithOptions(opts...),
		)
		if limit, ok := IsRateLimited(err); ok {
			if err = Relax(ctx, limit); err != nil {
				return 0, err
			}
			continue
		}
		if err != nil {
			return 0, err
		}
		return it.OnData(bts)
	}
}

func IsRateLimited(err error) (RateLimit, bool) {
	terr, ok := err.(*Error)
	if ok && terr.StatusCode == http.StatusTooManyRequests {
		return terr.RateLimit, true
	}
	return RateLimit{}, false
}

func Relax(ctx context.Context, limit RateLimit) error {
	sleep := time.Second
	if t := limit.Reset; !t.IsZero() {
		sleep = time.Until(t)
	}
	select {
	case <-time.After(sleep):
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}
