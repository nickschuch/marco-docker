# Marco - Docker

Docker backend for https://github.com/nickschuch/marco

## Usage

```bash
$ marco-docker --marco=http://localhost:81
```

NOTE: We assume the Marco daemon is already running.

## Docker

The following will setup Marco + Docker backend pushes.

```bash
$ docker run -d \
             --name=marco \
             -p 0.0.0.0:80:80 nickschuch/marco
$ docker run -d \
             --link marco:marco \
             -e "MARCO_ECS_URL=http://marco:81" \
             -v /var/run/docker.sock:/var/run/docker.sock nickschuch/marco-ecs
```

