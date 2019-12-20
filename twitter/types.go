package twitter

//go:generate easyjson -all -snake_case

//easyjson:json
type Tweets []Tweet

type ErrorResponse struct {
	Errors []ErrorInfo
}

type ErrorInfo struct {
	Code    int
	Message string
}

type Tweet struct {
	CreatedAt string
	ID        int64
	IDStr     string
	Text      string
	Truncated bool
	Source    string
	User      User
	Lang      string
	Retweeted bool

	InReplyToStatusID   int64
	InReplyToUserID     int64
	InReplyToScreenName string
}

type User struct {
	ID          int
	IDStr       string
	Name        string
	ScreenName  string
	Location    string
	Description string
	URL         string
}
