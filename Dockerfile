FROM scratch
ADD ca-certificates.crt /etc/ssl/certs/

ADD wsproxy /
CMD ["/wsproxy"]
