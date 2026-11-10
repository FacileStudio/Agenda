package env

import "testing"

func TestAntenneFromEnv(t *testing.T) {
	cases := map[string]struct {
		url, legacy, secret string
		wantOn, wantErr     bool
	}{
		"nothing set turns the pool off":      {},
		"a url and a secret turn it on":       {url: "https://antenne.example/", secret: "s", wantOn: true},
		"the older instance name still works": {legacy: "https://antenne.example", secret: "s", wantOn: true},
		"a url without a secret fails boot":   {url: "https://antenne.example", wantErr: true},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv("ANTENNE_URL", c.url)
			t.Setenv("ANTENNE_INSTANCE", c.legacy)
			t.Setenv("ANTENNE_SECRET", c.secret)
			t.Setenv("ANTENNE_APP", "")
			cfg, err := antenneFromEnv()
			if (err != nil) != c.wantErr || (cfg != nil) != c.wantOn {
				t.Fatalf("antenneFromEnv() = %+v, %v", cfg, err)
			}
			if cfg != nil && (cfg.App != "Agenda" || cfg.Instance != "https://antenne.example") {
				t.Fatalf("config = %+v, want app Agenda on https://antenne.example", cfg)
			}
		})
	}
}
