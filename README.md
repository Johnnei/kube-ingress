Kubernetes - Ingress
====================

An Ingress controller managing Nginx configuration

## Diagram

![Diagram](/diagram.png "Diagram")

## Setup

```bash
$ docker run -e KUBE_NGINX_API=http://172.17.20.2:8080 -p 80:80 --name ingress previousnext/kube-ingress:release-0.0.1
```

## Build

We use a tool called `gb`. To install run:

```bash
$go get github.com/constabulary/gb/...
```

To build the project run:

```bash
$ make
```

