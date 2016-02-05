package main

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"text/template"
)

const (
	tpl = `events {
  worker_connections  4096;
}
    
http {
    real_ip_header    X-Forwarded-For;
    set_real_ip_from  0.0.0.0/0;
    real_ip_recursive on;

{{ range $ud, $addresses := .Upstreams }}
    upstream {{ $ud }} {
        ip_hash;
{{ range $ad, $address := $addresses }}
        server {{ $address }};
{{ end }}
    }
{{ end }}

{{ range $sd, $servers := .Servers }}
    server {
        listen      {{ $.Port }};
        server_name {{ $sd }};

{{ range $ld, $location := $servers }}
        location {{ $location.Path }} {
            proxy_pass http://{{ $location.Upstream }};
        }
{{ end }}
    }
{{ end }}
}`
)

type Location struct {
	Path     string
	Upstream string
}

type Nginx struct {
	Template *template.Template
	Port     string

	// New configuration to be compared with against the private values.
	Servers   map[string][]Location
	Upstreams map[string][]string

	// Private values only used for comparison.
	servers   map[string][]Location
	upstreams map[string][]string
}

func (n *Nginx) SetServers(l map[string][]Location) {
	n.Servers = l
}

func (n *Nginx) SetUpstreams(l map[string][]string) {
	n.Upstreams = l
}

func (n *Nginx) Reload() error {
	// Has the configuration changed? If it has we can reload.
	if reflect.DeepEqual(n.Servers, n.servers) && reflect.DeepEqual(n.Upstreams, n.upstreams) {
		fmt.Println("Configuration has not changed. Not reloading the nginx daemon.")
		return nil
	}

	// Build a new configuration.
	if w, err := os.Create(*cliCfg); err != nil {
		return errors.New(fmt.Sprintf("Failed to open %v: %v\n", tpl, err))
	} else if err := n.Template.Execute(w, n); err != nil {
		return errors.New(fmt.Sprintf("Failed to write template %v\n", err))
	}

	// Reload the active daemon.
	err := shellOut("nginx -s reload")
	if err != nil {
		return err
	}
	return nil
}

// Standard method for loading a Nginx configuration.
func NewNginx(p string) (*Nginx, error) {
	// The template which will get used to expose Ingresses.
	tmpl, err := template.New("nginx").Parse(tpl)
	if err != nil {
		return &Nginx{}, err
	}

	// Return the object so we can act upon it.
	return &Nginx{
		Template: tmpl,
		Port:     p,

		// Real configuration.
		Servers:   make(map[string][]Location),
		Upstreams: make(map[string][]string),

		// Comparison properties.
		servers:   make(map[string][]Location),
		upstreams: make(map[string][]string),
	}, nil
}
