## Docker Swarm (= 1.12.x) Dashboard

Firstly, startup swarm dashboard service:
```sh
docker run -it --rm --privileged -p 8443:443 -p 10000-10010:10000-10010 -e INIT_ACCOUNT="admin:badmin" -v /export-docker:/var/lib/docker ghostplant/docker-dashboard
```

Then get access to dashboard:
```sh
firefox https://localhost:8443/
```

For Ubuntu 16.04:
```sh
sudo -i
apt-get install docker.io nginx openssl netcat-openbsd iptables
./dashboard/run  # default url is 'https://localhost:443/', and default account is 'admin:badmin'.
``` 
