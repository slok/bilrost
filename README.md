# Bilrost

Setting OAUTH/OIDC on a application running in a Kubernetes cluster should be easy, Bilrost removes this pain.

Bilrost is a kubernetes controller/operator to set up oauth2/OIDC on any ingress based service. It doesn't care  what ingress controller do you use, supports multiple auth bakckends and multiple OAUTH/OIDC proxies.

Bilrost will register/create OAUTH/OIDC clients, create secrets, setup proxies, rollback if required. Ina few words, it automates the ugly work of setting the OAUTH2/OIDC security in your applications.

## Features

- Secure any external (ingress) service independent of the ingress controller.
- Can rollback security to the original state.
- Heals automatically (controller pattern/feedback loop).
- Support multiple auth backends implementations (easy to add new ones).
- Supports multiple proxy implementations (easy to add new ones).
- Two modes of securing ingresses/apps
  - `simple` by using ingress annotations.
  - `advanced` by using CRDs to express more securing options.
- Prometheus metrics ready.
- Have infinite different auth backends, e.g:
  - Dex with Github for developers ingresses.
  - Dex with Google for company internal backend ingresses.
  - Dex with other Google accounts to allow access to poeple outside the company internal backends.
- Automatic registering OIDC clients on auth backends.
- Automatically create OAUTH/OIDC client secrets (no need to be manipulating secrets).
- Setup safe settings on proxies.
- Splits responsibility about the auth backends and application security.

## How does it work

Bilrost needs 2 things:

- The `AuthBackend` is a cluster scoped CRD that has the data to be able to interact with the auth backend system of your election, for example [Dex].
- The application to be secured, this can be achieved in 2 modes:
  - Simple: An Ingress that points to the auth backend that needs to be used to secure that service.
  - Advanced: A CRD that has the data of how to secure the app and points to the ingress and setup the proxy.

As you see, this way of splitting the concerns makes the applications being secured have no need to know about how the security backends work not provide data. The `AuthBackends` could be created by different people/roles/apps and be there to be used by the people/roles/apps that secure applications.

![kubernetes-architecture](docs/img/k8s-architecture.png)

As you see in the high level architecture graph, before Bilrost you have the regular setup of ingress->service->pods. After securing with Bilrost, this will set up a proxy in a kind of [MitM][mitm] style, so every request that forwards from the ingress to the service, will need to go through the OAUTH/OIDC proxy that has been configured and registered with an auth backend, so the OAUTH2/OIDC flow is triggered.

## Getting started

The requiremets are:

- A running and working Dex.
- The Auth backend CRD registered and a running Bilrost (check [manifests]).
- An ingress that points to a running application (service, pods...).

This is an example, you will need to change the settings accordingly.

First you need to register the dex as an auth backend for bilrost, using a cluster scoped CR, this example is for a backend on a Dex in `https://dex.my.cluster.slok.dev` public URL and running in `auth` namespace `dex` service:

```yaml
apiVersion: auth.bilrost.slok.dev/v1
kind: AuthBackend
metadata:
  name: dex
spec:
  dex:
    publicURL: https://dex.my.cluster.slok.dev
    apiAddress: dex.auth.svc.cluster.local:81
```

Now you need to select this backend with Bilrost's ingress annotation and let the magic happen (this example is for an app available at `https://app.my.cluster.slok.dev` and a service at `app` at namespace `app`):

```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: app
  namespace: app
  annotations:
    auth.bilrost.slok.dev/backend: test-bilrost-dex
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

For more advanced examples check: [examples]

### Supported auth backends

- [Dex]: Bilrost will use the API to register and unregister the secured clients automatically. 

### Supported OAUTH/OIDC Proxies

- [oauth2-proxy]: Bilrost will create a deployment, service and secret, configure the proxy with OIDC settings for the auth backend and forwardthe ingress to the proxy instead the original app service.

[mitm]: https://en.wikipedia.org/wiki/Man-in-the-middle_attack
[Dex]: https://github.com/dexidp/dex
[oauth2-proxy]: https://github.com/oauth2-proxy/oauth2-proxy
[manifests]: ./manifests
[examples]: ./examples