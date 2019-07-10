# Swapper 

Swapper is a simple way to deploy containers, on your existing infrastructure, and swap their version on the fly.


2 kind of swapper exist:
- masters: those who spread the configuration of your cluster
- nodes: those who run the containers

## Installation

Run this command to download the current stable release of Swapper:

```bash
sudo curl -L "https://github.com/SachaMorard/swapper/releases/download/1.0.0/swapper-$(uname -s)-$(uname -m)" -o /usr/local/bin/swapper
```
>To install a different version of Swapper, substitute 1.0.0 with the version of Swapper you want to use.

Apply executable permissions to the binary:

```bash
sudo chmod +x /usr/local/bin/swapper
```

>Note: If the command `swapper` fails after installation, check your path. You can also create a symbolic link to /usr/bin or any other directory in your path. 

For example: 
`sudo ln -s /usr/local/bin/swapper /usr/bin/swapper`


## How to run swapper

### Start Master(s)

First, connect to a server (with swapper installed) and start a master:
```
swapper master start
```

If you want more than one master to get resilient, connect to an other server and start an other master that'll join the first one:
```
swapper master start --join first-master-hostname
```

### Start Node(s)

Then, connect to a new server and start the first node:
```
swapper node start --join first-master-hostname,second-master-hostname
```
As you can see, your node is syncing with the masters and run some containers.
You can start as many nodes as you want

### Deploy your containers

Connect to a master, then create a swapper.yml configuration file to describe what your nodes will do
```
version: '1'

services:
  nginx:
    ports:
      - 80:80
    containers:
      - image: nginx
        tag: 1.16.0
```

Deploy your file.
```bash
swapper deploy -f swapper.yml
```
Instantly, the nodes will understand that they have to rollout new containers.


### To deploy a new version of your containers

Connect to a master and change the swapper.yml file (look at the nginx tag)
```
version: '1'

services:
  nginx:
    ports:
      - 80:80
    containers:
      - image: nginx
        tag: 1.17.0
```

Then, deploy the new conf
```bash
swapper deploy -f swapper.yml
```
You'll see that your node will update without any interruption.

### Dynamic configuration

You can add variables to your swapper.yml with `${}` syntax
```
version: '1'

services:
  nginx:
    ports:
      - 80:80
    containers:
      - image: nginx
        tag: ${TAG}
```
Then, deploy the new conf
```bash
swapper deploy -f swapper.yml --var TAG=1.15.10
```

You can add command to your swapper.yml with `$()` syntax. When a node retrieves the swapper.yml configuration, it replaces $(COMMAND) with the result of the command. 
```
version: '1'

services:
  nginx:
    ports:
      - 80:80
    containers:
      - image: nginx
        tag: 1.17.0
        port: 80
        logging:
          driver: fluentd
          options:
            fluentd-address: $(hostname):24224
        extra_hosts:
          - machine-host:$(ifconfig | grep "inet " | grep -E "broadcast|Bcast" | awk '{print $2}' | tail -n1 | sed "s/adr://g" | sed "s/addr://g")
```


### More?

To know more about the swapper capability, you can inspect [the swapper.yml examples](https://github.com/SachaMorard/swapper/tree/master/doc/swapper.yml.examples)

