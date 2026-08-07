package router

import "testing"

func TestExpandEnv(t *testing.T) {
	t.Setenv("FOO", "bar")
	got := expandEnv("value: ${FOO}, other: ${MISSING:-default}, empty: ${EMPTY:-}")
	want := "value: bar, other: default, empty: "
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestRouterResolve(t *testing.T) {
	cfg := &Config{
		Routes: []Rule{
			{Match: map[string]string{"severity": "critical"}, Channels: []string{"telegram", "bale"}},
			{Match: map[string]string{"alertname": "KafkaTopicStale"}, Channels: []string{"bale"}},
		},
		DefaultChannels: []string{"telegram"},
	}
	r := New(cfg)

	got := r.Resolve(map[string]string{"severity": "critical", "alertname": "X"})
	if len(got) != 2 || got[0] != "telegram" || got[1] != "bale" {
		t.Fatalf("critical rule failed: %v", got)
	}

	got = r.Resolve(map[string]string{"alertname": "KafkaTopicStale"})
	if len(got) != 1 || got[0] != "bale" {
		t.Fatalf("kafka rule failed: %v", got)
	}

	got = r.Resolve(map[string]string{"alertname": "SomethingElse"})
	if len(got) != 1 || got[0] != "telegram" {
		t.Fatalf("default rule failed: %v", got)
	}
}
