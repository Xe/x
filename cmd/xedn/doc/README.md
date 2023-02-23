# XeDN

Xe's CDN replacement service

## Goal

XeDN is a CDN replacement service. It is a tool that enables you to
serve files from a central storage pool (such as Backblaze B2) in a
way that caches all those files locally. This lets you put several
instances of XeDN in different datacentres globally and then each
instance will act as a local cache. This strategy lets you decrease
observable latency for users.

The goal of this service is to replace Cloudflare for my own uses.

## Design

At a high level, XeDN is a caching HTTP proxy. It caches files locally
using [BoltDB](https://github.com/etcd-io/bbolt) and serves from that
cache whenever possible. XeDN pulls files from its source (currently
over HTTP, but this can be changed in the future) and aggressively
caches them in the database. When each file is cached, it has a
default lifetime of one week. This lifetime is extended every time a
file is requested, hopefully making sure that each file that is
commonly used is never requested from backend servers. This does make
genuinely updating content hard, so users of XeDN are encouraged to
assume that the backend is an _append-only_ store.

