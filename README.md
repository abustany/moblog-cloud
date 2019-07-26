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
