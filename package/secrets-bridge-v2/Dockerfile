FROM ubuntu:16.04
RUN apt-get update && \
    apt-get install -y jq curl nfs-common
COPY storage /usr/bin/
COPY common/* /usr/bin/
ADD https://github.com/rancher/secrets-bridge-v2/releases/download/v0.3.2/secrets-bridge-v2 /usr/bin/secrets-bridge-v2
RUN chmod +x /usr/bin/secrets-bridge-v2
CMD /bin/bash -c '/usr/bin/start.sh storage --save-on-attach --driver-name secrets-bridge-v2 --cattle-access-key ${CATTLE_ACCESS_KEY} --cattle-secret-key ${CATTLE_SECRET_KEY}'
