FROM ubuntu:18.04

RUN useradd -ms /bin/bash appuser 
RUN mkdir /usr/src/app

COPY main-bad.sh /usr/src/app
COPY main-good.sh /usr/src/app
COPY .bashrc /usr/src/app/
COPY docker-entrypoint.sh /usr/local/bin/

RUN chmod o+rwx -R /usr/src/app \
    && chmod o+rwx /usr/local/bin/docker-entrypoint.sh

ENV HOME=/usr/src/app
ENV APP_HOME=/usr/src/app

USER appuser
WORKDIR /usr/src/app

ENTRYPOINT ["docker-entrypoint.sh"]
