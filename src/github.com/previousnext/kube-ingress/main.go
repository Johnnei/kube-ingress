package main

import (
	"fmt"

	"github.com/alecthomas/kingpin"
	"k8s.io/kubernetes/pkg/api"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/util"
)

var (
	cliApi  = kingpin.Flag("api", "URL to the Kubernetes API component").Default("http://localhost").OverrideDefaultFromEnvar("KUBE_NGINX_API").String()
	cliPort = kingpin.Flag("port", "Port to accept incoming connections on").Default("80").OverrideDefaultFromEnvar("KUBE_NGINX_PORT").String()
	cliCfg  = kingpin.Flag("cfg", "Nginx config file").Default("/etc/nginx/nginx.conf").OverrideDefaultFromEnvar("KUBE_NGINX_CFG").String()
)

func main() {
	kingpin.Parse()

	// Create a client which we can use to connect to the remote Kubernetes cluster.
	kubeClient, err := client.New(&client.Config{
		Host: *cliApi,
	})
	Check(err)

	ingClient := kubeClient.Extensions().Ingress(api.NamespaceAll)
	rl := util.NewTokenBucketRateLimiter(0.1, 1)
	svcs := NewServices(kubeClient)
	nginx, err := NewNginx(*cliPort)
	Check(err)

	// Controller loop.
	for {
		rl.Accept()

		// Query for the current list of ingresses.
		ings, err := ingClient.List(labels.Everything(), fields.Everything())
		if err != nil {
			fmt.Printf("Error retrieving ingresses: %v\n", err)
			continue
		}

		// Ensure we have ingress items.
		if len(ings.Items) <= 0 {
			fmt.Printf("No ingresses were found\n", err)
			continue
		}

		var (
			servers   = make(map[string][]Location)
			upstreams = make(map[string][]string)
		)

		// Load up the pods for the service in this ingress.
		for _, i := range ings.Items {
			// Build a our listeners based on the ingress rules.
			for _, r := range i.Spec.Rules {
				var locations []Location

				for _, pa := range r.HTTP.Paths {
					name := MergeNameNameSpace(i.ObjectMeta.Namespace, pa.Backend.ServiceName)

					// Get the list of backends from this rule.
					list, err := svcs.Get(name)
					if err != nil {
						fmt.Println("Failed to get service pods: %s", err)
						continue
					}

					// We have a set of IPs so we are now free to add the upstream and location
					// to our nginx configuration and be a part of the next reload.
					upstreams[name] = list

					// Add this to our list of paths to implement in Nginx.
					l := Location{
						Path:     pa.Path,
						Upstream: name,
					}
					locations = append(locations, l)
				}

				// Add our list of generated locations to the nginx backend. These have been verified
				// as having a backend so this is a safe operation.
				if len(locations) > 0 {
					servers[r.Host] = locations
				}
			}
		}

		// Add the upstreams and servers to the nginx configuration.
		nginx.SetServers(servers)
		nginx.SetUpstreams(upstreams)

		err = nginx.Reload()
		if err != nil {
			fmt.Println(err)
			continue
		}

		fmt.Println("Successfully reloaded Nginx with updated Ingresses")
	}
}
