<p align="center">
    <img src="docs/img/logo.png" width="30%" align="center" alt="bilrost">
</p>

# Bilrost [![Build Status][ci-image]][ci-url] [![Go Report Card][goreport-image]][goreport-url]

Setting OAUTH2/OIDC in Kubernetes running apps should be easy, Bilrost solves this and removes the pain.

Bilrost is a kubernetes controller/operator to set up OAUTH2/OIDC on any ingress based service. It doesn't care  what ingress controller do you use, it supports multiple auth backends and multiple OAUTH2/OIDC proxies.

Bilrost will register/create OAUTH2/OIDC clients, create secrets, setup proxies, rollback if required... In a few words, it automates the ugly work of setting the OAUTH2/OIDC security in your applications.

## Table of contents

- [Features](#features)
- [How does it work](#how-does-it-work)
- [Getting started](#getting-started)
- [Advanced examples](#advanced-examples)
- [Supported auth backends](#supported-auth-backends)
- [Supported OAUTH2 OIDC proxies](#supported-oauth2-oidc-proxies)
- [F.A.Q](#faq)

## Features

- Secure any external (ingress) service independent of the ingress controller.
- Can rollback security to the original state.
- Heals automatically (controller pattern/feedback loop).
- Support multiple auth backends implementations (easy to add new ones).
- Supports multiple proxy implementations (easy to add new ones).
- Two modes of securing ingresses/app `simple` and `advanced`.
- Prometheus metrics ready.
- Have infinite different auth backends, e.g:
  - Dex with Github for developers ingresses.
  - Dex with Google for company internal backend ingresses.
  - Dex with other Google accounts to allow access to poeple outside the company internal backends.
- Automatic registering OIDC clients on auth backends.
- Automatically create OAUTH2/OIDC client secrets (no need to be manipulating secrets).
- Setup safe settings on proxies.
- Splits responsibility about the auth backends and application security.

## How does it work

Bilrost needs 2 things:

- The `AuthBackend` is a cluster scoped CRD that has the data to be able to interact with the auth backend system of your election, for example [Dex].
- The public application to be secured, this can be achieved in 2 ways:
  - First select the ingress to be secured by adding an annotation that points to the auth backend that will act as the auth entity.
  - As an optional step, along with the ingress annotatiton you can set advanced settings for the proxy using a CR called `IngressAuth`, this has the same name and namespace as the ingress.

After this Bilrost is ready to start the setup of a secure OAUTH2/OIDC application like this:

![kubernetes-architecture](docs/img/k8s-architecture.png)

As you see in the high level architecture graph, before Bilrost you have the regular setup of ingress->service->pods. After securing with Bilrost, it will set up a proxy in a kind of [MitM][mitm] style, so every request that forwards from the ingress to the service, will need to go through the OAUTH2/OIDC proxy that has been configured and registered with an auth backend, so the OAUTH2/OIDC flow is triggered. This is how eliminates the need of a specific ingress controller and is compatible with any setup.

## Getting started

The requirements are:

- A running and working Dex.
- The Auth backend CRD registered and a running Bilrost (check [manifests]).
- An ingress that points to a running application (service, pods...).

This is an example, you will need to change the settings accordingly.

First you need to register the dex as an auth backend for bilrost, using a cluster scoped CR, this example is for a backend on a Dex in `https://dex.my.cluster.slok.dev` public URL and running in `auth` namespace `dex` service:

```yaml
apiVersion: auth.bilrost.slok.dev/v1
kind: AuthBackend
metadata:
  name: my-dex
spec:
  dex:
    publicURL: https://dex.my.cluster.slok.dev
    apiAddress: dex.auth.svc.cluster.local:81
```

Now you need to select this backend, to do this you will need to use the bilrost backed ingress annotation `auth.bilrost.slok.dev/backend` (**The annotation is mandatory**). and let the magic happen (this example is for an app available at `https://app.my.cluster.slok.dev` and a service `app`, namespace `app`):

```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: app
  namespace: app
  annotations:
    auth.bilrost.slok.dev/backend: my-dex
spec:
  rules:
    - host: app.my.cluster.slok.dev
      http:
        paths:
          - backend:
              serviceName: app
              servicePort: 80
            path: /
```

In case you want (**The CR is optional**) to set advanced settings to secure the app instead of the defaults you can use a CRD that should live in the same namespace of the ingress and the same name. e.g

```yaml
apiVersion: auth.bilrost.slok.dev/v1
kind: IngressAuth
metadata:
  name: app
  namespace: app
spec:
  authSettings:
    scopeOrClaims: ["email", "profile", "groups", "offline_access"]
  oauth2Proxy:
    image: "quay.io/oauth2-proxy/oauth2-proxy:latest"
    replicas: 4
    resources:
      requests:
        memory: "64Mi"
        cpu: "250m"
      limits:
        memory: "128Mi"
        cpu: "500m"
```

## Advanced examples

For more advanced examples check [examples] dir, be aware of the `CHANGE_ME` prefix on the lines that you will need to change/pay attention.

## Supported auth backends

- [Dex]: Will set the applicaiton ready to be used in a Dex instance by:
  - Creating a new Client secret.
  - Storing this secret internally.
  - Register the app with a client ID and the generated client secret using the Dex API.

## Supported OAUTH2 OIDC proxies

- [oauth2-proxy]: Will set up an oauth2-proxy by:
  - Create a Service for the proxy.
  - Create a Kubernetes secret with the OIDC client information (already ready on the auth backend).
  - Setup a deployment with the proxy configured to use the auth backend and the original app service as the upstream.
  - Store a backup of the app's ingress original data.
  - Update the app ingress to forward the traffic to the proxy.

## F.A.Q

### Do I need bilrost?

Depends, if you already have a way of securing public services with OAUTH2/OIDC in an automated way, I would say that you don't need Bilrost.

For example If you already use [nginx-controller] with `auth-url` and  `auth-signin` annotations ([more info here][nginx-oauth2]) you probably don't need Bilrost.

But if you use [Skipper], [Traefik] or any other ingress controller, you can use Bilrost to setup fast and easy OAUTH2/OIDC on public services.


### How can I deploy Bilrost?

Check [deployment example][bilrost-deployment]

### Where are the CRDs?

You can register Bilrost CRDs with [these][CRD] manifests.

### How I manage the secrets of OAUTH2/OIDC clients?

You don't, Bilrost manages them, it generates random secrets, stores them on kubernetes, sets up on the proxies and registers them on the auth backends.

### Can I rotate OAUTH2/OIDC client secrets?

Depends on the auth backend used. Here are the ways in the different auth backends:

#### Dex

Bilrost stores the autogenerated client secrets on its running namespace.

You can get all of them with:

```bash
kubectl -n {BILROST_NS} get secrets -l app.kubernetes.io/component=dex-client-data
```

If you delete those secrets, on the next resync interval, Bilrost will generate new secrets and setup everything again.

### Why `ClusterRoleBinding`?

You only need one bilrost per cluster, this Bilrost instance needs to manage deployments, secrets, ingresses... outside its namespace, this means that needs to access at a cluster scope level.

If you are concerned about this cluster wide security you can use multiple role bindings, one `ClusterRoleBinding` for cluster scope resources (`AuthBackends`) and multiple `RoleBinding` for each namespace access you want Bilrst access namespaced resources (`Secrets`, `Ingress`, `IngressAuth`, `Deployments`).

### Can I rollback a secured application?

Yes, Bilrost will detect that the ingress is no longer require to be secured and will trigger a rollback process to let the ingress as it was. This can be triggerend in different ways:

- Deleting the ingress annotation.
- Deleting the ingress.
- Deleting security CRD.

### What triggers a reconciliation loop?

- At regular intervals all ingresses (`5m` by default, use `--resync-interval` flag for custom interval).
- Updates on `Ingress` core resources.
- Updates on `IngressAuth` custom resources (CR).

### I'm not happy with the default proxy settings

It's ok, use the `IngressAuth` CR in case you want special settings for the proxy, like number of replicas or setting resources.

The ingress annotation method without CR is a fast and simple way of enabling and disabling security, make tests and enable security in a temporary way.

### If I use the CR, do I need to use the ingress annotation?

Yes, at the begginning we though of use the annotation or the CR to enable, but that opens corner cases and adds internal complexity, that translates in bugs.

Making the annotation a requirement also has good side effects, like:

Only have a single way of enabling/disabling bilrost security on an ingress, the annotation. If not this could mean that sometimes you would enable this with an annotation and others wiht the CR and we don't like making the same thing in different ways.

Also, although you can have the CR present, with the annotation you can enable and disable the security in a fast way without the need of deleting resources.

### Do we have Bilrost metrics?

Yes, we support [Prometheus] metrics, by default metrics will be served in `0.0.0.0:8081/metrics`.

### Do you support https `Service`s

No, this will come with an avialable setting in the `IngressAuth` CR. By default and without advanced options will be http.

### In what state is this controller?

Is not stable yet.

If you are using it a medium-big scale please let us know how is working in case we need to optimize or fix parts of the controller.

### Is this another ingress controller?

Well is an ingress controller, but only sets up OAUTH2/OIDC security, it has small responsibility, so in other words no, this will not replace Ngix, Skipper, Traefik...

### How about having multiple Bilrost instances?

Although is not required because of its async nature and you could configure the number of workers to run, should be safe to have multiple instances, and in case of sharding you could have instances per namespace if you want.

### Are you planning to support more auth backends and proxies?

Yes.

In the short term we are planning Auth0 `AuthBackend` as an alternative to Dex.

Regarding auth proxy we are planning what would it take to support [nginx-controller] annotation and setup process, if it makes sense we could do it.

Anyway, if you want support for other kinds of auth backends and/or proxies, please open an Issue, that would be awesome.


### What does Bilrost mean?

Well is another name for [Bifrost], but there are a lot of projects called Bifrost, including in Kubernetes landscape.

[ci-image]: https://github.com/slok/bilrost/workflows/CI/badge.svg
[ci-url]: https://github.com/slok/bilrost/actions
[goreport-image]: https://goreportcard.com/badge/github.com/slok/bilrost
[goreport-url]: https://goreportcard.com/report/github.com/slok/bilrost
[mitm]: https://en.wikipedia.org/wiki/Man-in-the-middle_attack
[Dex]: https://github.com/dexidp/dex
[oauth2-proxy]: https://github.com/oauth2-proxy/oauth2-proxy
[manifests]: ./manifests
[examples]: ./examples
[Bifrost]: https://en.wikipedia.org/wiki/Bifr%C3%B6st
[bilrost-deployment]: ./manifests/bilrost-deployment.yaml
[CRD]: ./manifests/crd
[nginx-oauth2]: https://github.com/kubernetes/ingress-nginx/tree/master/docs/examples/auth/oauth-external-auth
[Skipper]: https://github.com/zalando/skipper/
[Traefik]: https://github.com/containous/traefik
[nginx-controller]: https://github.com/kubernetes/ingress-nginx
[Prometheus]: https://prometheus.io/