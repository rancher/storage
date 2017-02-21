FROM ubuntu:16.04
RUN apt-get update && \
    apt-get install -y jq curl nfs-common
COPY storage /usr/bin/
ADD https://github.com/rancher/secrets-flexvol/releases/download/v0.0.5/secrets-flexvol /usr/bin/rancher-secrets
RUN chmod +x /usr/bin/rancher-secrets
COPY common/* /usr/bin/
CMD /bin/bash -c '/usr/bin/start.sh storage --driver-name rancher-secrets --cattle-access-key ${CATTLE_ENVIRONMENT_ACCESS_KEY} --cattle-secret-key ${CATTLE_ENVIRONMENT_SECRET_KEY}'
