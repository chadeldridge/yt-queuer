# yt-queuer
Lightweight service and addon to enable queueing and playback of youtube videos on a remote device.

# Current State
## Build
The Makefile build targets will build binary in 'bin/' then copy the binary and any other needed files to 'pkg/'.

Using the default Makefile build target:
```
make build
```
Will build ytqueuer for the architecture you are currently on.

OR

You can specify an architecture.
```
make build GOARCH=amd64
```

OR

Use a predefined build target for a tested architecture.
```
make build-arm64
```
Will build for arm64.

## Deploy
Copy pkg/ to the host you want to run it on. Example:
```
rsync -avz pkg/ user@host:ytqueuer
```

## Run
From the host, run ytqueuer via whatever method you prefer.
```
cd ytqueuer
nohup ytqueuer &
```
ytqueuer will listen on all interfaces.

## Access
From your preferred browser on the host go to:
```
http://localhost:8080
```

## Control
From a browser on any other device which can reach an interface on the you can use the following GET requests.
<hostIP> is a reachable IP on the host where ytqueuer is running.
<videoID> is the id of the youtube video you want to add to the queue.
?start=seconds is always optional. The video will start from the beginning if ommited or set to 0.

### Add
Add a video to the end of the queue.
```
http://<hostIP>:8080/queue/add/<videoID>[?start=seconds]
```

Examples:
```
http://192.168.1.10:8080/queue/add/M7lc1UVf-VE
```
Will add the youtbue api example video to the queue and will start playing from the beginning when it is played.

```
http://192.168.1.10:8080/queue/add/M7lc1UVf-VE?start=120
```
Will add the youtbue api example video to the queue and will start playing from the 2 minute mark when it is played.

### Play Next
Add a video to the beginning of the queue so it will be the next video in queue.
```
http://<hostIP>:8080/queue/playnext/<videoID>
```
