FROM ubuntu:16.04
RUN apt-get update && \
    apt-get install -y jq curl nfs-common
COPY storage /usr/bin/
COPY cifs/rancher-cifs common/* /usr/bin/
CMD ["start.sh", "storage", "--driver-name", "rancher-cifs"]
