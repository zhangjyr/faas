In Container Scheduler (ICS)
==========

The ics provides an managed and generic interface between the outside world and your function. Its job is to marshal a HTTP request accepted on the API Gateway and to invoke your chosen application. The ics is a tiny Golang tcp forwarder that
support HTTP only so far.

### Compatible to watchdog

This is an unsupported use-case for the OpenFaaS project however if your container conforms to the requirements below then the OpenFaaS API gateway and other tooling will manage and scale your service.

You will need to provide a lock-file at `/tmp/.lock` so that the orchestration system can run healthchecks on your container. If you are using Docker Swarm make sure you provide a `HEALTHCHECK` instruction in your Dockerfile - samples are given in the `faas` repository.

* Expose TCP port 8080 over HTTP
* Create `/tmp/.lock` or in whatever location responds to the OS tempdir syscall

### Deployment

~~~
dep ensure
make container
~~~
