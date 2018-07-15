From inside the container in a Pod you can get the inode for the network namespace:

```bash
ls -l /proc/self/ns/net
lrwxrwxrwx 1 root root 0 Jun 21 21:18 /proc/self/ns/net -> net:[4026532246]
```
in the above example the inode is the 4026532246 in net:[4026532246]


From the NSM you can look up the network namespace in /var/run/netns (which will need to be mounted into the NSM)
and find a file with the same inode for each namespace.   If you want to try this by hand, you can use ```ls -i```
in /var/run/netns

Note: For some versions of docker they put this directory in /var/run/docker/netns

[Example of how to get netns inode from inside the container written in Go without privilege](https://github.com/edwarnicke/goscratch/blob/master/netnsinode/main.go)

