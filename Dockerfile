FROM   hub.iflytek.com/aiaas/kong-watchdog-base:0.1.1

MAINTAINER sjliu7@iflytek.com
COPY ./webgate-aipaas /webgate/webgate-aipaas
WORKDIR /webgate
STOPSIGNAL  3
