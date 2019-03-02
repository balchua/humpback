FROM ubuntu:18.04

RUN mkdir /usr/src/app

COPY main-bad.sh /usr/src/app
COPY main-good.sh /usr/src/app
COPY .bashrc /usr/src/app/
COPY docker-entrypoint.sh /usr/local/bin/

RUN chmod 777 -R /usr/src/app \
    && chmod 777 /usr/local/bin/docker-entrypoint.sh

ENV HOME=/usr/src/app
ENV APP_HOME=/usr/src/app

WORKDIR /usr/src/app

ENTRYPOINT ["docker-entrypoint.sh"]
