# neterra-tv-proxy
Docker container with neterra.tv proxy

## Running the docker image

```sh
docker run --name ntv -e USERNAME='your_neterra_user' -e PASSWORD='your_neterra_password' -e HOST='the_host_ip_of_your_server' -p 8889:8889 npenkov/neterra-tv-proxy:1.0
```

## On FireTV use the 

This project copies the functionallity at https://github.com/sgloutnikov/NeterraProxy, but you can run it on your home server.