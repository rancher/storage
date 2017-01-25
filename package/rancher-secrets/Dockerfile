FROM ubuntu:16.04
RUN apt-get update && \
    apt-get install -y jq curl nfs-common
COPY storage /usr/bin/
ADD https://github.com/rancher/secrets-flexvol/releases/download/v0.0.2/secrets-flexvol /usr/bin/rancher-secrets
COPY common/* /usr/bin/
CMD ["start.sh", "storage", "--driver-name", "rancher-secrets"]
