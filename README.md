# Moblog cloud

[![Go Report Card](https://goreportcard.com/badge/github.com/abustany/moblog-cloud)](https://goreportcard.com/report/github.com/abustany/moblog-cloud)

This is the "platformization" of a homebrewd blogging system I've been using
while traveling over the past few years [1], to take it from a couple of
hand-crafted bash scripts to something more robust.

**This project is a work in progress, and is still in a very early stage.**

## System overview

The aim is to allow users to write their blog while traveling (offline), and to
push the new content when they have an internet connection. The blog posts are
stored as markdown files that are then rendered by [Hugo](https://gohugo.io/)
using a specific theme. Git is used for versioning and synchronizing the posts.

On the server side, we have three components:

- The `adminserver` exposes a JSON-RPC API allowing to create/edit/delete users
  and blogs. It stores its data in a SQL database. An HTML UI will be developed
  to allow normal humans to interact with the API.
- The `gitserver` speaks the [Git HTTP smart protocol](https://github.com/git/git/blob/master/Documentation/technical/http-protocol.txt)
  and receives the clones/pulls/pushes of the users. It talks to the
  `adminserver` for authenticating users, and pushes a "render" job to the work
  queue when a push happens.
- The `worker` watches the work queue and renders the blogs into HTML files.
  Those files can then be stored in a traditional filesystem or in a cloud
  storage system like Amazon S3.
- Repository data is stored on a traditional filesystem, NFS can be used to
  share the repositories among many servers.

[1] For some examples, see [Chopsticks](https://chopsticks.bustany.org/),
[Caramba](https://caramba.bustany.org/), [Salam](https://salam.bustany.org/) or
[Ni Hao](https://nihao.bustany.org/).

## Deployment

There are two main options to deploy moblog-cloud. Both require at least a Redis
server for storing sessions, and a PostgreSQL database for storing user/blog
information.

### Small scale: Omnibus deployment 🚌

moblog-cloud builds an `omnibus` binary that groups all the server-side
components as well as the admin UI into a single static binary. The generated
blogs can be stored in either Amazon S3 or on the local filesystem. In the
latter case `omnibus` will also serve the generated blogs.

The following URLs are exposed when running the `omnibus` binary:

- `/api`: JSON-RPC API, accessed by the admin UI as well as other clients like
  the mobile application
- `/git`: Git HTTP smart protocol, used for cloning/updating the blogs by Git
  clients like the mobile application, or the `scripts/mgit` Git wrapper
- `/admin`: Admin web application, that allows users to register and create new
  blogs. Publishing content to a blog via the admin UI is at the moment not
  supported and must be done using the mobile application, or manually by
  pushing Hugo-formatted markdown files in the repository's `content/post`
  directory.
- `/`: Serves the blogs, if the blog output directory is set to a local folder

### Large scale: Kubernetes 🌐

For a more resilient/scalable deployment, moblog-cloud can be deployed on
Kubernetes. See the [deployment README](deployment/README.md) for more
information.
