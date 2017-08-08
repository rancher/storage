## Rancher ABS Volume Plugin Driver

### ABS plugin driver is a bash script and invocation commands are:

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

Currently we support below use cases

##### Create command
Input disk name(optional), size(default 5, unit G), disk category(default cloud), snapshotId(optional) and etc, `abs` can create a new disk. [Reference](https://help.aliyun.com/document_detail/25513.html?spm=5176.doc25514.6.866.Ou38fP)
The output is the status, newly created diskId, regionId, zoneId and name.

```
./abs create '{"diskName":"frank-test"}'

# name and size key-value pairs must be provided in json_options and others are optional parameters

# using snapshotId, will create a new disk base the snapshot.

stdout output: {"status":"Success","created":true,"diskId":"d-wz9bk31u5tgf303mxi57","name":"frank-test","regionId":"cn-shenzhen","zoneId":"cn-shenzhen-a"}
```

In processing, `abs` will try to `mkfs.ext4` the disk without snapshotId.

##### Delete command
Input pre-existing disk id(required), `abs` will try to delete the disk. [Reference](https://help.aliyun.com/document_detail/25516.html?spm=5176.doc25513.6.869.Ir4McA)

```
./abs delete '{"created":true,"diskId":"d-wz9bk31u5tgf303mxi57"}'

stdout output: {"status":"Success","message":""}
```

#### Attach command
Input pre-existing disk id(required) and name(required), `abs` will try to attach the disk. [Reference](https://help.aliyun.com/document_detail/25515.html?spm=5176.doc25516.6.868.sw11UL)
The result is that disk(d-wz9bk31u5tgf303mxi57) is attached to the host as block device: /dev/vdb.

```
./abs attach '{"diskId":"d-wz9bk31u5tgf303mxi57", "name": "abc"}'

stdout output: {"status":"Success","device":"/dev/vdb"}
```

In processing, `abs` will create a new directory: `/var/lib/rancher/volumes/rancher-abs/abc-staging`, and mount device path to it.
After attach success, can find the description of disk(d-wz9bk31u5tgf303mxi57) has change to `rancher_abs/dev/vdb`. If change the description, abs can't work for detach.

#### Detach command
Input correct device path(required), in Aliyun ECS, Block Storage always look like: /dev/vdb, /dev/vdc,..., /dev/vdz. `abs` will try to detach the disk. [Reference](https://help.aliyun.com/document_detail/25516.html?spm=5176.doc25515.6.870.QVxTUO)
The result is that disk(/dev/vdb) is detached.

```
./abs detach /dev/vdb

stdout output: {"status":"Success","message":""}
```

If the disk is attached on `abs`, it's easy to detached by `abs`. If use `abs` to detached other disk, just modify the description of disk as `rancher_abs${device path}`.
After detach success, can find the description of Aliyun Block Storage has change to `rancher_detach`.