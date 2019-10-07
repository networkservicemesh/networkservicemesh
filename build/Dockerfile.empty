FROM alpine as runtime
COPY "*" "/bin/"
ARG ENTRY
ENV ENTRY=${ENTRY}
CMD /bin/${ENTRY}