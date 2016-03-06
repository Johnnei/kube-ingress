package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReload(t *testing.T) {
	b := Backend{
		Servers: map[string][]Location{
			"server1": []Location{
				Location{
					Path:     "/v1",
					Upstream: "foo",
				},
				Location{
					Path:     "/v2",
					Upstream: "bar",
				},
			},
		},
		Upstreams: map[string][]string{
			"foo": []string{
				"1.2.3.4",
				"1.2.3.5",
			},
			"bar": []string{
				"1.2.3.6",
				"1.2.3.7",
			},
		},
	}
	n := Nginx{
		New:  b,
		Prev: b,
	}
	err := n.Reload()
	assert.Equal(t, "Configuration has not changed. Not reloading the nginx daemon.", err.Error(), "Don't need to restart nginx")
}
