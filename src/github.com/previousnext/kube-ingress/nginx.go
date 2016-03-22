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

{{ range $ud, $addresses := .New.Upstreams }}
    upstream {{ $ud }} {
        ip_hash;
{{ range $ad, $address := $addresses }}
        server {{ $address }};
{{ end }}
    }
{{ end }}

{{ range $sd, $servers := .New.Servers }}
    server {
        listen      {{ $.Port }};
        server_name {{ $sd }};
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

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

type Backend struct {
	Servers   map[string][]Location
	Upstreams map[string][]string
}

type Nginx struct {
	Template *template.Template
	Port     string

	// New configuration to be compared with against the private values.
	New Backend

	// The previously reloaded values.
	Prev Backend
}

func (n *Nginx) SetServers(l map[string][]Location) {
	n.New.Servers = l
}

func (n *Nginx) SetUpstreams(l map[string][]string) {
	n.New.Upstreams = l
}

func (n *Nginx) Reload() error {
	// Has the configuration changed? If it has we can reload.
	if reflect.DeepEqual(n.New.Servers, n.Prev.Servers) && reflect.DeepEqual(n.New.Upstreams, n.Prev.Upstreams) {
		return errors.New("Configuration has not changed. Not reloading the nginx daemon.")
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

	// Set the previous values so Nginx doesn't continue to restart.
	n.Prev.Servers = n.New.Servers
	n.Prev.Upstreams = n.New.Upstreams

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
		New: Backend{
			Servers:   make(map[string][]Location),
			Upstreams: make(map[string][]string),
		},
		Prev: Backend{
			Servers:   make(map[string][]Location),
			Upstreams: make(map[string][]string),
		},
	}, nil
}
