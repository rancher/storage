FROM ubuntu:16.04
RUN apt-get update && \
    apt-get install -y jq python2.7 python-pip curl nfs-common dnsutils
RUN pip install awscli
COPY storage /usr/bin/
COPY efs/rancher-efs common/* /usr/bin/
CMD ["start.sh", "storage", "--driver-name", "rancher-efs"]
