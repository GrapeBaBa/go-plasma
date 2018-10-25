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
