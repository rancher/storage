#!/bin/bash

update-rancher-ssl
mount --rbind /host/dev /dev
exec "$@"
