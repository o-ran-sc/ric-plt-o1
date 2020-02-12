#   Copyright (c) 2019 AT&T Intellectual Property.
#   Copyright (c) 2019 Nokia.
#
#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at
#
#       http://www.apache.org/licenses/LICENSE-2.0
#
#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.

#----------------------------------------------------------
FROM nexus3.o-ran-sc.org:10004/bldr-ubuntu18-c-go:4-u18.04-nng AS o2mediator-build

RUN apt-get update -y && apt-get install -y jq \
      git \
      cmake \
      build-essential \
      vim \
      supervisor \
      libpcre3-dev \
      pkg-config \
      libavl-dev \
      libev-dev \
      libprotobuf-c-dev \
      protobuf-c-compiler \
      libssh-dev \
      libssl-dev \
      swig \
      iputils-ping \
      python-dev

ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/go"

# ======================================================================
# First make the netconf sysrepo stuff
# add netconf user
RUN \
      adduser --system netconf && \
      echo "netconf:netconf" | chpasswd

# generate ssh keys for netconf user
RUN \
      mkdir -p /home/netconf/.ssh && \
      ssh-keygen -A && \
      ssh-keygen -t dsa -P '' -f /home/netconf/.ssh/id_dsa && \
      cat /home/netconf/.ssh/id_dsa.pub > /home/netconf/.ssh/authorized_keys

# use /opt/dev as working directory
RUN mkdir /opt/dev
WORKDIR /opt/dev

# libyang
RUN \
      cd /opt/dev && \
      git clone https://github.com/CESNET/libyang.git && \
      cd libyang && mkdir build && cd build && \
      cmake -DCMAKE_BUILD_TYPE:String="Release" -DENABLE_BUILD_TESTS=OFF .. && \
      make -j2 && \
      make install && \
      ldconfig

# sysrepo
RUN \
      cd /opt/dev && \
      git clone https://github.com/sysrepo/sysrepo.git && \
      cd sysrepo && mkdir build && cd build && \
      cmake -DCMAKE_BUILD_TYPE:String="Release" -DSR_RPC_CB_TIMEOUT=30000 -DENABLE_TESTS=OFF -DREPOSITORY_LOC:PATH=/etc/sysrepo .. && \
      make -j2 && \
      make install && make sr_clean && \
      ldconfig

# libnetconf2
RUN \
      cd /opt/dev && \
      git clone https://github.com/CESNET/libnetconf2.git && \
      cd libnetconf2 && mkdir build && cd build && \
      cmake -DCMAKE_BUILD_TYPE:String="Release" -DENABLE_BUILD_TESTS=OFF .. && \
      make -j2 && \
      make install && \
      ldconfig

# netopeer2
RUN \
      cd /opt/dev && \
      git clone https://github.com/CESNET/Netopeer2.git && \
      cd Netopeer2/server && mkdir build && cd build && \
      cmake -DCMAKE_BUILD_TYPE:String="Release" .. && \
      make -j2 && \
      make install && \
      cd ../../cli && mkdir build && cd build && \
      cmake -DCMAKE_BUILD_TYPE:String="Release" .. && \
      make -j2 && \
      make install
      
# ======================================================================

# RMR
ARG RMRVERSION=1.11.0
ARG RMRLIBURL=https://packagecloud.io/o-ran-sc/staging/packages/debian/stretch/rmr_${RMRVERSION}_amd64.deb/download.deb
ARG RMRDEVURL=https://packagecloud.io/o-ran-sc/staging/packages/debian/stretch/rmr-dev_${RMRVERSION}_amd64.deb/download.deb

RUN wget --content-disposition ${RMRLIBURL} && dpkg -i rmr_${RMRVERSION}_amd64.deb
RUN wget --content-disposition ${RMRDEVURL} && dpkg -i rmr-dev_${RMRVERSION}_amd64.deb
RUN rm -f rmr_${RMRVERSION}_amd64.deb rmr-dev_${RMRVERSION}_amd64.deb
RUN ldconfig

