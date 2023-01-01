---
title: Introduction
slug: introduction
desc: Modular building blocks for building collaborative applications like Google Docs and Figma.
sort: 1
---

> This documentation website is a work in progress. The best source of information is still the Yjs README and the yjs-demos repository.

Yjs is a high-performance [CRDT](https://en.wikipedia.org/wiki/Conflict-free_replicated_data_type) for building collaborative applications that sync automatically.
It exposes its internal CRDT model as shared data types that can be manipulated concurrently. Shared types are similar to common data types like Map and Array. They can be manipulated, fire events when changes happen, and automatically merge without merge conflicts.

# Quick Start

This is a working example of how shared types automatically sync. We also have a getting-started guide, API documentation, and lots of live demos with source code.

```js
import * as Y from 'yjs'

// Yjs documents are collections of
// shared objects that sync automatically.
const ydoc = new Y.Doc()
// Define a shared Y.Map instance
const ymap = ydoc.getMap()
ymap.set('keyA', 'valueA')

// Create another Yjs document (simulating a remote user)
// and create some conflicting changes
const ydocRemote = new Y.Doc()
const ymapRemote = ydocRemote.getMap()
ymapRemote.set('keyB', 'valueB')

// Merge changes from remote
const update = Y.encodeStateAsUpdate(ydocRemote)
Y.applyUpdate(ydoc, update)

// Observe that the changes have merged
console.log(ymap.toJSON()) // => { keyA: 'valueA', keyB: 'valueB' }
```

# Editor Support

Yjs supports several popular text and rich-text editors. We are working with different projects to enable collaboration-support through Yjs.

# Network Agnostic 📡


Yjs doesn't make any assumptions about the network technology you are using. As long as all changes eventually arrive, the documents will sync. The order in which document updates are applied doesn't matter.
You can integrate Yjs into your existing communication infrastructure, or use one of the several existing network providers that allow you to jump-start your application backend.
Scaling shared editing backends is not trivial. Most shared editing solutions depend on a single source of truth - a central server - to perform conflict resolution. Yjs doesn't need a central source of truth. This enables you to design the backend using ideas from distributed system architecture. In fact, Yjs can be scaled indefinitely as it is shown in the y-redis section.
Another interesting application for Yjs as a data model for decentralized and Local-First software.

