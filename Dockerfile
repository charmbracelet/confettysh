FROM scratch
COPY confettysh /usr/local/bin/confettysh
ENTRYPOINT [ "confettysh" ]
