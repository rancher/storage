FROM ubuntu:16.04
RUN apt-get update && \
    apt-get install -y jq python2.7 python-pip curl nvme-cli
RUN pip install awscli
COPY storage /usr/bin/
COPY ebs/rancher-ebs common/* /usr/bin/
CMD ["start.sh", "storage", "--driver-name", "rancher-ebs"]
