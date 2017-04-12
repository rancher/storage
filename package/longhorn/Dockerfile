FROM ubuntu:16.04
RUN apt-get update && apt-get install -y curl jq
COPY storage common/* longhorn/longhorn /usr/bin/
CMD ["start.sh", "storage", "--driver-name", "longhorn"]
