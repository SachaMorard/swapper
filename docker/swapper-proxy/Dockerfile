FROM haproxy:1.9

MAINTAINER sachamorard <sachamorard@gmail.com>

ENV APP_DIR=/app/src
RUN mkdir -p "${APP_DIR}"
WORKDIR "${APP_DIR}"

COPY docker/swapper-proxy/entrypoint.sh /
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
