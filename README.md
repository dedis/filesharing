# Calypso-enabled Filesharing

This directory contains a simple file-sharing app using Calypso.
When starting up, the system creates three users.
The first user shares a file with the second, and the second with the third.
When trying to access a file that is not allowed, the chain will refuse to do so.

## Start it

Before you start it, you need to have Docker installed:

https://www.docker.com/products/docker-desktop

As everything is already compiled and put in a docker image, you can simply start it with:

```bash
docker-compose up
```

This will start to download the docker images, and then run them.
Once they are started, you can launch your webbrowser to connect to the following address:

http://localhost:8080

## Update it

If there is a new version of the software, you need to update the docker images:

```bash
docker-compose pull
```

## Compile it

If you do changes, you can compile it again using:

```bash
make docker
```

Afterwards you have to re-start the docker containers.

## What it does

When the browser connects to the webapp the first time, it does set up the following:

- creates a new testing blockchain
- creates a new LTS
- creates three users: user1, user2, user3
- creates sharing groups: user12, user23
- stores two files on the blockchain:
  - password.txt that is shared between user1 and user2
  - cats.png that is shared between user2 and user3

Once this is done, it allows you to check what is happening and to create groups and files yourself.

# What you can do with it

Here is a short walkthrough of the things you can do with this demo.

## 
