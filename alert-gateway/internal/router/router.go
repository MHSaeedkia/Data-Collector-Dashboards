package router

// Router holds the configured rules and decides, for each incoming alert,
// which channels it should be sent to.
type Router struct {
	cfg *Config
}

func New(cfg *Config) *Router {
	return &Router{cfg: cfg}
}

// Resolve takes an alert's labels (e.g. severity, alertname, team) and returns
// the list of channel names it should be sent to.
// The first rule whose key-value pairs all match the labels wins.
func (r *Router) Resolve(labels map[string]string) []string {
	for _, rule := range r.cfg.Routes {
		if matches(rule.Match, labels) {
			return rule.Channels
		}
	}
	return r.cfg.DefaultChannels
}

// ChannelConfig returns a channel by name so a sender can be built.
func (r *Router) ChannelConfig(name string) (ChannelConfig, bool) {
	cc, ok := r.cfg.Channels[name]
	return cc, ok
}

// matches checks that all of want's key-value pairs exist in got and are equal.
// An empty match (with no conditions at all) never matches, to prevent
// accidentally wide-open rules.
func matches(want, got map[string]string) bool {
	if len(want) == 0 {
		return false
	}
	for k, v := range want {
		if got[k] != v {
			return false
		}
	}
	return true
}
