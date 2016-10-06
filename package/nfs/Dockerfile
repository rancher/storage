FROM ubuntu:16.04
RUN apt-get update && \
    apt-get install -y jq curl
COPY storage /usr/bin/
COPY nfs/rancher-nfs common/* /usr/bin/
CMD ["start.sh", "storage", "--driver-name", "rancher-nfs"]
