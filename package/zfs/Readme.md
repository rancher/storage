## Rancher ZFS Volume Plugin Driver

### ZFS plugin driver is a bash script and invocation commands are:

Create:
```
driver  create  json_options
```

Delete:
```
driver  delete  json_options
```

Currently we support 2 major use cases

### 1. User provides existing volumeName

##### Create command
driver does nothing

```
./zfs create '{"volumeName":"foo"}'

stdout output: {"status":"Success","message":""}
```

##### Delete command
driver does nothing

```
./zfs delete '{"volumeName":"foo"}'

stdout output: {"status":"Success","message":""}
```

### 2. User does not provide volumeName

##### Create command
user needs to supplies volume name, size.
The output is the status and newly created volumeName.

```
./zfs create '{"name":"foo","reservation":"5","quota":"10"}'

name, reservation and quota key-value pairs must be provided in json_options and others are optional parameters

stdout output: {"status":"Success","created":true,"volumeName":"foo"}

the options map from the output will be passed in as part of json_input for delete command
```

##### Delete command
driver deletes created ZFS file system

```
./zfs delete '{"created":true,"volumeName":"foo"}'

the options map from create command output is passed in to delete command as part of json_input

stdout output: {"status":"Success","message":""}
```
