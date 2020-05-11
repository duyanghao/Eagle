# Eagle Quick Start

## Prerequisites

* Docker
* Golang

## Step 1: Build binary

### Build Proxy(EagleClient) and Seeder
 
```bash
$ git clone https://github.com/duyanghao/Eagle.git
$ make build
go mod vendor
make src.build
cd proxy && GO111MODULE=on go build -mod=vendor -v -o .././build/proxy
github.com/duyanghao/eagle/proxy/cmd
github.com/duyanghao/eagle/proxy
cd seeder && GO111MODULE=on go build -mod=vendor -v -o .././build/seeder
github.com/duyanghao/eagle/seeder/cmd
github.com/duyanghao/eagle/seeder
$ ls build
proxy  seeder
```

### Build Tracker

```bash
$ git clone git@github.com:chihaya/chihaya.git
$ cd chihaya
$ go build ./cmd/chihaya    
$ mkdir build && cp -f chihaya build/tracker
```

## Step 2: Deploy proxy

```bash
$ bash hack/start_proxy.sh
```

## Step 3: Deploy seeder

```bash
$ bash hack/start_seeder.sh
``` 

## Step 4: Deploy Tracker

```bash
$ bash hack/start_tracker.sh
```

## Step 5: Configure Docker Daemon

```bash
$ cat << EOF > /etc/systemd/system/docker.service.d/http-proxy.conf
[Service]
Environment="HTTP_PROXY=http://x.x.x.x:43002"
EOF
$ cat << EOF > /etc/systemd/system/docker.service.d/https-proxy.conf
[Service]
Environment="HTTPS_PROXY=http://x.x.x.x:43002"
EOF
```

2. Restart Docker Daemon.

```bash
$ systemctl daemon-reload
$ systemctl restart docker
```

## Step 6: Pull images with Eagle

Through the above steps, we can start to validate if Eagle works as expected.

And you can pull the image as usual, for example:

```bash
$ docker pull nginx:latest
```

## SEE ALSO

- [eagle configuration](../configuration/configuration.md)
