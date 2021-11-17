FROM ubuntu:18.04
RUN mkdir -p /home
WORKDIR /home

COPY ./bin/ ./
COPY ./conf/ ./conf/

RUN apt update \
&& apt install -y tzdata \
&& rm -f /etc/localtime \
&& ln -s /usr/share/zoneinfo/Asia/Shanghai /etc/localtime 

ENTRYPOINT ["./executor"]
