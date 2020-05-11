Customize Eagle
===============

This is a configuration document for Eagle components and stays tuned as this evolves.

## Proxy

The following startup parameters are supported for Eagle `Proxy`

| Parameter | Default | Description |
| ------------- | ------------- | ------------- |
| **clientCfg** |
| port | 61007 | EagleClient bt listening port |
| trackers |  | tracker list for EagleClient |
| seeders |  | seeder list for EagleClient |
| rootDirectory | /data/bt/proxy | cache directory of EagleClient |
| limitSize | 100G | cache directory limit size of EagleClient |
| downloadRateLimit | 50M | download rate limiter for EagleClient to serve bt download tasks |
| uploadRateLimit | 50M | upload rate limiter for EagleClient to serve bt upload tasks |
| downloadTimeout | 30 | download timeout for EagleClient to download through bt|
| **proxyCfg** |
| port | 43002 | Proxy daemon listening port |
| verbose | true | enable Proxy debug mode |
| rules | | filtered hosts for using EagleClient |
| certFile | | certificate path of Proxy |
| keyFile | | key file of Proxy |

## Seeder

The following startup parameters are supported for Eagle `Seeder`

| Parameter | Default | Description |
| ------------- | ------------- | ------------- |
| **seederCfg** |
| port | 61008 | Seeder bt listening port |
| origin |  | access address of docker distribution |
| trackers |  | tracker list for Seeder |
| rootDirectory | /data/bt/seeder | cache directory of Seeder |
| limitSize | 1T | cache directory limit size of Seeder |
| downloadTimeout | 30 | download timeout for Seeder to download blob from origin |
| storageBackend | fs | cache storage backend of seeder(only fs supported currently) |
| **daemonCfg** |
| port | 55008 | Seeder daemon listening port |
| verbose | true | enable Seeder debug mode |

## Tracker

Refers to [example_config.yaml](https://github.com/chihaya/chihaya/blob/master/dist/example_config.yaml)