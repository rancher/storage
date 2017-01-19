FROM ubuntu:16.04
RUN apt-get update && \
    apt-get install -y jq curl cifs-utils python-pip libssl-dev libffi-dev && \
    pip install azure-storage && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*
COPY storage /usr/bin/
COPY azurefile/rancher-azurefile azurefile/azurefile-share.py common/* /usr/bin/
CMD ["start.sh", "storage", "--driver-name", "rancher-azurefile"]
