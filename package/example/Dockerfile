FROM ubuntu:16.04
ENV SSL_SCRIPT_COMMIT 98660ada3d800f653fc1f105771b5173f9d1a019
RUN apt-get update && \
    apt-get install -y curl && \
    curl -sLf https://raw.githubusercontent.com/rancher/rancher/${SSL_SCRIPT_COMMIT}/server/bin/update-rancher-ssl > /usr/bin/update-rancher-ssl && \
    chmod +x /usr/bin/update-rancher-ssl
COPY storage /usr/bin/
COPY common/common.sh example/rancher-loop common/start.sh /usr/bin/
CMD ["start.sh", "storage", "--driver-name", "rancher-loop"]
