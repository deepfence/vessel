# Vessel <img src="http://www.pngmart.com/files/12/Vessel-PNG-Transparent-Picture.png" width="40" height="40" alt=":vessel:" class="emoji" title=":vessel:"/>

Vessel is the Go based utility that autodetects underlying Container Runtime in Kubernetes.

## Containerd namespace

Vessel scans every available namespaces from containerd.
Current behavior consists of issuing a command to every namespace until one succeeds.
