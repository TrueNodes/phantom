# Dockerfile for Phantom Masternode Backend
# Phantom Hosting - 2020-07-26

FROM photon:3.0-20200721
LABEL Name="Dockerized Phantom Nodes"
LABEL Publisher="GOSSIP Blockchain"

WORKDIR /root/phantom/

ADD https://github.com/GOSSIP-Blockchain/phantom/releases/download/v0.0.5/phantom-linux-amd64 /usr/local/bin/
ADD https://github.com/GOSSIP-Blockchain/phantom/raw/master/configs/nort.json /root/phantom/
RUN chmod a+x /usr/local/bin/phantom-linux-amd64
RUN mkdir conf

ENTRYPOINT ["phantom-linux-amd64", "-broadcast_listen", "-coin_conf=/root/phantom/nort.json", "-masternode_conf=/root/phantom/conf/masternode.txt"]
