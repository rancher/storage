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

stdout output: {"status":"Success","options":{}}
```

##### Delete command
driver does nothing

```
./efs delete '{"fsid":"fs-90d12d39","options":{}}'

stdout output: {"status":"Success"}
```

#### Mount command
driver mounts EFS using pre-existing fsid provided by user

```
./efs mount /home/ubuntu/efsMnt '{"fsid":"fs-90d12d39","export":"/test","mntOptions":"ro,vers=4.1"}'

mntOptions key-value pair is optional for mount, but fsid and export are the must

stdout output: {"status":"Success"}
```

The result is that us-west-2b.fs-90d12d39.efs.us-west-2.amazonaws.com://test is mounted to /home/ubuntu/efsMnt.
Here we assume user's EFS file system already has a directory called test created before.

#### Unmount command
driver unmount EFS file system

```
./efs unmount /home/ubuntu/efsMnt

stdout output: {"status":"Success"}
```

### 2. User does not provide fsid

##### Create command
user needs to supplies volume name, driver creates a new file system at AWS EFS site and tags it using name.
The output is the status and newly created fsid etc

```
./efs create '{"creationToken":"anystring","name":"vol1","performanceMode":"maxIO"}'

name and creationToken key-value pairs must be provided in json_options and performanceMode is optional parameter

stdout output: {"status":"Success","options":{"created":true,"fsid":"fs-b59c621c","mountTargetId":"fsmt-b20df41b"}}

the options map from the output will be passed as json_input for delete command
```

##### Delete command
driver deletes created EFS file system

```
./efs delete '{"options":{"created":true,"fsid":"fs-b59c621c","mountTargetId":"fsmt-b20df41b"}}'

the options map is part of output from create command

stdout output: {"status":"Success"}
```

#### Mount command
driver mounts EFS using fsid created at create command phase

```
./efs mount /home/ubuntu/efsMnt '{"fsid":"fs-b59c621c","export":"/","mntOptions":"ro,vers=4.1"}'

mntOptions key-value pair is optional for mount, but fsid and export are the must

stdout output: {"status":"Success"}
```

The result is that us-west-2b.fs-b59c621c.efs.us-west-2.amazonaws.com:/ is mounted to /home/ubuntu/efsMnt

#### Unmount command
driver unmount EFS file system

```
./efs unmount /home/ubuntu/efsMnt

stdout output: {"status":"Success"}
```