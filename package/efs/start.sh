#!/bin/bash

mount --rbind /host/dev /dev
exec "$@"
