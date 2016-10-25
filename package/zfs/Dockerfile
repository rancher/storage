FROM ubuntu:16.04
RUN apt-get update && \
    apt-get install -y jq curl zfsutils-linux
COPY storage /usr/bin/
COPY zfs/rancher-zfs common/* /usr/bin/
CMD ["start.sh", "storage", "--driver-name", "rancher-zfs"]
