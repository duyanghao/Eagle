Eagle
=====

Eagle is a lightweight and intelligent p2p based docker image distribution system.

<img src="https://github.com/duyanghao/eagle/blob/master/docs/images/logo.png" width=200px/>

# Features

* Non-invasive: Eagle can seamlessly support docker for distributing images. 
* High-availability: Eagle supports high-availability in both client-side and server-side, no component is a single point of failure. 
* SSI(Seeder Storage Interface). Eagle Seeder plugs into reliable blob storage options, like local FileSystem or S3. The seeder storage interface is simple and new options are easy to add.
* Peer optimal arithmetic: Eagle supports peer optimal arithmetic to improve performance and save cross-IDC bandwidth.  
* Host level speed limit: Many downloading tools(wget/curl) only have rate limit for the current download task, but Eagle also provides rate limit for the entire host.
* LRUCache delete policy: Both Seeder and EagleClient achieves the LRUCache delete policy.
* Lightweight: Eagle consists of only several necessary components, which makes it understandable, maintainable and easy-to-use.

# Architecture

The principle of eagle is kept as simple as possible and can be illustrated as follows:

![](docs/images/eagle_arch.png)

- Proxy
  - Deployed on every host
  - Proxy the blob request(EagleClient => Original Request)
- EagleClient
  - Announces available content to tracker
  - Connects to peers returned by tracker to download or upload content
- Seeder
  - Stores blobs as files on disk backed by pluggable storage (e.g. FileSystem, S3)
  - Provides meta info of blob to EagleClient and acts as the first uploader
- [Tracker](https://github.com/chihaya/chihaya)
  - Tracks which peers have what content (both in-progress and completed)
  - Provides ordered lists of peers to connect to for any given blob
- Origin
  - Docker Distribution

## Workflow

The workflow of Eagle shows below:

![](docs/images/eagle_process.svg)

For a more detailed description of Eagle design, refer to [design document](docs/design/design.md). 

# Comparison With Other Projects

## [Dragonfly from Alibaba](https://github.com/dragonflyoss/Dragonfly)

Dragonfly provides many great features, such as `interruption resuming capability`, `Host level speed limit` and so on, which makes it the most popular p2p based image distribution solution. And more recently it becomes the [CNCF Incubating Project](https://www.cncf.io/projects/).  

One drawback of Dragonfly is that it doesn't support high-availability, and its central supernode design makes performance degrade linearly as either blob size or cluster size increases.     

## [kraken from uber](https://github.com/uber/kraken)

Kraken uses several components, such as `Agent`, `Origin`, `Tracker`, `Proxy` and `Build-Index`, combined with its own designed driver protocol to build a p2p distribution system.
 
Eagle uses almost the same components with [kraken](https://github.com/uber/kraken), but it is more compact and simple as it uses [BitTorrent protocol](http://bittorrent.org/beps/bep_0003.html) underlayer and drops some unessential components.   

## Suggestion

In my opinion, both Dragonfly and Kraken are a good option inside a large, established enterprise since each of them is complicated enough and has its own set of advantages. However I would recommend the Eagle if you want a more flexible and lightweight solution since eagle is absolutely more compact, simple and maintainable compared with them.     

## TODO

* Peer optimal arithmetic
* Push notification mechanism

## Refs

* [Dragonfly](https://github.com/dragonflyoss/Dragonfly)
* [kraken](https://github.com/uber/kraken)
* [FID: A Faster Image Distribution System for Docker Platform](https://ieeexplore.ieee.org/stamp/stamp.jsp?arnumber=8064123)
* [The BitTorrent Protocol Specification](http://bittorrent.org/beps/bep_0003.html)
* [oci-torrent](https://github.com/hustcat/oci-torrent)
* [tracker](https://github.com/chihaya/chihaya)
* [torrent](https://github.com/anacrolix/torrent)
* [etcd Client Design](https://github.com/etcd-io/etcd/blob/master/Documentation/learning/design-client.md)
