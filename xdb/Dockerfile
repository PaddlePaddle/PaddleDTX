FROM ubuntu:18.04
ENV DEBIAN_FRONTEND noninteractive
RUN mkdir -p /home
RUN mkdir /home/data
WORKDIR /home

COPY ./xdb . 
COPY ./xdb-cli .
COPY ./conf ./conf

RUN apt-get update \
&& apt-get install -y tzdata \
&& rm -f /etc/localtime \
&& ln -s /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
&& dpkg-reconfigure -f noninteractive tzdata

ENTRYPOINT ["./xdb"]
