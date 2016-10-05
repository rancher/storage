## Rancher EBS Volume Plugin Driver

### EBS plugin driver is a bash script and invocation commands are:

Create:
```
driver  create  json_options
```

Delete:
```
driver  delete  json_options
```

Attach:
```
driver  attach  mountpoint  json_options
```

Detach:
```
driver  detach  mountpoint
```

Currently we support 2 major use cases

### 1. User provides existing volumeID

##### Create command
driver does nothing

```
./ebs create '{"volumeID":"vol-870fdb33"}'

stdout output: {"status":"Success","message":""}
```

##### Delete command
driver does nothing

```
./ebs delete '{"volumeID":"vol-870fdb33"}'

stdout output: {"status":"Success","message":""}
```

#### Attach command
driver attaches EBS using pre-existing volumeID provided by user

```
./ebs attach '{"volumeID":"vol-870fdb33"}'

stdout output: {"status":"Success","device":"/dev/xvdf"}
```

The result is that vol-870fdb33 is attached to the host as block device: /dev/xvdf

#### Detach command
driver detach EBS block device

```
./ebs detach /dev/xvdf

stdout output: {"status":"Success","message":""}
```

### 2. User does not provide volumeID

##### Create command
user needs to supplies volume name, size and optionally EBS volume type and iops etc.
The output is the status and newly created volumeID.

```
./ebs create '{"name":"vol1","size":"5","volumeType":"io1","iops":"200"}'

name and size key-value pairs must be provided in json_options and others are optional parameters

stdout output: {"status":"Success","created":true,"volumeID":"vol-870fdb33"}

the options map from the output will be passed in as part of json_input for delete command
```

##### Delete command
driver deletes created EBS file system

```
./ebs delete '{"created":true,"volumeID":"vol-870fdb33"}'

the options map from create command output is passed in to delete command as part of json_input

stdout output: {"status":"Success","message":""}
```

#### Attach command
driver attaches EBS using pre-existing volumeID provided by user

```
./ebs attach '{"volumeID":"vol-870fdb33"}'

stdout output: {"status":"Success","device":"/dev/xvdf"}
```

The result is that vol-870fdb33 is attached to the host as block device: /dev/xvdf

#### Detach command
driver detach EBS block device

```
./ebs detach /dev/xvdf

stdout output: {"status":"Success","message":""}
```