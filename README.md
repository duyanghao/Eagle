eagle
=====

# OverView

eagle is a lightweight and intelligent p2p based docker image distribution system.

<img src="https://github.com/duyanghao/eagle/blob/master/docs/images/logo.png" width=200px/>

# Features

* Non-invasive: Eagle can seamlessly support docker for distributing images. 
* High-availability: No component is a single point of failure.
* Pluggable storage options. Eagle plugs into reliable blob storage options, like S3 or local FileSystem. The storage interface is simple and new options are easy to add.
* Peer optimal arithmetic: Eagle supports peer optimal arithmetic to improve performance and save cross-IDC bandwidth.  
* I/O speed limit: Many downloading tools(wget/curl) only have rate limit for the current download task, but eagle also provides rate limit for the entire host.
* Strong consistency: Eagle can guarantee that all downloaded files must be consistent even if users do not provide any check code(MD5).
* Lightweight: Eagle consists of only several necessary components, which makes it understandable, maintainable and easy-to-use.

# Architecture

The principle of eagle is quite simple and can be illustrated as follows:

![](images/eagle_arch.png)

# Comparison With Other Projects

## [Dragonfly from Alibaba](https://github.com/dragonflyoss/Dragonfly)

Dragonfly cluster has one or a few "supernodes" that coordinates transfer of every 4MB chunk of data in the cluster.

While the supernode would be able to make optimal decisions, the throughput of the whole cluster is limited by the processing power of one or a few hosts, and the performance would degrade linearly as either blob size or cluster size increases.

Eagle's tracker only helps orchestrate the connection graph, and leaves negotiation of actual data transfer to individual peers, so Eagle scales better with large blobs. On top of that, Eagle is HA. 

## [kraken from uber](https://github.com/uber/kraken)

[Kraken] uses several components, such as `Agent`, `Origin`, `Tracker`, `Proxy` and `Build-Index`, combined with its own designed driver protocol to build a p2p based docker distribution system.      
 
Eagle uses almost the same components with [kraken](https://github.com/uber/kraken), but it is more compact as it uses [BitTorrent protocol](http://bittorrent.org/beps/bep_0003.html) underlayer and drops some unessential components.   

## Refs

* [Dragonfly](https://github.com/dragonflyoss/Dragonfly)
* [kraken](https://github.com/uber/kraken)
* [FID: A Faster Image Distribution System for Docker Platform](https://ieeexplore.ieee.org/stamp/stamp.jsp?arnumber=8064123)
* [The BitTorrent Protocol Specification](http://bittorrent.org/beps/bep_0003.html)