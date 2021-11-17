FROM ubuntu:18.04
ARG ENV
RUN mkdir -p /home
RUN mkdir /home/data
WORKDIR /home

COPY ./xdata . 
COPY ./xdata-cli .
COPY ./conf ./conf

RUN apt update \
&& apt install -y tzdata \
&& rm -f /etc/localtime \
&& ln -s /usr/share/zoneinfo/Asia/Shanghai /etc/localtime 

ENTRYPOINT ["./xdata"]
