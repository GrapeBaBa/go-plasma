# Install Docker CE (Community Edition)
https://www.docker.com/community-edition#/download

### CentOS:
  - Installation instructions: https://docs.docker.com/engine/installation/linux/docker-ce/centos/

### Mac:
  - Installation instructions: https://store.docker.com/editions/community/docker-ce-desktop-mac

### Others:
  - https://www.docker.com/community-edition#/download
  
# System Prerequisites

|OS| Prerequisite |
|--|:--|
|CentOS|7.x (64-bit)|
|RedHat|RHEL 7.x (64-bit)|
|Debian|Stretch, Jessie 8.0, Wheezy 7.7 (64-bit)|
|Fedora|Fedora 25, Fedora 24 (64-bit)|
|Ubuntu|Zesty 17.04 (LTS),Yakkety 16.10, Xenial 16.04 (LTS),Trusty 14.04 (LTS)|
|OSX|Yosemite 10.11 or above|
|MS|Windows 10 Professional or Enterprise (64-bit)|

### Before downloading the docker image, please install Python27 or higher on your system:
  - https://www.python.org/downloads/release/python-latest

### Once installed, verify the python version:
      $ python -V
      Python 2.7.10
      
## Downloading and deploying Docker

## Downloading and deploying Docker

### PLASMA
#### Download PLASMA Docker Image
```
docker pull wolkinc/go-plasma
```

#### Deploying PLASMA Docker Container
_Note: This command also automatically starts the PLASMA server_
```
docker run --name=plasma -dit -p 5001:5000 -p 5001:5000 -p 8505:8505 -p 30303:30303 -p 30303:30303/udp wolkinc/go-plasma /Users/plasma/qdata/dd 8505
```

## Verify PLASMA process

Once the Docker image and containers are deployed following above instructions, it will start the PLASMA process inside the Docker container. To verify if PLASMA is running:

##### Attach to the docker shell/console
    $ docker exec -it `docker ps -q -l` /bin/bash

##### From outside the Docker container
    $ docker exec `docker ps -a | grep go-plasma | awk '{print$1}'` ps aux | grep "plasma --datadir"
    
##### From within the Docker container
    $ ps aux | grep "plasma --datadir"
    
### PLASMA Port Mapping:

| Ports | Descriptions |
|--|--|
| 5001:5000 | <Syslog_system_port>:<Syslog_container_port> |
| 8505:8505 | <RPC_system_port>:<RPC_container_port> |
| 30303:30303 | <Network_Listening_system_TCP_port>:<Network_Listening_container_TCP_port> |
| 30303:30303/udp | <Network_Listening_system_UDP_port>:<Network_Listening_container_UDP_port> |
