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
make build
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

## Looking up available files

Under each user, a list of contacts is available.
The button `Lookup Files` will search the contact and verify if any of the
 files published by this contact are accessible to the user.
So if for `User 1` you click on `Lookup Files` to his first contact (User 2
), you will see two entries:

* password.txt - refused - Download
* budget.txt - allowed - Download

For demonstration purposes, the system also allows to try to download a file
 that is refused by the author of the user.
 
Clicking on `Download` for the refused file will add a red entry to the
 blockchain.
Red means that the transaction has been refused by the blockchain.
So the user doesn't get a proof, and cannot get a re-encryption of the key to
 the file.

Clicking on `Download` for the allowed file creates a green transaction in
 the blockchain, which is then used to re-encrypt the symmetric key of the
  file, downloads the file, and decrypts it.

## Looking and modifying the files

Every user has 2 files uploaded by default.
You can get information about the files, delete them, or upload new files.
If you followed the steps under _Looking up available files_, you can see the
 access to the file under `User 2`.
In the column for User 2, click on the `Details` button next to the 
`budget.txt` file.
It will now crawl through the chain and search for accesses to this file and
 list them in the window.
 
You can also remove a file using the `Delete` button next to it.

If you want to create a file, click on `New File` and enter the following
 information:
- Name: chose any name you like
- Content: enter one or more lines of content
- Group: chose either group, where the name represents the IDs of the users
 that will have access: group_2_3 means that User 2 and User 3 will have access
 
## Editing the DARCs

Finally you can change the access rights to your files in the `DARCs` section
 of each user.
The DARCs set up by the system when you start it describe the default access
: `group_2_3` means that user 2 and user 3 have access.
You can change the access the DARC gives by clicking on `Edit`, which will
 pop up the current definition of the DARC.
By clicking on one of the boxes, you can move the access rights of the user
 from the `Available Access` to `Chosen Access` back and forth.
To confirm the new access rights, the `Apply` button will send a transaction
 to the chain that will update the access rights.
 
## Explore the blockchain

At the bottom of the screen, you can see the last four blocks that have been
 created by the blockchain.
Entries in green are transactions that have been accepted.
Entries in red are transactions that have been rejected, e.g., a read-request
 from a user that has not the right to access a certain file.

Using the arrow to the left you can go back in time and see previous blocks.
With the rightmost arrow you can go back to the more recent blocks.
