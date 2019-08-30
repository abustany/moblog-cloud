# Kubernetes deployment for moblog-cloud

This set of files allows deploying the various services of moblog-cloud on a
Kubernetes cluster. The manifests are designed to be used using Kustomize.

The base layer includes the moblog-cloud services only, and depend on a number
of property files to be populated. The dev layer gives an example of populated
property files, and also includes manifests for deploying services required by
moblog-cloud like a frontend server to serve the generated blogs, a Redis
instance and a Postgres database. The dev layer is suitable for easy local
development, but should **NOT BE USED FOR PRODUCTION DEPLOYMENTS** (if only
because there are passwords hardcoded in there).

The dev layer can be deployed to a running Kubernetes cluster with the following
command: `kubectl apply -k dev`.

The Docker images for the various services can be generated using the Makefile
in the `server` directory, for example to generate the Docker image of
adminserver run `make docker-adminserver`. The Docker image for the admin UI can
be generated from the Dockerfile in the `admin-ui` directory.
