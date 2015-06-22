# Bridge proxy

## Goal

Bridge is a convenient way to lauch web applications in Cocaine Cloud without any changes
it the application code for testing purposes.
But it provides a limited support of the platform power right now.
For example, it does not support services, so you have to use a framework for your language to
work with services.

## Configuration

Bridge is configurated by environment variables. The best place for such options is a manifest.
Also the options could be set in Dockerfile. Let's look at an example:

```json
{
    "environment": {
        "slave": "./app.py",  	// path to your application
        "port": "8080",  	  	// port which is listened by your app (default: "8080")
        "startup-timeout": "5"
    },

    "slave": "/usr/bin/bridge"  // path to bridge binary inside a container
}

```

Bridge supports folowing options:
 + *slave* - this a path to your actual application inside container
 + *port* - a port which is listened by your application
 + *startup-timeout* - timeout for pinging your app before accepts load

## Usage

To use Bridge you have to copy its binary inside a container during a build phase.
For example, let's look at a part of Dockerfile that copies the binary:

```
# it copies the binary from a Docker context dir into a container
ADD ./bridge /usr/bin/bridge
```

In manifest `slave` option has to point out to a bridge binary.


## How it works

### Requests

Bridge works like a proxy between cocaine-runtime and your app. It delivers HTTP requests from users
encoded to Cocaine protocol, decodes them and delivers to your web app. Of course, it transfers them back to users.

Bridge starts sending requests to your application after successful connection to your port.

```
COCAINE HTTP REQUEST						   HTTP REQUEST		   COCAINE RESPONSE
==========================================================================================================
[METHOD, URI, VERSION, HEADERS, BODY]	--->   HTTP RESPONSE  ---> [STATUS CODE, HEADERS], [BODY], [BODY]
```

### Logs

Bridge collects `stdout/stderr` of the application and logs both of them by Cocaine logging service.
`stderr` is logged using **ERROR** level, `stdout` using **INFO**.

### Overwatch

Bridge keeps an eye on your application. If the application crashes, Bridge sends an error message about it to Cocaine and
dies too.
If Cocaine decides to kill the application, Bridge sends a SIGTERM to your application, waits *5* seconds for a termination.
If your app is still alive it would be killed by SIGKILL. If bridge dies, you receive a SIGKILL too (via *pdeathsig*).
Before delivering a load your application is being pinged for `startup-timeout` to find out that you are ready to work.
Simple TCP-check is used. An successfully established connection to your port means that it's time to work.


## Build

```make bridge```
