## Docker Prerequisites
### Docker CE (Community Edition)

|OS| System Prerequisites | Installation
|:--:|:--|:--:|
|CentOS|7.x (64-bit)| [Download](https://store.docker.com/editions/community/docker-ce-server-centos)  ([Instruction](https://docs.docker.com/install/linux/docker-ce/centos/)) |
|Debian|Buster 10, Stretch 9, Jessie 8.0, Wheezy 7.7 (64-bit)| [Download](https://store.docker.com/editions/community/docker-ce-server-debian)  ([Instruction](https://docs.docker.com/install/linux/docker-ce/debian/)) |
|Fedora|Fedora 28, Fedora 27, Fedora 26 (64-bit)| [Download](https://store.docker.com/editions/community/docker-ce-server-fedora) ([Instruction](https://docs.docker.com/install/linux/docker-ce/fedora/)) |
|Ubuntu|Bionic 18.04 (LTS), Xenial 16.04 (LTS), Trusty 14.04 (LTS)| [Download](https://store.docker.com/editions/community/docker-ce-server-ubuntu) ([Instruction](https://docs.docker.com/install/linux/docker-ce/ubuntu/)) |
|OSX|El Capitan 10.11 or above| [Download](https://store.docker.com/editions/community/docker-ce-desktop-mac) ([Instruction](https://docs.docker.com/docker-for-mac)) |
|MS|Windows 10 Pro, Enterprise or Education(64-bit)| [Download](https://store.docker.com/editions/community/docker-ce-desktop-windows) ([Instruction](https://docs.docker.com/docker-for-windows/)) |
|Others | See  Docker website for details |  [Download](https://store.docker.com/search?type=edition&offering=community) ([Instruction](https://docs.docker.com/install))

### Python
[Python 2.7](https://www.python.org/downloads/release/python-latest) or higher is required for docker image. Verify that python version has been installed in the system:
```
$ python -V
Python 2.7.10
```

## Plamsa Docker Image
#### Starting Docker
##### Download Plamsa Docker Image
```
$ docker pull wolkinc/go-plasma
```
##### Deploying Plamsa Docker Container
```
$ docker run --name=plasma -dit -p 8505:8505 -p 30303:30303 -p 30303:30303/udp wolkinc/go-plasma /tmp/plasmachain/ 8505
```
##### Plamsa Port Mapping

| Ports | Descriptions |
|--|:--|
| 8505:8505 | <RPC_system_port>:<RPC_container_port> |
| 30303:30303 | <Network_Listening_system_TCP_port>:<Network_Listening_container_TCP_port> |
| 30303:30303/udp | <Network_Listening_system_UDP_port>:<Network_Listening_container_UDP_port> |

_*Note_: This command also automatically starts the Plamsa server

#### Interacting with Docker
Once the Docker image and containers are deployed, Plamsa server will run inside the Docker container. Using the following cmds to interact with Plasma server:

##### Attach to the docker container/console
```
$ docker exec -it `docker ps -q -l` /bin/bash
```
##### Exit the Docker container/console
```
$ ctrl + d (or simply type exit and Enter)
```
##### Check Plasma Status outside of the Docker container
```
$ docker exec `docker ps -a | grep go-plasma | awk '{print$1}'` ps aux | grep "plasma --datadir"
```
##### Clean/delete all containers
```
$ docker stop `docker ps -a -q`; docker rm `docker ps -a -q`
```
##### Clean/delete all the images (once the container/s have been deleted)
```
$ docker rmi `docker images -q`
```
