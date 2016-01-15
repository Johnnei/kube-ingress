package main

type Nginx struct {
    Port      string              // The port to run the Nginx on.
    Servers   []Server            // A list of servers which will use Upstreams as a backend.
    Upstreams map[string]Upstream // A list of upstream backends for Servers.
}

type Upstream struct {
    Name      string
    Addresses []string
}

type Server struct {
    Domain    string
    Locations []Location
}

type Location struct {
    Path     string
    Upstream string
}

// Standard method for loading a Nginx configuration.
func NewNginx(p string) *Nginx {
    return &Nginx{
        Port:      p,
        Upstreams: make(map[string]Upstream),
    }
}