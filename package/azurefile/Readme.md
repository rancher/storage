## Rancher AzureFile Volume Plugin Driver

Mount a File Share on an Azure Storage Account.

### Requirements

* Azure Storage Account
* The Storage Account Key

If the file share doesn't already exist it will be created by the mount process.

### Limitations
This is a SMB v3 file share using the mount.cifs binary. As such it may not be appropriate for all workloads. See here for a list of things that the Azure File service doesn't support: [Features Not Supported By the Azure File Service](https://docs.microsoft.com/en-us/rest/api/storageservices/fileservices/features-not-supported-by-the-azure-file-service)

### Azure Files plugin driver is a bash script and invocation commands are:

**Mount:**
```
driver  mount  mountpoint  json_options
```

**Unmount:**
```
driver  unmount  mountpoint
```

**Other Functions:**  
create, delete, attach, detach functions don't do anything.  

### Usage
Launch this stack with the Azure Storage Account Name and Key, then define a `version: "2"` style volume entry in your user stack `docker-compose.yml`.

#### Available Driver Options:

* `file_mode` (Default: 0644)
* `dir_mode`  (Default: 0755)
* `uid`       (Default: 0)
* `gid`       (Default: 0)
* `mount_opts` - Comma separated list of additional [`mount.cifs(8)`](https://linux.die.net/man/8/mount.cifs) options.

#### Named `share` Option or Rancher Generated Name

You can specify a `share` name or let Rancher generate one base on volume scope.
See [Rancher Persistent Storage](http://docs.rancher.com/rancher/v1.2/en/rancher-services/storage-service/#storage-service) documentation for more details on scope names.

If you use `share`:
* `share` supports %{environment_name} template substitution.
* `share` must match [a-z0-9\\-] - This is an Azure limitation.

#### Example
```
version: "2"

services:
  test:
    image: busybox
    volumes:
      - test_volume:/data

volumes:
  test_volume:
    driver: rancher-azurefile
    driver_opts:
      share: test-share
      file_mode: "0644"
      dir_mode: "0755"
      uid: "100"
      gid: "101"
      mount_opts: nolock,rw
```
