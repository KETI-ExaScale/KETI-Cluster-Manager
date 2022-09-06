FROM ubuntu:16.04
ADD cluster-manager /cluster-manager
ENTRYPOINT ["/cluster-manager"]
