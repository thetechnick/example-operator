FROM scratch

WORKDIR /
COPY passwd /etc/passwd
COPY example-operator-manager /

USER "noroot"

ENTRYPOINT ["/example-operator-manager"]