# Swagger
RUN mkdir -p /go/bin
RUN cd /go/bin \
    && wget --quiet https://github.com/go-swagger/go-swagger/releases/download/v0.19.0/swagger_linux_amd64 \
    && mv swagger_linux_amd64 swagger \
    && chmod +x swagger

RUN mkdir -p /go/src/ws
WORKDIR "/go/src/ws/agent"

# Module prepare (if go.mod/go.sum updated)
COPY agent /go/src/ws
RUN GO111MODULE=on go mod download

RUN mkdir -p api \
    && mkdir -p pkg \
    && git clone "https://gerrit.o-ran-sc.org/r/ric-plt/appmgr" \
    && cp appmgr/api/appmgr_rest_api.yaml api/ \
    && rm -rf appmgr
    
# build and test
COPY . /go/src/ws

# generate swagger client
RUN /go/bin/swagger generate client -f api/appmgr_rest_api.yaml -t pkg/ -m appmgrmodel -c appmgrclient
# build the o1agent
RUN GO111MODULE=on GO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o o1agent cmd/o1agent.go

COPY . /go/src/ws

# make the data model based on the ric yang model
RUN /usr/local/bin/sysrepoctl -i /go/src/ws/agent/yang/o-ran-sc-ric-xapp-desc-v1.yang
RUN /usr/local/bin/sysrepoctl -i /go/src/ws/agent/yang/o-ran-sc-ric-ueec-config-v1.yang

CMD ["/bin/bash"]

#----------------------------------------------------------
FROM ubuntu:18.04 as o1mediator

RUN apt-get update -y && apt-get install -y jq \
      net-tools \
      tcpdump \
      netcat \
      keychain \
      nano \
      supervisor \
      openssl \
      python-pip \
      libpcre3-dev \
      pkg-config \
      libavl-dev \
      libev-dev \
      libprotobuf-c-dev \
      protobuf-c-compiler \
      libssh-dev \
      libssl-dev \
      swig \
      python-dev \
      && pip install supervisor-stdout \
      && pip install psutil \
      && apt-get clean

RUN rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# add netconf user
RUN \
      adduser --system netconf && \
      echo "netconf:netconf" | chpasswd

# generate ssh keys for netconf user
RUN \
      mkdir -p /home/netconf/.ssh && \
      ssh-keygen -A && \
      ssh-keygen -t dsa -P '' -f /home/netconf/.ssh/id_dsa && \
      cat /home/netconf/.ssh/id_dsa.pub > /home/netconf/.ssh/authorized_keys

# copy the supervisor config
ARG CONFIGDIR=/etc/supervisor
RUN mkdir -p ${CONFIGDIR}
COPY config/supervisord.conf ${CONFIGDIR}/supervisord.conf
    
# libraries and binaries & config
COPY --from=o2mediator-build /usr/local/share/ /usr/local/share/
COPY --from=o2mediator-build /usr/local/etc/ /usr/local/etc/
COPY --from=o2mediator-build /usr/local/bin/ /usr/local/bin/
COPY --from=o2mediator-build /usr/local/lib/ /usr/local/lib/
RUN ldconfig

# copy yang models with data
COPY --from=o2mediator-build /etc/sysrepo /etc/sysrepo

COPY --from=o2mediator-build /go/src/ws/agent/o1agent /usr/local/bin
COPY --from=o2mediator-build /go/src/ws/manager/src/process-state.py /usr/local/bin
RUN mkdir -p /etc/o1agent
COPY --from=o2mediator-build /go/src/ws/agent/config/* /etc/o1agent/

# ports available outside 8080 for mediator and 9001 supervise http control interrface
# port 830 for netconf client ssh session
# port 3000 for process-event handler web server
EXPOSE 9001 830 8080 3000

CMD ["/usr/bin/supervisord"]
