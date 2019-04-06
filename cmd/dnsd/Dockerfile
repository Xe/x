ARG X_VERSION

FROM xena/xperimental:$X_VERSION as build
FROM xena/alpine

ENV PORT 53
ENV FORWARD_SERVER 1.1.1.1:53
EXPOSE 53/udp

COPY --from=build /usr/local/bin/dnsd /usr/local/bin/dnsd
CMD /usr/local/bin/dnsd
