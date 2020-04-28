# Bilrost

Setting OAUTH2/OIDC in Kubernetes running apps should be easy, Bilrost achieves this and removes the pain easily.

Bilrost is a kubernetes controller/operator to set up OAUTH2/OIDC on any ingress based service. It doesn't care  what ingress controller do you use, supports multiple auth bakckends and multiple OAUTH2/OIDC proxies.

Bilrost will register/create OAUTH2/OIDC clients, create secrets, setup proxies, rollback if required. Ina few words, it automates the ugly work of setting the OAUTH2/OIDC security in your applications.

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
- Automatically create OAUTH2/OIDC client secrets (no need to be manipulating secrets).
- Setup safe settings on proxies.
- Splits responsibility about the auth backends and application security.

## How does it work

Bilrost needs 2 things:

- The `AuthBackend` is a cluster scoped CRD that has the data to be able to interact with the auth backend system of your election, for example [Dex].
- The application to be secured, this can be achieved in 2 modes:
  - Simple: An ingress annotation that points to the auth backend that needs to be used to secure that service.
  - Advanced: A CRD that has the data of how to secure the app and points to the ingress and setup the proxy.

As you see, this way of splitting the concerns makes the applications being secured have no need to know about how the security backends work not provide data. The `AuthBackends` could be created by different people/roles/apps and be there to be used by the people/roles/apps that secure applications.

![kubernetes-architecture](docs/img/k8s-architecture.png)

As you see in the high level architecture graph, before Bilrost you have the regular setup of ingress->service->pods. After securing with Bilrost, this will set up a proxy in a kind of [MitM][mitm] style, so every request that forwards from the ingress to the service, will need to go through the OAUTH2/OIDC proxy that has been configured and registered with an auth backend, so the OAUTH2/OIDC flow is triggered.

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
  name: my-dex
spec:
  dex:
    publicURL: https://dex.my.cluster.slok.dev
    apiAddress: dex.auth.svc.cluster.local:81
```

Now you need to select this backend, we will use the simple way of using the Bilrost's ingress annotation (`auth.bilrost.slok.dev/backend`) and let the magic happen (this example is for an app available at `https://app.my.cluster.slok.dev` and a service `app`, namespace `app`):

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

## Advanced examples

For more advanced examples check [examples] dir, be aware of the `CHANGE_ME` prefix on the lines that you will need to change/pay attention.

## Supported auth backends

- [Dex]: Will set the applicaiton ready to be used in a Dex instance by:
  - Creating a new Client secret.
  - Storing this secret internally.
  - Register the app with a client ID and the generated client secret using the Dex API.

## Supported OAUTH2/OIDC proxies

- [oauth2-proxy]: Will set up an oauth2-proxy by:
  - Create a Service for the proxy.
  - Create a Kubernetes secret with the OIDC client information (already ready on the auth backend).
  - Setup a deployment with the proxy configured to use the auth backend and the original app service as the upstream.
  - Store a backup of the app's ingress original data.
  - Update the app ingress to forward the traffic to the proxy.

## F.A.Q

### Can I rollback a secured application?

Yes, Bilrost will detect that the ingress is no longer require to be secured and will trigger a rollback process to let the ingress as it was. This can be triggerend in different ways:

- Deleting the ingress annotation.
- Deleting the ingress.
- Deleting security CRD.

### What triggers a reconciliation loop?

Apart from the regular interval reconcliation (every 3m).

- Updates on ingresses
- Updates on `AppSecurity` CRs.
- Update on Bilrost generated `Services`, `Secrets`, `Deployments`.

### I'm not happy with the default proxy settings

It's ok, use the CRs in case you want special settings for the proxy, like number of replicas or setting resources.

The ingress annotation method is a fast and simple way of enabling and disabling security, make tests and enable security in a temporary way.

### In what state is this controller?

Is not stable yet.

If you are using it a medium-big scale please let us know how is working in case we need to optimize or fix parts of the controller.

### Is this another ingress controller?

Well is an ingress controller, but only sets up OAUTH2/OIDC security, it has small responsibility, so in other words no, this will not replace Ngix, Skipper, Traefik...

### What does Bilrost mean?

Well is another name for [Bifrost], but there are a lot of projects called Bifrost, including in Kubernetes landscape.

### How about having multiple Bilrost instances?

Although is not required because of its async nature and you could configure the number of workers to run, should be safe to have multiple instances, and in case of sharding you could have instances per namespace if you want.

[mitm]: https://en.wikipedia.org/wiki/Man-in-the-middle_attack
[Dex]: https://github.com/dexidp/dex
[oauth2-proxy]: https://github.com/oauth2-proxy/oauth2-proxy
[manifests]: ./manifests
[examples]: ./examples
[Bifrost]: https://en.wikipedia.org/wiki/Bifr%C3%B6st