FROM alpine:3.2
COPY bin/blobstore /bin/blobstore
COPY ./public ./public
VOLUME /var/state
ENTRYPOINT ["/bin/blobstore"]
