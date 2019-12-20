package twitter

import "testing"

func TestResourceRender(t *testing.T) {
	for _, test := range []struct {
		name    string
		res     Resource
		expPath string
		params  map[string]string
	}{
		{
			name:    "no params",
			res:     resource("GET", "https://twitter.com/some/path/here.json"),
			expPath: "/some/path/here.json",
		},
		{
			name: "id param",
			res:  resource("GET", "https://twitter.com/some/path/to/:id.json"),
			params: map[string]string{
				"id": "foo",
			},
			expPath: "/some/path/to/foo.json",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			res, err := test.res.Render(func(name string) string {
				return test.params[name]
			})
			if err != nil {
				t.Fatal(err)
			}
			if act, exp := res.url.Path, test.expPath; act != exp {
				t.Fatalf("unexpected url path:\nact: %q\nexp: %q", act, exp)
			}
		})
	}
}
