# rmtweet

> A program to manipulate your tweets and likes.

# Usage

To use this tool you should register your own application [here][twitter].

After that you need to obtain authorization parameters from
[here][twitter-apps] (select your application by clicking "details", then go to
the "keys and tokens" tab).

Now you are ready to remove some tweets or likes:

```bash
$ export RMTWEET_TWITTER_CONSUMER=your consumer id here
$ export RMTWEET_TWITTER_CONSUMER_SECRET=...
$ export RMTWEET_TWITTER_TOKEN=...
$ export RMTWEET_TWITTER_TOKEN_SECRET=...
$
$ rmtweet --favorites --max-date 05.12.2019
```

[twitter]:      https://developer.twitter.com/
[twitter-apps]: https://developer.twitter.com/en/apps/
