FROM ubuntu:16.04
RUN apt-get update && \
    apt-get install -y jq python2.7 python-pip curl
RUN pip install awscli
COPY storage /usr/bin/
COPY cattle-nfs start.sh /usr/bin/
CMD ["start.sh", "storage", "--driver-name", "cattle-nfs"]
