package main

import (
	"log"
	"os"
    "fmt"
    "errors"
	"os/exec"
	"reflect"
	"text/template"

    "k8s.io/kubernetes/pkg/labels"
    "k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/api"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/util"
    "github.com/alecthomas/kingpin"
)

const (
	tpl = `events {
  worker_connections  4096;
}
    
http {
    real_ip_header    X-Forwarded-For;
    set_real_ip_from  0.0.0.0/0;
    real_ip_recursive on;

{{ range $ud, $upstream := .Upstreams }}
    upstream {{ $upstream.Name }} {
        ip_hash;
{{ range $ad, $address := $upstream.Addresses }}
        server {{ $address }};
{{ end }}
    }
{{ end }}

{{ range $sd, $server := .Servers }}
    server {
        listen      {{ $.Port }};
        server_name {{ $server.Domain }};

{{ range $ld, $location := $server.Locations }}
        location {{ $location.Path }} {
            proxy_pass http://{{ $location.Upstream }};
        }
{{ end }}
    }
{{ end }}
}`
)

var (
    cliApi  = kingpin.Flag("api", "URL to the Kubernetes API component").Default("http://localhost").OverrideDefaultFromEnvar("KUBE_NGINX_API").String()
    cliPort = kingpin.Flag("port", "Port to accept incoming connections on").Default("80").OverrideDefaultFromEnvar("KUBE_NGINX_PORT").String()
    cliCfg  = kingpin.Flag("cfg", "Nginx config file").Default("/etc/nginx/nginx.conf").OverrideDefaultFromEnvar("KUBE_NGINX_CFG").String()
)

func main() {
    kingpin.Parse()
    
    // Create a client which we can use to connect to the remote Kubernetes cluster.
    config := &client.Config{
        Host: *cliApi,
    }
    kubeClient, err := client.New(config)
    Check(err)

    // The template which will get used to expose Ingresses.
	tmpl, err := template.New("nginx").Parse(tpl)
    Check(err)
    
    var (
       ingClient   = kubeClient.Extensions().Ingress(api.NamespaceAll)
	   rateLimiter = util.NewTokenBucketRateLimiter(0.1, 1)
	   known       = &Nginx{}
    )

	// Controller loop.
	for {
		rateLimiter.Accept()
        
        nginx := NewNginx(*cliPort)
        
        // Query for the current list of ingresses.
		ingresses, err := ingClient.List(labels.Everything(), fields.Everything())
		if err != nil {
			log.Printf("Error retrieving ingresses: %v", err)
			continue
		}
        
        // Ensure we have ingress items.
        if len(ingresses.Items) <= 0 {
            log.Printf("No ingresses were found", err)
			continue
        }
        
        // Load up the pods for the service in this ingress.
        for _, i := range ingresses.Items {
            // Build a our listeners based on the ingress rules.
            for _, r := range i.Spec.Rules {
                // Add this to the Server configuration.
                svr := Server{
                    Domain: r.Host,
                }
                
                for _, pa := range r.HTTP.Paths {
                    // Add this to our list of paths to implement in Nginx.
                    np := Location{
                        Path:     pa.Path,
                        Upstream: pa.Backend.ServiceName,
                    }
                    svr.Locations = append(svr.Locations, np)
                    
                    // Add an Upstream to the list if it has not already been setup.
                    if _, ok := nginx.Upstreams[pa.Backend.ServiceName]; ok {
                        continue
                    }
                    
                    u := Upstream{
                        Name: pa.Backend.ServiceName,
                    }
                    
                    // First we need to load the service configuration from the backend.
                    s, err := kubeClient.Services(i.ObjectMeta.Namespace).Get(pa.Backend.ServiceName)
                    if err != nil {
                        log.Printf("Error retrieving service: %v", err)
                        continue
                    }
                    
                    // Now we load all the pods for this service.
                    ps, err := kubeClient.Pods(i.ObjectMeta.Namespace).List(labels.SelectorFromSet(labels.Set(s.Spec.Selector)), fields.Everything())
                    if err != nil {
                        log.Printf("Error retrieving service: %v", err)
                        continue
                    }
                    
                    // Populate the list of pod IPs.
                    for _, p := range ps.Items {
                        if p.Status.Phase != api.PodRunning {
                            continue
                        }
                        u.Addresses = append(u.Addresses, p.Status.PodIP + ":" + pa.Backend.ServicePort.String())
                    }
                    
                    // Add it to the list of upstreams.
                    nginx.Upstreams[pa.Backend.ServiceName] = u
                }
                
                nginx.Servers = append(nginx.Servers, svr) 
            }
        }
        
        // Is this already our list of ingresses.
		if reflect.DeepEqual(nginx, known) {
            log.Println("Backend has not changed - No action was taken")
			continue
		}
		
		
        // Write out the new configuration.
        if w, err := os.Create(*cliCfg); err != nil {
			log.Println("Failed to open %v: %v", tpl, err)
		} else if err := tmpl.Execute(w, nginx); err != nil {
			log.Println("Failed to write template %v", err)
		}
		err = shellOut("nginx -s reload")
        if err != nil {
            log.Println(err)
            continue
        }
        
        // We only set this at the very end to ensure the service was restarted successfully.
        known = nginx
        log.Println("Successfully reloaded Nginx with updated Ingresses")
	}
}

// Helper function execute commands on the commandline.
func shellOut(cmd string) error {
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
        return errors.New(fmt.Sprintf("Failed to execute %v: %v, err: %v", cmd, string(out), err))
	}
    return nil
}

// Helper function to exit the application is errors.
func Check(e error) {
    if e != nil {
        log.Println(e)
        os.Exit(1)
    }
}