FROM ubuntu:16.04
RUN apt-get update && \
    apt-get install -y jq python2.7 python-pip curl
RUN pip install --upgrade pip && \
    pip install aliyun-python-sdk-ecs && \
    pip install aliyuncli
COPY storage /usr/bin/
COPY abs/rancher-abs common/* /usr/bin/
RUN chmod a+x /usr/bin/rancher-abs
CMD ["start.sh", "storage", "--driver-name", "rancher-abs"]
