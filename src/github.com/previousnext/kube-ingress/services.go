package main

import (
    "fmt"
    "errors"
    
    "k8s.io/kubernetes/pkg/api"
    "k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
    "k8s.io/kubernetes/pkg/util"
    client "k8s.io/kubernetes/pkg/client/unversioned"
)

type Services struct {
    Client *client.Client
	List   map[string][]string
}

func (s *Services) Start() {
    rl := util.NewTokenBucketRateLimiter(0.1, 1)

    for {
        rl.Accept()

        // First we need to load the services.
        svcs, err := s.Client.Services("").List(labels.Everything())
        if err != nil {
            fmt.Printf("Error retrieving services: %v", err)
            continue
        }

        // Now we go over all the services and associate the pod IP addresses
        // to each of the services.
        for _, svc := range svcs.Items {
            var addrs []string
            
            ps, err := s.Client.Pods(svc.ObjectMeta.Namespace).List(labels.SelectorFromSet(labels.Set(svc.Spec.Selector)), fields.Everything())
            if err != nil {
                fmt.Printf("Error retrieving service: %v", err)
                continue
            }
            
            // Add all the running pods to the list.
            for _, p := range ps.Items {
                if p.Status.Phase != api.PodRunning {
                    fmt.Printf("Skipping pod %s for service %s", p.Name, svc.Name)
                    continue
                }
                fmt.Printf("Added pod %s for service %s", p.Name, svc.Name)
                addrs = append(addrs, p.Status.PodIP+":80")
            }
            
            // Ensure we have some addresses, if we don't, we don't have to
            // worry about adding this service.
            if len(addrs) <= 0 {
                fmt.Printf("The service %s did not contain any upstream servers", svc.ObjectMeta.Name)
                continue
            }
            
            fmt.Printf("Added the service: %v", svc.ObjectMeta.Name)
            s.List[svc.ObjectMeta.Name] = addrs
        }
    }
}

func (s *Services) Get(n string) ([]string, error) {
    if val, ok := s.List[n]; ok {
        return val, nil
    }
    return []string{}, errors.New(fmt.Sprintf("Cannot find the service: %s\n", n))
}

// Standard method for loading a Services object.
func NewServices(c *client.Client) *Services {
	s := &Services{
        Client: c,
		List:   make(map[string][]string),
	}
    
    // Start the continual process of pull the services and
    // associated pods.
    go s.Start()
    
    // Return the object so we can query it.
    return s    
}
