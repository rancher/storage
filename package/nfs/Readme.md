## Rancher NFS Volume Plugin Driver

### NFS plugin driver is a bash script and invocation commands are:

Create:
```
driver  create  json_options
```

Delete:
```
driver  delete  json_options
```

Mount:
```
driver  mount  mountpoint  json_options
```

Unmount:
```
driver  unmount  mountpoint
```

Currently we support 2 major use cases

### 1. User provides existing remote share in the form of host:export

##### Create command
driver does nothing

```
./nfs create '{"host":"146.148.46.118","export":"/var/nfs"}'

stdout output: {"status":"Success","options":{}}
```

##### Delete command
driver does nothing

```
./nfs delete '{"host":"146.148.46.118","export":"/var/nfs","options":{}}'

stdout output: {"status":"Success"}
```

#### Mount command
driver mounts NFS using pre-existing remote share provided by user

```
./nfs mount /home/ubuntu/nfsMnt '{"host":"146.148.46.118","export":"/var/nfs","mntOptions":"ro,vers=4.1"}'

mntOptions key-value pair is optional for mount, but host and export are the must

stdout output: {"status":"Success"}
```

The result is that 146.148.46.118:/var/nfs is mounted at /home/ubuntu/nfsMnt


#### Unmount command
driver unmount NFS file system

```
./nfs unmount /home/ubuntu/nfsMnt

stdout output: {"status":"Success"}
```

### 2. User does not provide remote share

Rancher have default host and export environment variables set on the host identifying default NFS remote share.
Driver will use them to create a directory for each volume during create command and delete it at delete command.
For instance:

```
export HOST=146.148.46.118
export EXPORT=/var/nfs
```

##### Create command
user supplies volume name, driver creates a directory under default remote share using volume name.
The output is the status and newly created directory name

```
./nfs create '{"mntDest":"/home/ubuntu/mnt","name":"vol1"}'

name represents volume name and mntDest is the mount point this remote volume(HOST:EXPORT/name) will be mounted.
mntDest is needed for driver to temporarily mount remote share in order to create a subdirectory named "name"

stdout output: {"status": "Success”,"options":{"created":true,"name":"vol1”}}

the options map from the output will be passed as json_input for delete command
```

##### Delete command
driver deletes created directory at create phase

```
./nfs delete '{"mntDest":"/home/ubuntu/mnt","options":{"created":true,"name":"vol1"}}'

the options map is part of output from create command

stdout output: {"status":"Success"}
```

#### Mount command
driver mounts NFS src share using HOST:EXPORT/name created at create command phase

```
./nfs mount /home/ubuntu/nfsMnt '{"name":"vol1","mntOptions":"ro,vers=4.1"}'

mntOptions key-value pair is optional for mount, but name is a must

stdout output: {"status":"Success"}
```

The result is that 146.148.46.118:/var/nfs/name is mounted at /home/ubuntu/nfsMnt

#### Unmount command
driver unmount NFS remote share

```
./nfs unmount /home/ubuntu/nfsMnt

stdout output: {"status":"Success"}
```