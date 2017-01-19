## Rancher EFS Volume Plugin Driver

### EFS plugin driver is a bash script and invocation commands are:

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

### 1. User provides existing fsid

##### Create command
driver does nothing

```
./efs create '{"fsid":"fs-90d12d39"}'

stdout output: {"status":"Success","message":""}
```

##### Delete command
driver deletes the file system

```
./efs delete '{"fsid":"fs-90d12d39"}

stdout output: {"status":"Success","message":""}
```

#### Mount command
driver mounts EFS using pre-existing fsid provided by user

```
./efs mount /home/ubuntu/efsMnt '{"fsid":"fs-90d12d39","export":"/test","mntOptions":"ro,vers=4.1"}'

mntOptions key-value pair is optional for mount, but fsid and export are the must

stdout output: {"status":"Success","message":""}
```

The result is that us-west-2b.fs-90d12d39.efs.us-west-2.amazonaws.com://test is mounted to /home/ubuntu/efsMnt.
Here we assume user's EFS file system already has a directory called test created before.

#### Unmount command
driver unmount EFS file system

```
./efs unmount /home/ubuntu/efsMnt

stdout output: {"status":"Success","message":""}
```

### 2. User does not provide fsid

##### Create command
User needs to supplies volume name, and an optional security group Id/name which we can use to setup the mount. If the 
security group name/id is not provided, then we create a new security group with port 2049 open to everyone.

Note: We use the security group name _tag_ and *NOT* the group name.

The driver creates a new file system at AWS EFS site and mounts it in the subnet of the EC2 instance where it is running
and tags it using name.

The output is the status and newly created fsid etc

```
./efs create '{"creationToken":"anystring","name":"vol1","performanceMode":"maxIO",mountTargetSGName": "target.efs.mount.security.group"}'

name and creationToken key-value pairs must be provided in json_options and performanceMode & mountTargetSGName are optional parameters.

If existing security group Id/Name is passed then we set the _useExistingSecGrp_ to _true_, so that we don't delete the security group
while deleting the volume.
```
{"status":"Success","options":{created": "true", "fsid": "fs-xxxxxxxx”, "mountTargetSGID": "sg-xxxxxxxx”, ”mountTargetSGName":"target.efs.mount.security.group", "name": "efsPubPrivTest03_EFS", "useExistingSecGrp": "true"}}
```

the options map from the output will be passed in as part of json_input for delete command
```

##### Delete command
driver deletes created EFS file system along with the security group if existing security group was not provided to the create command.

```
./efs delete '{created": "true", "fsid": "fs-xxxxxxxx”, "mountTargetSGID": "sg-xxxxxxxx”, ”mountTargetSGName":"target.efs.mount.security.group", "name": "efsPubPrivTest03_EFS", "useExistingSecGrp": "true"}'

the options map from create command output is passed in to delete command as part of json_input

stdout output: {"status":"Success","message":""}
```

#### Mount command
driver mounts EFS using fsid created at create command phase in the availability zone of the EC2 instance where the driver is running.

We check if mount point exists for this EFS volume in the EC2 instance subnet, if not we create a new mount point in the EC2 instance's subnet.

```
./efs mount /home/ubuntu/efsMnt '{"fsid":"fs-b59c621c","export":"/","mntOptions":"ro,vers=4.1"}'

mntOptions key-value pair is optional for mount, but fsid and export are the must

stdout output: {"status":"Success","message":""}
```

The result is that <availability zone>.fs-b59c621c.efs.us-west-2.amazonaws.com:/ is mounted to /home/ubuntu/efsMnt

#### Unmount command
driver unmount EFS file system

```
./efs unmount /home/ubuntu/efsMnt

stdout output: {"status":"Success","message":""}
```