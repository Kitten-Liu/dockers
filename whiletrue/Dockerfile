ARG BASEIMG=debian:bookworm
FROM $BASEIMG
WORKDIR /
COPY docker-entrypoint.sh . 
RUN chmod +x /docker-entrypoint.sh
CMD ["./docker-entrypoint.sh"]
