# Moblog cloud

This is the "platformization" of a homebrewd blogging system I've been using
while traveling over the past few years, to take it from a couple of
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
  `adminserver` for authenticating users.
- The `blogrenderer` (not developed yet) reacts to pushes on the repositories
  (by watching a work queue populated by the `gitserver`) and renders the blogs
  into HTML files. Those files can then be stored in a traditional filesystem or
  in a cloud storage system like Amazon S3.
- Repository data is stored on a traditional filesystem, NFS can be used to
  share the repositories among many servers.
