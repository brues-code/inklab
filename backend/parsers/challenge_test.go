package parsers

import "testing"

// blazingFastPage is the interstitial octowow.st serves behind BlazingFast,
// trimmed to the parts that identify it. It arrives with HTTP 200, which is what
// made it indistinguishable from a real page.
const blazingFastPage = `<!DOCTYPE HTML>
<html lang="en-US">
<head>
	<title>Just a moment please...</title>
	<meta name="robots" content="noindex, nofollow" />
	<script src="/bf.jquery.max.js"></script>
</head>
<body>
	<form id="bf-form" action="/blzgfst-shark/" method="get"></form>
</body>
</html>`

// challengeRealPage stands in for an aowow object page: the spawn data lives in
// a myMapper.update call.
const challengeRealPage = `<html><head><title>Copper Vein - Object</title></head><body>
<a href="#" onclick="myMapper.update({zone: 1519,coords: [[39.95,84.36,0]]})">Stormwind City</a>
</body></html>`

func TestIsChallengePage(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool
	}{
		{"blazingfast interstitial", blazingFastPage, true},
		{"mixed-case title", "<title>JUST A MOMENT PLEASE...</title>", true},
		// The variant that reaches the parsers as a page title.
		{"verifying variant", "Verifying your browser, please wait...DDoS Protection by Blazingfast.io", true},
		{"cloudflare style", "<h1>Checking your browser before accessing</h1>", true},
		{"real object page", challengeRealPage, false},
		{"empty body", "", false},
		// The proxy stamps its name on real pages too, so that alone must never
		// be enough to reject a response.
		{"real page mentioning the proxy", "<html>Served by BlazingFastWeb</html>", false},
	}
	for _, c := range cases {
		if got := IsChallengePage([]byte(c.body)); got != c.want {
			t.Errorf("%s: IsChallengePage = %v, want %v", c.name, got, c.want)
		}
	}
}
