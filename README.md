# Swapper 

Swapper is a simple solution to run containers, on your existing infrastructure, and swap their version on the fly without any interruption.

Swapper is fast, distributed, and **SIMPLE**!

2 kind of swapper exist:
- masters: those who store and share the configuration files
- nodes: those who run the containers described on the configuration file

![Swapper](doc/swapper.jpg?raw=true "Swapper")

## Installation

Run this command to download the current stable release of Swapper:

```bash
sudo curl -L "https://github.com/SachaMorard/swapper/releases/download/1.0.2/swapper-$(uname -s)-$(uname -m)" -o /usr/local/bin/swapper
```
>To install a different version of Swapper, substitute 1.0.2 with the version of Swapper you want to use.

Apply executable permissions to the binary:

```bash
sudo chmod +x /usr/local/bin/swapper
```

>Note: If the command `swapper` fails after installation, check your path. You can also create a symbolic link to /usr/bin or any other directory in your path. 

For example: 
`sudo ln -s /usr/local/bin/swapper /usr/bin/swapper`


## How to run swapper

[To use Google Cloud Storage as master](https://github.com/SachaMorard/swapper/tree/master/doc/deployWithGCS.md)

### Or you can run masters on your own servers

First, connect to a server (with swapper installed) and start a master:
```bash
swapper master start
```

If you want more than one master to get resilient, connect to an other server and start an other master that'll join the first one:
```bash
swapper master start --join first-master-hostname
```

### Deploy your containers configuration file

Connect to a master, then create a `myapp.yml` configuration file to describe what your nodes will do
```yaml
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
swapper deploy -f myapp.yml
```

### Start Node(s)

Then, connect to a new server and start the first node:
```bash
swapper node start --join first-master-hostname,second-master-hostname --apply myapp.yml
```
As you can see, your node is syncing with the masters and run some containers. You can start as many nodes as you want.
In the future, when you'll deploy a new version of your `myapp.yml` file, the nodes will instantly understand that they have to rollout new containers.


### To deploy a new version of your containers

Connect to a master and change the `myapp.yml` file (look at the nginx tag)
```yaml
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
swapper deploy -f myapp.yml
```
You'll see that your node(s) will update without any interruption.

### Dynamic configuration

You can add variables to your `myapp.yml` with `${}` syntax
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
swapper deploy -f myapp.yml --var TAG=1.15.10
```

You can add command to your `myapp.yml` with `$()` syntax. When a node retrieves the `myapp.yml` configuration, it replaces $(COMMAND) with the result of the command. 
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

To know more about the swapper capability, you can inspect [the yaml configuration file examples](doc/yml-examples)
