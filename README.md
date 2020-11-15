# Consul distributed lock library 

It's build for using consul distributed lock easily.

## Prerequisites

You need to install consul.
You can install consul by using docker for test.
```shell script
docker run -d -p 8500:8500/tcp consul agent -server -ui -bootstrap-expect=1 -client=0.0.0.0
```

## Usage
see test.
